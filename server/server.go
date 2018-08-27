package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	"github.com/hashicorp/raft-boltdb"
	"github.com/makersu/go-hashicorp-raft-http/config"
	"github.com/makersu/go-hashicorp-raft-http/fsm"
)

const (
	retainSnapshotCount = 2
	raftTimeout         = 10 * time.Second
)

type ClusterNode struct {
	RaftDir  string
	RaftAddr string

	raft *raft.Raft // The RAFT consensus mechanism

	fsm *fsm.NodeFSM // The FSM to manage replicated state machine

	Logger *log.Logger
}

func NewClusterNode(nodeConfig *config.ClusterNodeConfig) *ClusterNode {

	clusterNode := &ClusterNode{
		RaftDir:  nodeConfig.RaftDir,
		RaftAddr: nodeConfig.RaftAddr,
		Logger:   log.New(os.Stderr, "[ClusterNode] ", log.LstdFlags),
	}

	if err := os.MkdirAll(nodeConfig.RaftDir, 0700); err != nil {
		clusterNode.Logger.Println(err.Error())
		panic(err.Error())
	}

	if err := clusterNode.NewRaftNode(nodeConfig.IsBootstrap(), nodeConfig.NodeID); err != nil {
		clusterNode.Logger.Println(err.Error())
		panic(err.Error())
	}

	return clusterNode
}

func newRaftConfig(localNodeID string) *raft.Config {
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(localNodeID)
	raftConfig.SnapshotThreshold = 10
	return raftConfig
}

// Refactoring: rename
func (node *ClusterNode) NewRaftNode(bootstrap bool, localNodeID string) error {

	raftConfig := newRaftConfig(localNodeID)

	node.fsm = fsm.NewNodeFSM() // refactoring?

	transport, err := NewRaftTCPTransport(node.RaftAddr, os.Stderr)
	if err != nil {
		return err
	}

	logStore, err := raftboltdb.NewBoltStore(filepath.Join(node.RaftDir, "raft-log.db"))
	if err != nil {
		return err
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(node.RaftDir, "raft-stable.db"))
	if err != nil {
		return err
	}

	snapshotStore, err := raft.NewFileSnapshotStore(node.RaftDir, 2, os.Stderr)
	if err != nil {
		return err
	}

	// func NewRaft(conf *Config, fsm FSM, logs LogStore, stable StableStore, snaps SnapshotStore, trans Transport) (*Raft, error) {
	raftNode, err := raft.NewRaft(raftConfig, node.fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return err
	}

	node.raft = raftNode

	if bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raftConfig.LocalID,
					Address: transport.LocalAddr(),
				},
			},
		}
		node.raft.BootstrapCluster(configuration)
	}

	return nil
}

func NewRaftTCPTransport(raftBindAddr string, log io.Writer) (*raft.NetworkTransport, error) {

	address, err := net.ResolveTCPAddr("tcp", raftBindAddr)
	if err != nil {
		return nil, err
	}

	transport, err := raft.NewTCPTransport(address.String(), address, 3, 10*time.Second, log)
	if err != nil {
		return nil, err
	}

	return transport, nil
}

// TODO: http request retry?
func (node *ClusterNode) RequestJoinCluster(joinAddr, raftAddr, nodeID string) error {
	body, err := json.Marshal(map[string]string{"addr": raftAddr, "id": nodeID})
	if err != nil {
		return err
	}
	// node.Logger.Println(fmt.Sprintf("http://%s/join", joinAddr))
	resp, err := http.Post(fmt.Sprintf("http://%s/join", joinAddr), "application-type/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// Join joins a node, identified by nodeID and located at addr, to this store.
// The node must be ready to respond to Raft communications at that address.
func (node *ClusterNode) JoinClusterNode(nodeID, addr string) error {
	node.Logger.Printf("received join request for remote node %s at %s", nodeID, addr)

	configFuture := node.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		node.Logger.Printf("failed to get raft configuration: %v", err)
		return err
	}

	for _, srv := range configFuture.Configuration().Servers {
		// If a node already exists with either the joining node's ID or address,
		// that node may need to be removed from the config first.
		if srv.ID == raft.ServerID(nodeID) || srv.Address == raft.ServerAddress(addr) {
			// However if *both* the ID and the address are the same, then nothing -- not even
			// a join operation -- is needed.
			if srv.Address == raft.ServerAddress(addr) && srv.ID == raft.ServerID(nodeID) {
				node.Logger.Printf("node %s at %s already member of cluster, ignoring join request", nodeID, addr)
				return nil
			}

			future := node.raft.RemoveServer(srv.ID, 0, 0)
			if err := future.Error(); err != nil {
				return fmt.Errorf("error removing existing node %s at %s: %s", nodeID, addr, err)
			}
		}
	}

	f := node.raft.AddVoter(raft.ServerID(nodeID), raft.ServerAddress(addr), 0, 0)
	if f.Error() != nil {
		return f.Error()
	}
	node.Logger.Printf("Node \" %s \" at %s joined successfully", nodeID, addr)
	return nil
}

// LeaveClusterNode remove from Cluster
func (node *ClusterNode) LeaveClusterNode(nodeID, addr string) error {
	node.Logger.Printf("received leave request for remote node %s at %s", nodeID, addr)

	configFuture := node.raft.GetConfiguration()

	if err := configFuture.Error(); err != nil {
		node.Logger.Printf("failed to get raft configuration")
		return err
	}

	for _, server := range configFuture.Configuration().Servers {
		if server.ID == raft.ServerID(nodeID) {
			f := node.raft.RemoveServer(server.ID, 0, 0)
			if err := f.Error(); err != nil {
				node.Logger.Printf("failed to remove server %s", nodeID)
				return err
			}

			node.Logger.Printf("node %s leaved successfully", nodeID)
			return nil
		}
	}

	node.Logger.Printf("node %s not exists in raft group", nodeID)

	return nil

}

// SnapshotClusterNode handle ClusterNode snapshost mannually
func (node *ClusterNode) SnapshotClusterNode() error {
	node.Logger.Printf("SnapshotClusterNode() doing snapshot mannually")
	future := node.raft.Snapshot()
	return future.Error()
}

// Set sets the value for the given key.
func (node *ClusterNode) Set(key, value string) error {
	node.Logger.Println("Set() clusterNode.raft.State():", node.raft.State())

	if node.raft.State() != raft.Leader {
		return fmt.Errorf("not leader") //?
	}

	cmd := &fsm.Command{
		Op:    "set",
		Key:   key,
		Value: value,
	}

	bytes, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	applyFuture := node.raft.Apply(bytes, raftTimeout)
	return applyFuture.Error()
}

// Get returns the value for the given key.
func (node *ClusterNode) Get(key string) (string, error) {
	return node.fsm.Get(key)
}

package fsm

/**
	// FSM provides an interface that can be implemented by
	// clients to make use of the replicated log.
	type FSM interface {
	// Apply log is invoked once a log entry is committed.
	// It returns a value which will be made available in the
	// ApplyFuture returned by Raft.Apply method if that
	// method was called on the same Raft node as the FSM.
	Apply(*Log) interface{}

	// Snapshot is used to support log compaction. This call should
	// return an FSMSnapshot which can be used to save a point-in-time
	// snapshot of the FSM. Apply and Snapshot are not called in multiple
	// threads, but Apply will be called concurrently with Persist. This means
	// the FSM should be implemented in a fashion that allows for concurrent
	// updates while a snapshot is happening.
	Snapshot() (FSMSnapshot, error)

	// Restore is used to restore an FSM from a snapshot. It is not called
	// concurrently with any other command. The FSM must discard all previous
	// state.
	Restore(io.ReadCloser) error
}
**/

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/hashicorp/raft"
)

type Command struct {
	Op    string `json:"op,omitempty"`
	Key   string `json:"key,omitempty"`
	Value string `json:"value,omitempty"`
}

// Node State Machine
type NodeFSM struct {
	mutex sync.Mutex

	// db *badger.DB
	// rbt *rbt.Tree
	fsmState map[string]string // The key-value store for the system.

	logger *log.Logger
}

func NewNodeFSM() *NodeFSM {

	return &NodeFSM{
		fsmState: make(map[string]string),
		logger:   log.New(os.Stderr, "[NodeFSM] ", log.LstdFlags),
	}
}

// Apply applies a Raft log entry to the key-value store.
func (fsm *NodeFSM) Apply(l *raft.Log) interface{} {
	var cmd Command
	if err := json.Unmarshal(l.Data, &cmd); err != nil {
		panic(fmt.Sprintf("failed to unmarshal command: %s", err.Error())) //?
	}

	// fsm.logger.Println("Apply() command", cmd)

	switch cmd.Op {
	case "set":
		return fsm.applySet(cmd.Key, cmd.Value)
	// case "delete":
	// return fsm.applyDelete(c.Key)
	default:
		panic(fmt.Sprintf("unrecognized command op: %s", cmd.Op))
	}
}

func (fsm *NodeFSM) applySet(key, value string) interface{} {
	fsm.logger.Printf("applySet() %s to %s\n", key, value) //?

	fsm.mutex.Lock()
	defer fsm.mutex.Unlock()

	fsm.fsmState[key] = value
	return nil
}

// Get returns the value for the given key.
func (fsm *NodeFSM) Get(key string) (string, error) {

	fsm.mutex.Lock()
	defer fsm.mutex.Unlock()

	return fsm.fsmState[key], nil //?
}

// Snapshot returns a snapshot of the key-value store.
func (fsm *NodeFSM) Snapshot() (raft.FSMSnapshot, error) {
	fsm.logger.Println("Snapshot()")

	fsm.mutex.Lock()
	defer fsm.mutex.Unlock()

	// Clone the map.
	store := make(map[string]string)
	for k, v := range fsm.fsmState {
		store[k] = v
	}
	return &NodeFSMSnapshot{snapshotState: store}, nil
}

// Restore restores an FSM from a snapshot.
func (fsm *NodeFSM) Restore(rc io.ReadCloser) error {
	fsm.logger.Println("Restore()")

	store := make(map[string]string)
	if err := json.NewDecoder(rc).Decode(&store); err != nil {
		return err
	}

	// Set the state from the snapshot, no lock required according to
	// Hashicorp docs.
	fsm.fsmState = store
	return nil
}

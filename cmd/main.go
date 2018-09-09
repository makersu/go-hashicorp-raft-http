package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/makersu/go-hashicorp-raft-http/config"
	"github.com/makersu/go-hashicorp-raft-http/server"
)

//
const (
	DefaultNodeID   = "matching1"
	DefaultRaftDir  = "./matching1"
	DefaultRaftAddr = ":7777"
	DefaultHTTPAddr = ":8887"
)

var (
	nodeID       string
	raftDir      string
	raftAddr     string
	httpAddr     string
	joinHttpAddr string
	// inmem    bool

)

func init() {
	flag.StringVar(&nodeID, "id", DefaultNodeID, "Set the Node ID")
	flag.StringVar(&raftDir, "raftDir", DefaultRaftDir, "Set the Raft data directory")
	flag.StringVar(&raftAddr, "raftAddr", DefaultRaftAddr, "Set the Raft bind address")
	flag.StringVar(&httpAddr, "httpAddr", DefaultHTTPAddr, "Set the HTTP bind address")
	flag.StringVar(&joinHttpAddr, "joinHttpAddr", "", "Set the HTTP address to request join cluster, if any")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <raft-data-path> \n", os.Args[0])
		flag.PrintDefaults()
	}
}

/**
go build -o bin/go-hashicorp-raft-http cmd/*
bin/go-hashicorp-raft-http -id matching1 -raftDir ./snaphosts/matching1 -raftAddr :7777  -httpAddr :8887
bin/go-hashicorp-raft-http -id matching2 -raftDir ./snaphosts/matching2 -raftAddr :7778  -httpAddr :8888 -joinHttpAddr :8887
bin/go-hashicorp-raft-http -id matching3 -raftDir ./snaphosts/matching3 -raftAddr :7779  -httpAddr :8889 -joinHttpAddr :8887
*/
func main() {
	flag.Parse()

	clusterNodeConfig := config.NewClusterNodeConfig(nodeID, raftDir, raftAddr, httpAddr, joinHttpAddr)

	node := server.NewClusterNode(clusterNodeConfig)

	// Request to join cluster leader node if not bootstrap
	if !clusterNodeConfig.IsBootstrap() {
		if err := node.RequestJoinCluster(clusterNodeConfig.JoinHttpAddr, clusterNodeConfig.RaftAddr, clusterNodeConfig.NodeID); err != nil {
			panic(err.Error())
		}
	}

	// Start http handler
	httpHandler := server.NewNodeHttpHandler(node, clusterNodeConfig.HttpAddr)
	if err := httpHandler.Start(); err != nil {
		panic(err.Error())
	}

	node.Logger.Println("Cluster Node Id \"", nodeID, "\" started successfully")

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Kill, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-terminate

	// snapshot when process terminated
	node.SnapshotClusterNode()
	node.Logger.Println("Cluster Node Id \"", nodeID, "\" exiting")

}

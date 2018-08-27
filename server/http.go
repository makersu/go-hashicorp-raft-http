package server

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
)

type NodeHttpHandler struct {
	httpBindAddr string
	listener     net.Listener
	clusterNode  *ClusterNode
	logger       *log.Logger
}

func NewNodeHttpHandler(node *ClusterNode, httpBindAddr string) *NodeHttpHandler {
	return &NodeHttpHandler{
		logger:       log.New(os.Stderr, "[NodeHttpHandler] ", log.LstdFlags),
		httpBindAddr: httpBindAddr,
		clusterNode:  node,
	}
}

// Start the httphandler
func (handler *NodeHttpHandler) Start() error {
	server := http.Server{
		Handler: handler,
	}

	ln, err := net.Listen("tcp", handler.httpBindAddr)
	if err != nil {
		return err
	}
	handler.listener = ln

	http.Handle("/", handler)

	go func() {
		err := server.Serve(handler.listener)
		if err != nil {
			log.Fatalf("HTTP serve: %s", err) //
		}
	}()

	return nil
}

// ServeHTTP allows Service to serve HTTP requests.
func (handler *NodeHttpHandler) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if strings.HasPrefix(req.URL.Path, "/key") {
		handler.handleKeyRequest(resp, req)
	} else if req.URL.Path == "/join" {
		handler.handleJoinClusterRequest(resp, req)
	} else if req.URL.Path == "/leave" {
		handler.handleLeaveClusterRequest(resp, req)
	} else if req.URL.Path == "/snapshot" {
		handler.handleSnapshotRequest(resp, req)
	} else {
		resp.WriteHeader(http.StatusNotFound)
	}
}

func (handler *NodeHttpHandler) handleJoinClusterRequest(resp http.ResponseWriter, req *http.Request) {

	m := map[string]string{}
	if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(m) != 2 {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	remoteAddr, ok := m["addr"]
	if !ok {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	nodeID, ok := m["id"]
	if !ok {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := handler.clusterNode.JoinClusterNode(nodeID, remoteAddr); err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (handler *NodeHttpHandler) handleLeaveClusterRequest(resp http.ResponseWriter, req *http.Request) {

	m := map[string]string{}
	if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(m) != 2 {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	remoteAddr, ok := m["addr"]
	if !ok {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	nodeID, ok := m["id"]
	if !ok {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := handler.clusterNode.LeaveClusterNode(nodeID, remoteAddr); err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (handler *NodeHttpHandler) handleKeyRequest(resp http.ResponseWriter, req *http.Request) {

	getKey := func() string {
		parts := strings.Split(req.URL.Path, "/")
		if len(parts) != 3 {
			return ""
		}
		return parts[2]
	}

	switch req.Method {
	case "GET":
		k := getKey()
		if k == "" {
			resp.WriteHeader(http.StatusBadRequest)
		}
		v, err := handler.clusterNode.Get(k)
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		bytes, err := json.Marshal(map[string]string{k: v})
		if err != nil {
			resp.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp.Write(bytes)

	case "POST":
		// Read the value from the POST body.
		m := map[string]string{}
		if err := json.NewDecoder(req.Body).Decode(&m); err != nil {
			resp.WriteHeader(http.StatusBadRequest)
			return
		}

		for k, v := range m {
			if err := handler.clusterNode.Set(k, v); err != nil {
				resp.WriteHeader(http.StatusInternalServerError)
				return
			}
		}

	// case "DELETE":

	default:
		resp.WriteHeader(http.StatusMethodNotAllowed)
	}
	return
}

func (handler *NodeHttpHandler) handleSnapshotRequest(resp http.ResponseWriter, req *http.Request) {
	if err := handler.clusterNode.SnapshotClusterNode(); err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}
}

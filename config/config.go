package config

type ClusterNodeConfig struct {
	NodeID       string
	RaftDir      string
	RaftAddr     string
	HttpAddr     string
	JoinHttpAddr string
	Desc         string
}

func NewClusterNodeConfig(nodeID, raftDir, raftAddr, httpAddr, joinHttpAddr string) *ClusterNodeConfig {
	return &ClusterNodeConfig{
		NodeID:       nodeID,
		RaftDir:      raftDir,
		RaftAddr:     raftAddr,
		HttpAddr:     httpAddr,
		JoinHttpAddr: joinHttpAddr,
		Desc:         "",
	}
}

func (clusterNodeConfig *ClusterNodeConfig) IsBootstrap() bool {
	return clusterNodeConfig.JoinHttpAddr == ""
}

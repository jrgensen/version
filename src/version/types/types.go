package types

import (
	"github.com/docker/docker/api/types"
)

type Container struct {
	ServiceName          string
	ContainerNames       []string
	Image                types.ImageSummary
	LatestRegistryLabels map[string]string
	Created              int64
	Labels               map[string]string
	State                string
	Status               string
	SizeRootFs           int64
	DeployState          string
}

type Project struct {
	ProjectName string
	Containers  []Container
}

type Message struct {
	//Author    string `json:"author"`
	//Body      string `json:"body"`
	//Container types.Container
	Name       string
	Projects   map[string]Project
	Containers []Container
}

func (self *Message) String() string {
	return self.Name
}

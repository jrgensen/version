package docker

import (
	"context"
	"encoding/json"
	"fmt"
	dockertypes "github.com/docker/docker/api/types"
	//	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"io"
	"log"
	"net/http"
	"strings"
	"version/types"
)

type Api struct {
	client     *client.Client
	containers []*types.Container
	host       *types.Message
	recipient  dockerEventRecipient
	registry   *registry
}

type Response struct {
	Projects   map[string]types.Project
	Containers []types.Container
}

/*
type EventMessage struct {
    // Deprecated information from JSONMessage.
    // With data only in container events.
    Status string `json:"status,omitempty"`
    ID     string `json:"id,omitempty"`
    From   string `json:"from,omitempty"`

    Type   string
    Action string
    Actor  type Actor struct {
        ID         string
        Attributes map[string]string
    }
    // Engine events are local scope. Cluster events are swarm scope.
    Scope string `json:"scope,omitempty"`

    Time     int64 `json:"time,omitempty"`
    TimeNano int64 `json:"timeNano,omitempty"`
}//*/
type dockerEventRecipient interface {
	SendAll(*types.Message)
}

func NewApiClient(recipient dockerEventRecipient, reg registry) *Api {
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}
	containers := []*types.Container{}
	host := &types.Message{Name: "init"}
	api := &Api{cli, containers, host, recipient, &reg}
	api.reloadCache()
	go api.listenEvents()
	return api
}

func (api *Api) listenEvents() {
	messages, errs := api.client.Events(context.Background(), dockertypes.EventsOptions{})

	for {
		select {
		case err := <-errs:
			if err != nil && err != io.EOF {
				log.Fatal(err)
			}
			return
		case e := <-messages:
			fmt.Println("got event", e)
			api.reloadCache()
		}
	}
}
func (api *Api) fetchImages() (map[string]dockertypes.ImageSummary, error) {

	images := make(map[string]dockertypes.ImageSummary)

	fmt.Println("fetching images")
	imageSummaries, err := api.client.ImageList(context.Background(), dockertypes.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	for _, imageSummary := range imageSummaries {
		images[imageSummary.ID] = imageSummary
	}
	return images, nil
}

func (api *Api) reloadCache() {
	host := types.Message{
		Name:     "localhost",
		Projects: make(map[string]types.Project),
	}
	images, err := api.fetchImages()
	if err != nil {
		panic(err)
	}

	fmt.Println("fetching containers")
	containers, err := api.client.ContainerList(context.Background(), dockertypes.ContainerListOptions{All: true})
	for _, container := range containers {
		image := images[container.ImageID]
		var repo string
		if len(image.RepoTags) > 0 {
			repo = strings.Split(image.RepoTags[0], ":")[0]
		}
		fmt.Println("repo name", repo)

		c := types.Container{
			ContainerNames:       container.Names,
			Image:                image,
			LatestRegistryLabels: api.registry.Labels(repo, "latest"),
			Created:              container.Created,
			Labels:               container.Labels,
			State:                container.State,
			Status:               container.Status,
			SizeRootFs:           container.SizeRootFs,
		}
		if projectName, ok := c.Labels["com.docker.compose.project"]; ok {
			c.ServiceName = c.Labels["com.docker.compose.service"]
			project := host.Projects[projectName]
			project.ProjectName = projectName
			project.Containers = append(project.Containers, c)

			host.Projects[projectName] = project
		} else {
			host.Containers = append(host.Containers, c)
		}
		//js, _ := json.Marshal(container)
		//fmt.Printf("%s %s\n", container.ID[:10], js)
	}
	api.recipient.SendAll(&host)
	api.host = &host
}

func (api *Api) Handler(w http.ResponseWriter, r *http.Request) {
	jsonstr, err := json.Marshal(api.host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonstr)
}

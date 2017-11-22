package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"io"
	"io/ioutil"
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
			jsonevent, _ := json.Marshal(e)
			fmt.Println("GOT EVENT", string(jsonevent))
			if e.Type == events.ContainerEventType {
				api.reloadCache()
			}
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

	fmt.Print("fetching containers from local docker engine... ")
	containers, err := api.client.ContainerList(context.Background(), dockertypes.ContainerListOptions{All: true})
	if err != nil {
		panic(err)
	}
	fmt.Printf("[found %d]\n", len(containers))
	for _, container := range containers {
		image := images[container.ImageID]
		var repo string
		if len(image.RepoTags) > 0 {
			repo = strings.Split(image.RepoTags[0], ":")[0]
		}
		//fmt.Println("repo name", repo)

		c := types.Container{
			ContainerNames:       container.Names,
			Image:                image,
			LatestRegistryLabels: api.registry.Labels(repo, "latest"),
			Created:              container.Created,
			Labels:               container.Labels,
			State:                container.State,
			Status:               container.Status,
			SizeRootFs:           container.SizeRootFs,
			DeployState:          "unknown",
		}
		if tagLabelLocal, ok := c.Labels["TAG_VERSION"]; ok {
			if tagLabelRegistry, ok := c.LatestRegistryLabels["TAG_VERSION"]; ok {
				c.DeployState = "uptodate"
				if tagLabelLocal != tagLabelRegistry {
					c.DeployState = "behind"
				}
			}
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

func (api *Api) Login(registry string, un string, pw string) {
	auth := dockertypes.AuthConfig{
		Username:      un,
		Password:      pw,
		ServerAddress: registry,
	}
	authResp, err := api.client.RegistryLogin(context.Background(), auth)

	js, _ := json.Marshal(authResp)
	fmt.Printf("%s\n", js)
	fmt.Printf("%s %s\n", authResp, err)

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

func (api *Api) PullHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("login to registry")
	api.Login("registry.blackwoodseven.com", "kj", "12345678")
	fmt.Println("pulling images")
	ref := r.URL.Query().Get("ref")

	authConfig := dockertypes.AuthConfig{
		Username: "kj",
		Password: "12345678",
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		panic(err)
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)

	readCloser, err := api.client.ImagePull(context.Background(), ref, dockertypes.ImagePullOptions{
		//        All: true,
		RegistryAuth: authStr,
	})
	if err != nil {
		fmt.Println("%s", err)
		w.Write([]byte(fmt.Sprintf("Got error: %v", err)))
		return
	}
	//defer readCloser.Close()

	body, err := ioutil.ReadAll(readCloser)
	if err != nil {
		fmt.Println("%s", err)
		w.Write([]byte(fmt.Sprintf("Got another error: %v", err)))
		return
	}
	w.Write(body)
}

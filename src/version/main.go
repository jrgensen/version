package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"log"
	"net/http"
	//"version/ws"
)

type Container struct {
	ServiceName    string
	ContainerNames []string
	Image          types.ImageSummary
	Created        int64
	Labels         map[string]string
	State          string
	Status         string
	SizeRootFs     int64
}

type Project struct {
	ProjectName string
	Containers  []Container
}

type Response struct {
	Projects   map[string]Project
	Containers []Container
}

func handler(w http.ResponseWriter, r *http.Request) {

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	images := make(map[string]types.ImageSummary)
	imageSummaries, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		panic(err)
	}
	for _, imageSummary := range imageSummaries {
		images[imageSummary.ID] = imageSummary
	}

	resp := Response{
		Projects: make(map[string]Project),
	}
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{Size: true})
	for _, container := range containers {
		c := Container{
			ContainerNames: container.Names,
			Image:          images[container.ImageID],
			Created:        container.Created,
			Labels:         container.Labels,
			State:          container.State,
			Status:         container.Status,
			SizeRootFs:     container.SizeRootFs,
		}
		if projectName, ok := c.Labels["com.docker.compose.project"]; ok {
			c.ServiceName = c.Labels["com.docker.compose.service"]
			project := resp.Projects[projectName]
			project.ProjectName = projectName
			project.Containers = append(project.Containers, c)

			resp.Projects[projectName] = project
		} else {
			resp.Containers = append(resp.Containers, c)
		}
		js, _ := json.Marshal(container)
		fmt.Printf("%s %s\n", container.ID[:10], js)
	}
	jsonstr, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonstr)
}
func main() {
	log.SetFlags(log.Lshortfile)

	//server := ws.NewServer("/entry")
	//go server.Listen()

	// static files
	http.HandleFunc("/ps", handler)
	http.Handle("/", http.FileServer(http.Dir("webroot")))

	log.Fatal(http.ListenAndServe(":80", nil))
}

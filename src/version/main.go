package main

import (
	"context"
	"encoding/json"
	//	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"log"
	"net/http"
	//"version/ws"
)

type Container struct {
	Names   []string
	Image   string
	Created int64
	Labels  map[string]string
	State   string
	Status  string
}
type Response struct {
	Containers []Container
}

func handler(w http.ResponseWriter, r *http.Request) {

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		panic(err)
	}

	cs := []Container{}
	for _, container := range containers {
		cs = append(cs, Container{
			Names:   container.Names,
			Image:   container.Image,
			Created: container.Created,
			Labels:  container.Labels,
			State:   container.State,
			Status:  container.Status,
		})
		//fmt.Printf("%s %s\n", container.ID[:10], "")
	}
	jsonstr, err := json.Marshal(cs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonstr)

	//fmt.Fprintf(w, "%s", jsonstr)
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

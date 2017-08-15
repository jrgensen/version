package main

import (
	"flag"
	"fmt"
	//	"github.com/docker/distribution/digest"
	//	"github.com/docker/distribution/manifest"
	//	"github.com/docker/libtrust"
	//	"github.com/heroku/docker-registry-client/registry"
	"log"
	"net/http"
	"strings"
	"version/docker"
	"version/ws"
)

/*
func listen(serv *ws.Server, dockerapi *docker.Api) {
	for {
		container := <-dockerapi.Ch
		msg := ws.Message{Author: "Go dispatcher", Body: "krop", Container: *container}
		serv.SendAll(&msg)
	}
}*/

func main() {
	//	url := "https://registry.blackwoodseven.com/"
	username := "kj"       // anonymous
	password := "12345678" // anonymous
	r := docker.NewRegistry("registry.blackwoodseven.com", username, password)
	//	r.Login()
	/*
		hub, err := registry.New(url, username, password)
		if err != nil {
			panic(err)
		}
		repositories, err := hub.Repositories()
		if err != nil {
			panic(err)
		}
		fmt.Println("repos", repositories)
		manifest, err := hub.ManifestV2("blackwoodseven/version", "latest")
		if err != nil {
			panic(err)
		}
		fmt.Println("manifest", manifest)
	*/
	port := flag.Int("port", 80, "listening on port")
	hosts := flag.String("hosts", "localhost", "hosts to include")
	flag.Parse()
	fmt.Println("starting server on port", *port)
	for _, host := range strings.Split(*hosts, ";") {
		fmt.Println("listening to:", host)
	}
	log.SetFlags(log.Lshortfile)

	// websocket server
	server := ws.NewServer("/ws")
	go server.Listen()

	dockerapi := docker.NewApiClient(server, *r)
	//go listen(server, dockerapi)

	// static files
	http.HandleFunc("/ps", dockerapi.Handler)
	http.Handle("/", http.FileServer(http.Dir("/var/www")))

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

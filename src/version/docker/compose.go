package docker

import (
	"fmt"
	"net/http"

	"golang.org/x/net/context"

	"github.com/docker/libcompose/docker"
	"github.com/docker/libcompose/docker/ctx"
	"github.com/docker/libcompose/project"
	"github.com/docker/libcompose/project/options"
)

func (api *Api) ComposeHandler(w http.ResponseWriter, r *http.Request) {
	project, err := docker.NewProject(&ctx.Context{
		Context: project.Context{
			ComposeFiles: []string{"docker-compose.yml"},
			ProjectName:  "my-compose",
		},
		//ConfigFile: []string{"config.json"},
	}, nil)

	if err != nil {
		fmt.Println("%s", err)
		w.Write([]byte(fmt.Sprintf("Got error: %v", err)))
		return
		//log.Fatal(err)
	}

	err = project.Pull(context.Background())

	if err != nil {
		fmt.Println("%s", err)
		//w.Write([]byte(fmt.Sprintf("Got error: %v", err)))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = project.Up(context.Background(), options.Up{})

	if err != nil {
		fmt.Println("%s", err)
		//w.Write([]byte(fmt.Sprintf("Got error: %v", err)))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	/*jsonstr, err := json.Marshal(api.host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}*/
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("ok"))
}

/*
func main() {
    project, err := docker.NewProject(&ctx.Context{
        Context: project.Context{
            ComposeFiles: []string{"docker-compose.yml"},
            ProjectName:  "my-compose",
        },
    }, nil)

    if err != nil {
        log.Fatal(err)
    }

    err = project.Up(context.Background(), options.Up{})

    if err != nil {
        log.Fatal(err)
    }
}*/

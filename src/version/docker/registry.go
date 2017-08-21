package docker

import (
	//	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type token struct {
	Token     string    `json:"token"`
	ExpiresIn int       `json:"expires_in"`
	IssuedAt  time.Time `json:"issued_at"`
}
type manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	Name          string `json:"name"`
	Tag           string `json:"tag"`
	Architecture  string `json:"architecture"`
	FsLayers      []struct {
		BlobSum string `json:"blobSum"`
	} `json:"fsLayers"`
	History []struct {
		V1Compatibility string `json:"v1Compatibility"`
	} `json:"history"`
	Signatures []struct {
		Header struct {
			Jwk struct {
				Crv string `json:"crv"`
				Kid string `json:"kid"`
				Kty string `json:"kty"`
				X   string `json:"x"`
				Y   string `json:"y"`
			} `json:"jwk"`
			Alg string `json:"alg"`
		} `json:"header"`
		Signature string `json:"signature"`
		Protected string `json:"protected"`
	} `json:"signatures"`
}
type catalog struct {
	Repositories []string `json:"repositories"`
}

type v1Config struct {
	Hostname     string              `json:"Hostname"`
	Domainname   string              `json:"Domainname"`
	User         string              `json:"User"`
	AttachStdin  bool                `json:"AttachStdin"`
	AttachStdout bool                `json:"AttachStdout"`
	AttachStderr bool                `json:"AttachStderr"`
	ExposedPorts map[string]struct{} `json:"ExposedPorts"`
	Tty          bool                `json:"Tty"`
	OpenStdin    bool                `json:"OpenStdin"`
	StdinOnce    bool                `json:"StdinOnce"`
	Env          []string            `json:"Env"`
	Cmd          []string            `json:"Cmd"`
	ArgsEscaped  bool                `json:"ArgsEscaped"`
	Image        string              `json:"Image"`
	Volumes      interface{}         `json:"Volumes"`
	WorkingDir   string              `json:"WorkingDir"`
	Entrypoint   interface{}         `json:"Entrypoint"`
	OnBuild      []interface{}       `json:"OnBuild"`
	Labels       map[string]string   `json:"Labels"`
}
type v1Compatibility struct {
	Architecture    string    `json:"architecture"`
	Config          *v1Config `json:"config"`
	Container       string    `json:"container"`
	ContainerConfig *v1Config `json:"container_config"`
	Created         time.Time `json:"created"`
	DockerVersion   string    `json:"docker_version"`
	ID              string    `json:"id"`
	Os              string    `json:"os"`
	Parent          string    `json:"parent"`
	Throwaway       bool      `json:"throwaway"`
}

type registry struct {
	host      string
	user      string
	pass      string
	tokens    map[string]token
	manifests map[string]manifest
	sync.Mutex
}

func NewRegistry(host string, user string, pass string) *registry {
	reg := &registry{host: host, user: user, pass: pass, tokens: make(map[string]token), manifests: make(map[string]manifest)}
	go reg.refreshManifestTimer(180)
	return reg
}

func (r *registry) refreshToken(scope string) error {
	url := fmt.Sprintf("https://%[1]s/v2/token?service=%[1]s&scope=%s", r.host, scope)

	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(r.user, r.pass)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	token := &token{}
	err = json.NewDecoder(res.Body).Decode(token)
	if err != nil {
		fmt.Println("GOT AN ERROR", err)
		// if registry auth service is in limbo, we might not get valid json
		return err
	}
	r.tokens[scope] = *token
	//jwt, _ := base64.StdEncoding.DecodeString(strings.Split(token.Token, ".")[1])
	//fmt.Printf("Token refreshed: %s\n", jwt)
	return nil
}
func (r *registry) refreshTokenIfNeeded(scope string) error {
	// FIXME only renew token if necessary
	return r.refreshToken(scope)
}

func (r *registry) Labels(repo string, tag string) map[string]string {
    err := r.refreshManifestIfNeeded(repo, tag)
    if err != nil {
        fmt.Println(err)
    }

	v1 := &v1Compatibility{
		Config:          &v1Config{Labels: make(map[string]string)},
		ContainerConfig: &v1Config{Labels: make(map[string]string)},
	}
	if manifest, ok := r.manifests[fmt.Sprintf("%s:%s", repo, tag)]; ok {
		if len(manifest.History) == 0 {
            fmt.Printf("No history found (%s:%s)\n", repo, tag)
			return nil
		}
		json.Unmarshal([]byte(manifest.History[0].V1Compatibility), v1)
		return v1.Config.Labels
	}
	return nil
}
func (r *registry) refreshManifestIfNeeded(repo string, tag string) error {
	if _, ok := r.manifests[fmt.Sprintf("%s:%s", repo, tag)]; ok {
		fmt.Printf("manifest cached: %s:%s\n", repo, tag)
		return nil
	}
	return r.refreshManifest(repo, tag)
}

func (r *registry) refreshManifest(repo string, tag string) error {
	var host, scope string
	if len(strings.Split(repo, "/")) == 3 {
		hs := strings.SplitN(repo, "/", 2)
		host, scope = hs[0], hs[1]
	} else {
		host, scope = "index.docker.io", repo
	}
	if host != r.host {
		// we only support login to one host
		return nil
	}
	r.refreshTokenIfNeeded(fmt.Sprintf("repository:%s:pull", scope))

	url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", host, scope, tag)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", r.tokens[fmt.Sprintf("repository:%s:pull", scope)].Token))
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		fmt.Println(repo, res.Status)
		return nil
	}

	mani := &manifest{}
	err = json.NewDecoder(res.Body).Decode(mani)
	if err != nil {
		return err
	}
	fmt.Printf("Refreshed manifest: %s:%s\n", repo, tag)
	r.Lock()
	defer r.Unlock()

	r.manifests[fmt.Sprintf("%s:%s", repo, tag)] = *mani
	return nil
}

func (r *registry) refreshManifestTimer(seconds time.Duration) {
	for {
		fmt.Println("TIMER REFRESH")
		for repotag, _ := range r.manifests {
			rt := strings.Split(repotag, ":")
			repo, tag := rt[0], rt[1]
			r.refreshManifest(repo, tag)
		}
		time.Sleep(seconds * time.Second)
	}
}

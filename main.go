package main

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/hcl"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
)

type IncomingWebhook struct {
	Repository      string   `json:"repository"`
	Namespace       string   `json:"namespace"`
	Name            string   `json:"name"`
	DockerURL       string   `json:"docker_url"`
	Homepage        string   `json:"homepage"`
	Visibility      string   `json:"visibility"`
	BuildID         string   `json:"build_id"`
	DockerTags      []string `json:"docker_tags"`
	TriggerKind     string   `json:"trigger_kind"`
	TriggerID       string   `json:"trigger_id"`
	TriggerMetadata struct {
		DefaultBranch string `json:"default_branch"`
		Ref           string `json:"ref"`
		Commit        string `json:"commit"`
		CommitInfo    struct {
			URL     string `json:"url"`
			Message string `json:"message"`
			Date    int64  `json:"date,string"`
			Author  struct {
				Username  string `json:"username"`
				URL       string `json:"url"`
				AvatarURL string `json:"avatar_url"`
			} `json:"author"`
			Committer struct {
				Username  string `json:"username"`
				URL       string `json:"url"`
				AvatarURL string `json:"avatar_url"`
			} `json:"committer"`
		} `json:"commit_info"`
	} `json:"trigger_metadata"`
}

type Service struct {
	Name       string `hcl:",key"`
	Repository string `hcl:"repository"`
	Cmd        string `hcl:"cmd"`
	Conditions string `hcl:"conditions"`
}

type Config struct {
	Services []Service `hcl:"service"`
}

func deploy(svc Service, ref string) {
	fmt.Printf("[Deploying] %s from %s\n", svc.Name, ref)

	content := []byte(ref)

	pattern := regexp.MustCompile(svc.Conditions)

	template := []byte(svc.Cmd)

	cmd := []byte{}

	for _, submatches := range pattern.FindAllSubmatchIndex(content, -1) {
		cmd = pattern.Expand(cmd, template, content, submatches)
	}

	fmt.Printf("[Executing Shell] %s\n", cmd)
	out, err := exec.Command("/bin/sh", "-c", string(cmd)).CombinedOutput()
	if err != nil {
		fmt.Printf("[ERROR] %s\n", err)
		return
	}
	fmt.Printf("%s\n", out)
}

func main() {
	var PORT string

	if os.Getenv("PORT") != "" {
		PORT = os.Getenv("PORT")
	} else {
		PORT = "2000"
	}

	if len(os.Args) < 2 {
		log.Fatal("Pass one argument to this program with the path to config.hcl")
	}

	configPath := os.Args[1]

	configContents, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	var config Config
	decodeErr := hcl.Unmarshal(configContents, &config)
	if decodeErr != nil {
		log.Fatal(decodeErr)
	}

	fmt.Printf("[Config loaded] %s\n", configPath)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _r *http.Request) {
		fmt.Fprintf(w, "ok")
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var payload IncomingWebhook
		err := decoder.Decode(&payload)
		if err != nil {
			fmt.Printf("%+v\n", err)
		}

		if os.Getenv("DEBUG") != "" {
			fmt.Printf("[DEBUG PAYLOAD] %+v\n", payload)
		}

		for _, svc := range config.Services {
			re := regexp.MustCompile(svc.Conditions)
			if svc.Repository == payload.Repository && len(re.Find([]byte(payload.TriggerMetadata.Ref))) != 0 {
				deploy(svc, payload.TriggerMetadata.Ref)
			}
		}

		fmt.Fprintf(w, "ok")
	})

	http.ListenAndServe(":"+PORT, nil)
}
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
	"time"
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
			Date    string `json:"date"`
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

func timestamp() string {
	t := time.Now()
	return t.Format("2006-01-02 15:04:05")
}

func deploy(svc Service, ref string) {
	fmt.Printf("[%s Deploying] %s from %s\n", timestamp(), svc.Name, ref)

	content := []byte(ref)

	pattern := regexp.MustCompile(svc.Conditions)

	template := []byte(svc.Cmd)

	cmd := []byte{}

	for _, submatches := range pattern.FindAllSubmatchIndex(content, -1) {
		cmd = pattern.Expand(cmd, template, content, submatches)
	}

	fmt.Printf("[%s Executing Shell] %s\n", timestamp(), cmd)
	out, err := exec.Command("/bin/sh", "-c", string(cmd)).CombinedOutput()
	if err != nil {
		fmt.Printf("[%s ERROR] [exec shell] %s\n", timestamp(), err)
		return
	}

	fmt.Printf("[%s Shell Output (begin)]\n", timestamp())
	fmt.Printf("%s\n", out)
	fmt.Printf("[%s Shell Output (end)]\n", timestamp())
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

	fmt.Printf("[%s Config loaded] %s\n", timestamp(), configPath)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _r *http.Request) {
		fmt.Fprintf(w, "ok")
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			fmt.Printf("[%s ERROR Read Req Body] %+v\n", timestamp(), err)
			fmt.Fprintf(w, "error")
			return
		}

		if len(body) == 0 {
			fmt.Printf("[%s ERROR Empty Req Body] skipping\n", timestamp())
			fmt.Fprintf(w, "error")
			return
		}

		if os.Getenv("DEBUG") != "" {
			fmt.Printf("[%s DEBUG Raw JSON Payload] %s\n", timestamp(), string(body))
		}

		var payload IncomingWebhook

		err = json.Unmarshal(body, &payload)

		if os.Getenv("DEBUG") != "" {
			fmt.Printf("[%s DEBUG Decoded Payload] %+v\n", timestamp(), payload)
		}

		if err != nil {
			fmt.Printf("[%s ERROR] [decode payload] %+v\n", timestamp(), err)
			fmt.Fprintf(w, "error")
			return
		}

		conditionsFound := 0
		for _, svc := range config.Services {
			if os.Getenv("DEBUG") != "" {
				fmt.Printf("[%s Checking Condition] %+v\n", timestamp(), svc.Conditions)
			}
			re := regexp.MustCompile(svc.Conditions)
			if svc.Repository == payload.Repository && len(re.Find([]byte(payload.TriggerMetadata.Ref))) != 0 {
				conditionsFound++
				deploy(svc, payload.TriggerMetadata.Ref)
			}
		}
		if os.Getenv("DEBUG") != "" && conditionsFound == 0 {
			fmt.Printf("[%s 0 Conditions Found]\n", timestamp())
		}

		fmt.Fprintf(w, "ok")
	})

	http.ListenAndServe(":"+PORT, nil)
}

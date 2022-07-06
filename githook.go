package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
)

type repoRule struct {
	URL                 string   `json:"url"`
	Branch              string   `json:"branch"`
	DeploymentScript    string   `json:"deployment_script"`
	DeploymentArguments []string `json:"deplyoyment_arguments"`
}

var webhookRules = make([]repoRule, 0, 0)

func main() {
	port := flag.Int("port", 8080, "Listening port for GitHub webhooks")
	path := flag.String("path", "/only_you_can_bring_me", "The path to listen to for the hook")
	rules := flag.String("rules", "", "JSON file that defines paths, repos and the branches that should be deployed in response to webhooks")
	flag.Parse()

	if *rules == "" {
		log.Fatalf("Must provide a JSON rules files. Use --help for list of commands.")
	}
	loadRules(*rules)

	http.HandleFunc(*path, hookHandler)

	hostAddress := fmt.Sprintf(":%d", *port)
	log.Printf("Starting HTTP (insecure) server on port %d", *port)
	err := http.ListenAndServe(hostAddress, nil)
	if err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}
}

func loadRules(rulesPath string) {
	// let's read the rules
	rulesFile, err := os.Open(rulesPath)
	if err != nil {
		log.Fatalf("Unable to open rules file: %v", err)
	}
	// sanity-check. is this file less than 8KB?
	fileInfo, err := rulesFile.Stat()
	if err != nil {
		log.Fatalf("Error getting info about rules file: %v", err)
	}
	if fileInfo.Size() > 8192 {
		log.Fatalf("Rules file is too big. Must be less than 8KB. This file is %d bytes.", fileInfo.Size())
	}
	fileBytes, err := ioutil.ReadAll(rulesFile)
	if err != nil {
		log.Fatalf("Unable to read data from rules file: %v", err)
	}
	err = json.Unmarshal(fileBytes, &webhookRules)
	if err != nil {
		log.Fatalf("Error unmarshalling JSON from rules file: %v", err)
	}

	log.Printf("Loaded rules:\n%v", webhookRules)
}

func hookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Must be a POST", http.StatusBadRequest)
		return
	}

	hookPayload := make(map[string]interface{})
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&hookPayload)
	if err != nil {
		http.Error(w, fmt.Sprintf("Unable to decode JSON: %v", err), http.StatusBadRequest)
		return
	}

	// get the repository of this event
	if hookPayload["repository"] == nil {
		fmt.Fprint(w, "Not a valid payload. Probably the initial webhook from GitHub")
		return
	}
	repoMap := hookPayload["repository"].(map[string]interface{})
	url := repoMap["url"].(string)
	// get the branch of this event
	branch := hookPayload["ref"].(string)

	// check if we have a rule for this event
	for i := range webhookRules {
		if url == webhookRules[i].URL &&
			branch == webhookRules[i].Branch {
			// perform a deploy
			msg := fmt.Sprintf("Deploying %s\n\n", hookPayload["after"].(string))
			w.Write([]byte(msg))
			output, err := deploy(&webhookRules[i])

			if err != nil {
				w.Write([]byte(fmt.Sprintf("Error deploying: %v", err)))
				continue
			}
			w.Write(output)
		}
	}
}

func deploy(rule *repoRule) ([]byte, error) {
	c := exec.Command(rule.DeploymentScript, rule.DeploymentArguments...)
	c.Dir = path.Dir(rule.DeploymentScript)
	return c.CombinedOutput()
}

package main

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

const githubAPI = "https://api.github.com"

var githubToken string
var githubRepository string
var maxWorkers int = 10

type artifact struct {
	ID   int
	Name string
}

type githubResponse struct {
	Artifacts []*artifact
}

var cli = &http.Client{
	Timeout: 60 * time.Second,
}

func main() {
	// load dotenv
	godotenv.Load()

	// check config
	githubToken = os.Getenv("DA_TOKEN")
	if githubToken == "" {
		log.Fatalf("githubToken (DA_TOKEN) is missing!")
		return
	}

	githubRepository = os.Getenv("DA_REPO")
	if githubRepository == "" {
		log.Fatalf("githubRepository (DA_REPO) is missing!")
		return
	}

	mw := os.Getenv("DA_MAX_WORKERS")
	if mwi, err := strconv.Atoi(mw); err == nil && mwi > 0 {
		maxWorkers = mwi
	}

	log.Printf("Target repository: %s\n", githubRepository)
	log.Printf("Max workers: %d\n", maxWorkers)

	artifacts := getArtifacts()
	if artifacts == nil || len(artifacts) == 0 {
		log.Printf("no artifacts found")
		return
	}

	log.Printf("found %d artifacts\n", len(artifacts))
	deleteArtifacts(artifacts)
}

func getArtifacts() (artifacts []*artifact) {
	page := 1 // starting from page 1
	for {
		as := getPageArtifacts(page)
		// TODO: parse Link header to avoid fetch overhead
		if len(as) == 0 { // no artifacts in page
			break
		}
		artifacts = append(artifacts, as...)
		page++
	}
	return
}

func getPageArtifacts(page int) (artifacts []*artifact) {
	dest := githubAPI + "/repos/" + githubRepository + "/actions/artifacts?page=" + strconv.Itoa(page)
	req, err := http.NewRequest("GET", dest, nil)
	if err != nil {
		log.Printf("failed to make a request for page (%s): %v\n", dest, err)
		return
	}
	req.Header.Add("Authorization", "bearer "+githubToken)

	resp, err := cli.Do(req)
	if err != nil {
		log.Printf("failed to get page (%s): %v\n", dest, err)
		return
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	dec := json.NewDecoder(resp.Body)
	gr := new(githubResponse)
	err = dec.Decode(gr)
	if err != nil {
		log.Printf("failed to unmarshal API response: %v\n", err)
		return
	}

	return gr.Artifacts
}

func deleteArtifacts(artifacts []*artifact) {
	sema := make(chan struct{}, maxWorkers)
	wg := sync.WaitGroup{}
	for _, af := range artifacts {
		wg.Add(1)
		sema <- struct{}{}
		a := af
		go func() {
			deleteArtifact(a)
			wg.Done()
			<-sema
		}()
	}

	wg.Wait()
}

func deleteArtifact(artifact *artifact) {
	resource := githubAPI + "/repos/" + githubRepository + "/actions/artifacts/" + strconv.Itoa(artifact.ID)
	log.Printf("Deleting (%s)\n", resource)
	req, err := http.NewRequest("DELETE", resource, nil)
	if err != nil {
		log.Printf("failed to make a request for resource (%s): %v\n", resource, err)
		return
	}
	req.Header.Add("Authorization", "bearer "+githubToken)

	resp, err := cli.Do(req)
	if err != nil {
		log.Printf("failed to delete (%s): %v\n", resource, err)
		return
	}
	defer func() {
		io.Copy(ioutil.Discard, resp.Body)
		resp.Body.Close()
	}()

	if resp.StatusCode != 204 {
		log.Printf("failed to delete (%s): API returned %s\n", resource, resp.Status)
	}
}

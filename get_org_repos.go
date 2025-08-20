package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

type RepoList struct {
	Repositories []string `yaml:"repositories"`
}

func main() {
	token := flag.String("token", "", "GitHub API token")
	org := flag.String("org", "", "GitHub Organization name (e.g. my-org)")
	output := flag.String("output", "repos.yaml", "Output YAML file")
	flag.Parse()

	if *token == "" || *org == "" {
		log.Fatal("Usage: go run get_org_repos.go -token <token> -org <org> [-output repos.yaml]")
	}

	var allRepos []string
	page := 1
	perPage := 100

	for {
		url := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=%d&page=%d", *org, perPage, page)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", "Bearer "+*token)
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "API error: %s\n%s\n", resp.Status, string(body))
			os.Exit(1)
		}

		var repos []struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&repos); err != nil {
			log.Fatalf("Failed to decode response: %v", err)
		}

		if len(repos) == 0 {
			break
		}

		for _, repo := range repos {
			allRepos = append(allRepos, repo.Name)
		}
		if len(repos) < perPage {
			break
		}
		page++
	}

	outputData := RepoList{Repositories: allRepos}
	outBytes, err := yaml.Marshal(outputData)
	if err != nil {
		log.Fatalf("Failed to marshal YAML: %v", err)
	}
	if err := ioutil.WriteFile(*output, outBytes, 0644); err != nil {
		log.Fatalf("Failed to write YAML file: %v", err)
	}
	fmt.Printf("Wrote %d repositories to %s\n", len(allRepos), *output)
}

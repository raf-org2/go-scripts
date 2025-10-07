package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type RepoList struct {
	Repositories []string `yaml:"repositories"`
}

func main() {
	tokenFlag := flag.String("token", "", "GitHub API token (or set GITHUB_TOKEN_ORG / GITHUB_TOKEN env var)")
	org := flag.String("org", "", "GitHub Organization name (e.g. my-org)")
	output := flag.String("output", "repos.yaml", "Output YAML file")
	ghesURL := flag.String("ghes-url", "", "Base URL for GHES api (ignored for GHEC)")
	flag.Parse()

	// GHES_URL env var fallback
	if *ghesURL == "" {
		if envURL := os.Getenv("GHES_URL"); envURL != "" {
			*ghesURL = strings.TrimRight(envURL, "/")
		}
	}

	token := strings.TrimSpace(*tokenFlag)
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN_ORG")
	}
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "GitHub token must be provided via -token flag or GITHUB_TOKEN_ORG / GITHUB_TOKEN environment variable")
		os.Exit(1)
	}

	var allRepos []string
	page := 1
	perPage := 100

	for {
		githubEndpoint := os.Getenv("GITHUB_ENDPOINT")
		var url string
		switch githubEndpoint {
		case "GHEC":
			url = fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=%d&page=%d", *org, perPage, page)
		case "GHES":
			if *ghesURL == "" { log.Fatal("Set -ghes-url or GHES_URL when GITHUB_ENDPOINT=GHES") }
			url = fmt.Sprintf("%s/orgs/%s/repos?per_page=%d&page=%d", *ghesURL, *org, perPage, page)
		default:
			log.Fatalf("Unknown GitHub endpoint: %s", githubEndpoint)
		}
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", "Bearer "+ token)
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

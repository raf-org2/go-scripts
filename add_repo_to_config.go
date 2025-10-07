
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func parseRepoListFromFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read repo file '%s': %w", path, err)
	}
	content := strings.TrimSpace(string(data))
	if content == "" {
		return "", fmt.Errorf("repo file '%s' is empty", path)
	}
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	seps := func(r rune) bool { return r == '\n' || r == ',' || r == ';' }
	parts := strings.FieldsFunc(content, seps)
	uniq := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		uniq = append(uniq, p)
	}
	if len(uniq) == 0 {
		return "", fmt.Errorf("repo file '%s' contains no valid repositories", path)
	}
	return strings.Join(uniq, ","), nil
}

func main() {
	repo := flag.String("repo", "", "Repository name, list, 'all' or path to the repo list file")
	repoFile := flag.String("repo-file", "", "Path to a file containing a list of repository names (one per line or comma/semicolon separated)")
	org := flag.String("org", "", "GitHub Organization name (e.g. my-org)")
	token := flag.String("token", "", "GitHub API token")
	configName := flag.String("config", "sample", "Name of the code security configuration template")
	ghesURL := flag.String("ghes-url", "", "GitHub Enterprise Server URL (if using GHES)")
	flag.Parse()

	// GHES_URL env var fallback
	if *ghesURL == "" {
		if envURL := os.Getenv("GHES_URL"); envURL != "" {
			*ghesURL = strings.TrimRight(envURL, "/")
		}
	}

	githubToken := *token
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN_ORG")
	}
	if githubToken == "" {
		log.Fatal(". Use -token flag or set GITHUB_TOKEN_ORG environment variable.")
	}

	if *repo != "" {
		if info , err := os.Stat(*repo); err == nil && !info.IsDir() {
			parsed, perr := parseRepoListFromFile(*repo)
			if perr != nil {
				log.Fatalf("Error parsing repo file: %v", perr)
			}
			*repo = parsed
		}
	}

	if *repo == "" && *repoFile != "" {
		parsed, perr := parseRepoListFromFile(*repoFile)
		if perr != nil {
			log.Fatalf("Error parsing repo file: %v", perr)
		}
		*repo = parsed
	}

	if *repo == "" || *org == "" || githubToken == "" {
		log.Fatal("Usage: go run add_repo_to_config.go -repo <repo|all> -org <org> -token <token> [-config sample]")
	}

	client := &http.Client{}

	// 1. Get all configs for the org
	githubEndpoint := os.Getenv("GITHUB_ENDPOINT")
	var url string 
	switch githubEndpoint {
	case "GHEC":
		url = fmt.Sprintf("https://api.github.com/orgs/%s/code-security/configurations", *org)
	case "GHES", "":
		url = fmt.Sprintf("%s/orgs/%s/code-security/configurations", *ghesURL, *org)
	default:
		log.Fatalf("GITHUB_ENDPOINT environment variable must be set either to GHEC or GHES. Got '%s'", githubEndpoint)
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Fatalf("API error: %s\n%s\n", resp.Status, string(body))
	}
	body, _ := ioutil.ReadAll(resp.Body)
	var configs []map[string]interface{}
	if err := json.Unmarshal(body, &configs); err != nil {
		log.Fatalf("Failed to parse configs JSON: %v", err)
	}

	var configID int
	for _, cfg := range configs {
		if name, ok := cfg["name"].(string); ok && name == *configName {
			if id, ok := cfg["id"].(float64); ok {
				configID = int(id)
				break
			}
		}
	}
	if configID == 0 {
		log.Fatalf("Could not find configuration with name '%s'", *configName)
	}

	var repoIDs []int
	var repoNames []string
	if *repo == "all" {
		// Get all repos in the org
		page := 1
		perPage := 100
		for {
			var reposURL string	
			switch githubEndpoint {
			case "GHEC":
				reposURL = fmt.Sprintf("https://api.github.com/orgs/%s/repos?type=all&per_page=%d&page=%d", *org, perPage, page)
			case "GHES", "":
				reposURL = fmt.Sprintf("%s/orgs/%s/repos?type=all&per_page=%d&page=%d", *ghesURL, *org, perPage, page)
			default:
				log.Fatalf("GITHUB_ENDPOINT environment variable must be set either to GHEC or GHES. Got '%s'", githubEndpoint)
			}
			reposReq, err := http.NewRequest("GET", reposURL, nil)
			if err != nil {
				log.Fatalf("Failed to create request: %v", err)
			}
			reposReq.Header.Set("Accept", "application/vnd.github+json")
			reposReq.Header.Set("Authorization", "Bearer "+githubToken)
			reposReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
			reposResp, err := client.Do(reposReq)
			if err != nil {
				log.Fatalf("Request failed: %v", err)
			}
			if reposResp.StatusCode < 200 || reposResp.StatusCode >= 300 {
				reposBody, _ := ioutil.ReadAll(reposResp.Body)
				log.Fatalf("API error getting repos: %s\n%s\n", reposResp.Status, string(reposBody))
			}
			var repos []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}
			if err := json.NewDecoder(reposResp.Body).Decode(&repos); err != nil {
				log.Fatalf("Failed to parse repos JSON: %v", err)
			}
			reposResp.Body.Close()
			if len(repos) == 0 {
				break
			}
			for _, r := range repos {
				repoIDs = append(repoIDs, r.ID)
				repoNames = append(repoNames, r.Name)
			}
			if len(repos) < perPage {
				break
			}
			page++
		}
		if len(repoIDs) == 0 {
			log.Fatalf("No repositories found in organization '%s'", *org)
		}
	} else if strings.Contains(*repo, ",") {
		repoList := strings.Split(*repo, ",")
		for _, repoName := range repoList {
			repoName = strings.TrimSpace(repoName)
			if repoName == "" {
				continue
			}
			var repoURL string
			switch githubEndpoint {
			case "GHEC":
				repoURL = fmt.Sprintf("https://api.github.com/repos/%s/%s", *org, repoName)
			case "GHES", "":
				repoURL = fmt.Sprintf("%s/repos/%s/%s", *ghesURL, *org, repoName)
			default:
				log.Fatalf("GITHUB_ENDPOINT environment variable must be set either to GHEC or GHES. Got '%s'", githubEndpoint)
			}
			repoReq, err := http.NewRequest("GET", repoURL, nil)
			if err != nil {
				log.Fatalf("Failed to create request for %s: %v", repoName, err)
			}
			repoReq.Header.Set("Accept", "application/vnd.github+json")
			repoReq.Header.Set("Authorization", "Bearer "+githubToken)
			repoReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
			repoResp, err := client.Do(repoReq)
			if err != nil {
				log.Fatalf("Request failed for %s: %v", repoName, err)
			}
			if repoResp.StatusCode < 200 || repoResp.StatusCode >= 300 {
				repoBody, _ := ioutil.ReadAll(repoResp.Body)
				log.Printf("Warning couldn not find repository '%s': %s\n%s\n", repoName, repoResp.Status, string(repoBody))
				repoResp.Body.Close()
				continue
			}
			repoBody, _ := ioutil.ReadAll(repoResp.Body)
			var repoInfo struct {
				ID int `json:"id"`
				Name string `json:"name"`
			}
			if err := json.Unmarshal(repoBody, &repoInfo); err != nil {
				log.Printf("Failed to parse repo JSON for '%s': %v", repoName, err)
				continue
			}
			repoIDs = append(repoIDs, repoInfo.ID)
			repoNames = append(repoNames, repoInfo.Name)
			repoResp.Body.Close()
		}
		if len(repoIDs) == 0 {
			log.Fatal("No valid repositories found from the provided list")
		}
	} else {
		var repoURL string
		switch githubEndpoint {
		case "GHEC":
			repoURL = fmt.Sprintf("https://api.github.com/repos/%s/%s", *org, *repo)
		case "GHES", "":
			repoURL = fmt.Sprintf("%s/repos/%s/%s", *ghesURL, *org, *repo)
		default:
			log.Fatalf("GITHUB_ENDPOINT environment variable must be set either to GHEC or GHES. Got '%s'", githubEndpoint)
		}
		repoReq, err := http.NewRequest("GET", repoURL, nil)
		if err != nil {
			log.Fatalf("Failed to create request for %s: %v", *repo, err)
		}
		repoReq.Header.Set("Accept", "application/vnd.github+json")
		repoReq.Header.Set("Authorization", "Bearer "+githubToken)
		repoReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		repoResp, err := client.Do(repoReq)
		if err != nil {
			log.Fatalf("Request failed for %s: %v", *repo, err)
		}
		defer repoResp.Body.Close()
		if repoResp.StatusCode < 200 || repoResp.StatusCode >= 300 {
			repoBody, _ := ioutil.ReadAll(repoResp.Body)
			log.Fatalf("API error getting repo '%s': %s\n%s\n", *repo, repoResp.Status, string(repoBody))
		}
		repoBody, _ := ioutil.ReadAll(repoResp.Body)
		var repoInfo struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(repoBody, &repoInfo); err != nil {
			log.Fatalf("Failed to parse repo JSON for '%s': %v", *repo, err)
		}
		repoIDs = []int{repoInfo.ID}
		repoNames = []string{repoInfo.Name}
	}

	//Attach configuration
	var attachURL string
	switch githubEndpoint {
	case "GHEC":
		attachURL = fmt.Sprintf("https://api.github.com/orgs/%s/code-security/configurations/%d/repositories", *org, configID)
	case "GHES", "":
		attachURL = fmt.Sprintf("%s/orgs/%s/code-security/configurations/%d/repositories", *ghesURL, *org, configID)
	default:
		log.Fatalf("GITHUB_ENDPOINT environment variable must be set either to GHEC or GHES. Got '%s'", githubEndpoint)
	}
	attachBody := map[string]interface{}{
		"scope": "selected",
		"selected_repository_ids": repoIDs,
	}
	attachBodyBytes, _ := json.Marshal(attachBody)
	attachReq, err := http.NewRequest("POST", attachURL, bytes.NewBuffer(attachBodyBytes))
	if err != nil {
		log.Fatalf("Failed to create request for attaching repositories: %v", err)
	}
	attachReq.Header.Set("Accept", "application/vnd.github+json")
	attachReq.Header.Set("Authorization", "Bearer "+githubToken)
	attachReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	attachReq
	attachResp, err := client.Do(attachReq)
	if err != nil {
		log.Fatalf("Request failed for attaching repositories: %v", err)
	}
	defer attachResp.Body.Close()
	if attachResp.StatusCode == 202 {
		if *repo == "all" {
			fmt.Printf("All repositories in organization '%s' have been attached to configuration '%s' (ID: %d).\n", *org, *configName, configID)
			fmt.Printf("Total repositories attached: %d\n", len(repoIDs))
		} else if strings.Contains(*repo, ",") {
			fmt.Printf("Multiple repositories have been attached to configuration '%s' (ID: %d):\n", *configName, configID)
			for i, name := range repoNames {
				fmt.Printf("  - %s (ID: %d)\n", name, repoIDs[i])
			}
			fmt.Printf("Total repositories attached: %d\n", len(repoIDs))
		} else {
			fmt.Printf("Repository '%s' (ID: %d) has been attached to configuration '%s' (ID: %d).\n", repoNames[0], repoIDs[0], *configName, configID)
		}
		fmt.Println(string(attachRespBody))
	} else {
		fmt.Fprintf(os.Stderr, "API error: %s\n%s\n", attachResp.Status, string(attachRespBody))
		os.Exit(1)
	}
}
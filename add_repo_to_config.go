
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
)



func main() {
	repo := flag.String("repo", "", "Repository name to add (e.g. my-repo or 'all')")
	org := flag.String("org", "", "GitHub Organization name (e.g. my-org)")
	token := flag.String("token", "", "GitHub API token")
	configName := flag.String("config", "sample", "Name of the code security configuration template")
	flag.Parse()

	if *repo == "" || *org == "" || *token == "" {
		log.Fatal("Usage: go run add_repo_to_config.go -repo <repo|all> -org <org> -token <token> [-config sample]")
	}

	client := &http.Client{}

	// 1. Get all configs for the org
	url := fmt.Sprintf("https://api.github.com/orgs/%s/code-security/configurations", *org)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+*token)
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
			reposURL := fmt.Sprintf("https://api.github.com/orgs/%s/repos?per_page=%d&page=%d", *org, perPage, page)
			reposReq, err := http.NewRequest("GET", reposURL, nil)
			if err != nil {
				log.Fatalf("Failed to create repos request: %v", err)
			}
			reposReq.Header.Set("Accept", "application/vnd.github+json")
			reposReq.Header.Set("Authorization", "Bearer "+*token)
			reposReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
			reposResp, err := client.Do(reposReq)
			if err != nil {
				log.Fatalf("Repos request failed: %v", err)
			}
			if reposResp.StatusCode < 200 || reposResp.StatusCode >= 300 {
				reposBody, _ := ioutil.ReadAll(reposResp.Body)
				log.Fatalf("API error getting repos: %s\n%s\n", reposResp.Status, string(reposBody))
			}
			var repos []struct {
				ID int `json:"id"`
				Name string `json:"name"`
			}
			if err := json.NewDecoder(reposResp.Body).Decode(&repos); err != nil {
				log.Fatalf("Failed to decode repos: %v", err)
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
	} else {
		// Get the repository ID for the given repo name
		repoURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", *org, *repo)
		repoReq, err := http.NewRequest("GET", repoURL, nil)
		if err != nil {
			log.Fatalf("Failed to create repo request: %v", err)
		}
		repoReq.Header.Set("Accept", "application/vnd.github+json")
		repoReq.Header.Set("Authorization", "Bearer "+*token)
		repoReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		repoResp, err := client.Do(repoReq)
		if err != nil {
			log.Fatalf("Repo request failed: %v", err)
		}
		defer repoResp.Body.Close()
		if repoResp.StatusCode < 200 || repoResp.StatusCode >= 300 {
			repoBody, _ := ioutil.ReadAll(repoResp.Body)
			log.Fatalf("API error getting repo: %s\n%s\n", repoResp.Status, string(repoBody))
		}
		repoBody, _ := ioutil.ReadAll(repoResp.Body)
		var repoInfo struct {
			ID int `json:"id"`
			Name string `json:"name"`
		}
		if err := json.Unmarshal(repoBody, &repoInfo); err != nil {
			log.Fatalf("Failed to parse repo JSON: %v", err)
		}
		repoIDs = []int{repoInfo.ID}
		repoNames = []string{repoInfo.Name}
	}

	// Attach the configuration to the repository(ies) using the /attach endpoint
	attachURL := fmt.Sprintf("https://api.github.com/orgs/%s/code-security/configurations/%d/attach", *org, configID)
	attachBody := map[string]interface{}{
		"scope": "selected",
		"selected_repository_ids": repoIDs,
	}
	attachBodyBytes, _ := json.Marshal(attachBody)
	attachReq, err := http.NewRequest("POST", attachURL, bytes.NewBuffer(attachBodyBytes))
	if err != nil {
		log.Fatalf("Failed to create attach request: %v", err)
	}
	attachReq.Header.Set("Accept", "application/vnd.github+json")
	attachReq.Header.Set("Authorization", "Bearer "+*token)
	attachReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	attachReq.Header.Set("Content-Type", "application/json")
	attachResp, err := client.Do(attachReq)
	if err != nil {
		log.Fatalf("Attach request failed: %v", err)
	}
	defer attachResp.Body.Close()
	attachRespBody, _ := ioutil.ReadAll(attachResp.Body)
	if attachResp.StatusCode == 202 {
		if *repo == "all" {
			fmt.Printf("All repositories in org '%s' attached to configuration '%s' (ID: %d)\n", *org, *configName, configID)
			fmt.Printf("Attached %d repositories.\n", len(repoIDs))
		} else {
			fmt.Printf("Repository '%s' (ID: %d) attached to configuration '%s' (ID: %d)\n", repoNames[0], repoIDs[0], *configName, configID)
		}
		fmt.Println(string(attachRespBody))
	} else {
		fmt.Fprintf(os.Stderr, "API error: %s\n%s\n", attachResp.Status, string(attachRespBody))
		os.Exit(1)
	}
}

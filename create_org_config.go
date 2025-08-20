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

	"gopkg.in/yaml.v3"
)

type CodeSecurityConfig struct {
	Name                               string `yaml:"name" json:"name"`
	Description                        string `yaml:"description" json:"description"`
	AdvancedSecurity                   string `yaml:"advanced_security" json:"advanced_security"`
	DependencyGraph                    string `yaml:"dependency_graph" json:"dependency_graph"`
	DependencyGraphAutosubmitAction    string `yaml:"dependency_graph_autosubmit_action" json:"dependency_graph_autosubmit_action"`
	DependencyGraphAutosubmitActionOptions struct {
		LabeledRunners bool `yaml:"labeled_runners" json:"labeled_runners"`
	} `yaml:"dependency_graph_autosubmit_action_options" json:"dependency_graph_autosubmit_action_options"`
	DependabotAlerts                   string `yaml:"dependabot_alerts" json:"dependabot_alerts"`
	DependabotSecurityUpdates          string `yaml:"dependabot_security_updates" json:"dependabot_security_updates"`
	CodeScanningDefaultSetup           string `yaml:"code_scanning_default_setup" json:"code_scanning_default_setup"`
	CodeScanningDefaultSetupOptions    struct {
		RunnerType  string      `yaml:"runner_type" json:"runner_type"`
		RunnerLabel interface{} `yaml:"runner_label" json:"runner_label"`
	} `yaml:"code_scanning_default_setup_options" json:"code_scanning_default_setup_options"`
	SecretScanning                     string `yaml:"secret_scanning" json:"secret_scanning"`
	SecretScanningPushProtection       string `yaml:"secret_scanning_push_protection" json:"secret_scanning_push_protection"`
	SecretScanningValidityChecks       string `yaml:"secret_scanning_validity_checks" json:"secret_scanning_validity_checks"`
	SecretScanningNonProviderPatterns  string `yaml:"secret_scanning_non_provider_patterns" json:"secret_scanning_non_provider_patterns"`
	PrivateVulnerabilityReporting      string `yaml:"private_vulnerability_reporting" json:"private_vulnerability_reporting"`
	Enforcement                        string `yaml:"enforcement" json:"enforcement"`
}

func main() {
	yamlPath := flag.String("yaml", "", "Path to YAML file with configuration")
	token := flag.String("token", "", "GitHub API token")
	org := flag.String("org", "", "GitHub Organization name (e.g. my-org)")
	flag.Parse()

	if *yamlPath == "" || *token == "" || *org == "" {
		log.Fatal("Usage: go run create_org_config.go -yaml config.yaml -token <token> -org <org>")
	}

	data, err := ioutil.ReadFile(*yamlPath)
	if err != nil {
		log.Fatalf("Failed to read YAML file: %v", err)
	}

	var config CodeSecurityConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	jsonBody, err := json.Marshal(config)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}

	url := fmt.Sprintf("https://api.github.com/orgs/%s/code-security/configurations", *org)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+*token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Println("Configuration created successfully:")
		fmt.Println(string(body))
	} else {
		fmt.Fprintf(os.Stderr, "API error: %s\n%s\n", resp.Status, string(body))
		os.Exit(1)
	}
}

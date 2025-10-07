
package main

import (
	"bufio"
	"bytes"
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

// Reviewer represents a reviewer for delegated bypass
type Reviewer struct {
	ReviewerID   int    `yaml:"reviewer_id" json:"reviewer_id"`
	ReviewerType string `yaml:"reviewer_type" json:"reviewer_type"`
}

// SecretScanningDelegatedBypassOptions represents the options for delegated bypass
type SecretScanningDelegatedBypassOptions struct {
	Reviewers []Reviewer `yaml:"reviewers" json:"reviewers"`
}

type DependencyGraphAutosubmitActionOptions struct {
	LabeledRunners bool `yaml:"labeled_runners" json:"labeled_runners"`
}

type CodeScanningDefaultSetupOptions struct {
	RunnerType  string      `yaml:"runner_type" json:"runner_type"`
	RunnerLabel interface{} `yaml:"runner_label" json:"runner_label"`
}

type CodeSecurityConfig struct {
	ID int `json:"id"`
	Name                               string `yaml:"name" json:"name"`
	Description                        string `yaml:"description" json:"description"`
	AdvancedSecurity                   string `yaml:"advanced_security" json:"advanced_security"`
	DependencyGraph                    string `yaml:"dependency_graph" json:"dependency_graph"`
	DependencyGraphAutosubmitAction    string `yaml:"dependency_graph_autosubmit_action" json:"dependency_graph_autosubmit_action"`
	DependencyGraphAutosubmitActionOptions DependencyGraphAutosubmitActionOptions `yaml:"dependency_graph_autosubmit_action_options" json:"dependency_graph_autosubmit_action_options"`
	DependabotAlerts                   string `yaml:"dependabot_alerts" json:"dependabot_alerts"`
	DependabotSecurityUpdates          string `yaml:"dependabot_security_updates" json:"dependabot_security_updates"`
	CodeScanningDefaultSetup           string `yaml:"code_scanning_default_setup" json:"code_scanning_default_setup"`
	CodeScanningDefaultSetupOptions    CodeScanningDefaultSetupOptions `yaml:"code_scanning_default_setup_options" json:"code_scanning_default_setup_options"`
	SecretScanning                     string `yaml:"secret_scanning" json:"secret_scanning"`
	SecretScanningPushProtection       string `yaml:"secret_scanning_push_protection" json:"secret_scanning_push_protection"`
	SecretScanningValidityChecks       string `yaml:"secret_scanning_validity_checks" json:"secret_scanning_validity_checks"`
	SecretScanningNonProviderPatterns  string `yaml:"secret_scanning_non_provider_patterns" json:"secret_scanning_non_provider_patterns"`
	SecretScanningGenericSecrets       string `yaml:"secret_scanning_generic_secrets" json:"secret_scanning_generic_secrets"`
	SecretScanningDelegatedBypass      string `yaml:"secret_scanning_delegated_bypass" json:"secret_scanning_delegated_bypass"`
	SecretScanningDelegatedBypassOptions SecretScanningDelegatedBypassOptions `yaml:"secret_scanning_delegated_bypass_options" json:"secret_scanning_delegated_bypass_options"`
	SecretScanningDelegatedAlertDismissal string `yaml:"secret_scanning_delegated_alert_dismissal" json:"secret_scanning_delegated_alert_dismissal"`
	PrivateVulnerabilityReporting      string `yaml:"private_vulnerability_reporting" json:"private_vulnerability_reporting"`
	Enforcement                        string `yaml:"enforcement" json:"enforcement"`
	DefaultForNewRepos                bool   `yaml:"default_for_new_repos" json:"default_for_new_repos"`
}

func main() {
	yamlPath := flag.String("yaml", "", "Path to YAML file with new configuration")
	tokenFlag := flag.String("token", "", "GitHub API token")
	org := flag.String("org", "", "GitHub Organization name (e.g. my-org)")
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

	if *yamlPath == "" || token == "" || *org == "" {
		log.Fatal("Usage: go run update_org_config.go -yaml config.yaml -token <token> -org <org>")
	}

	// Read new config from YAML
	data, err := ioutil.ReadFile(*yamlPath)
	if err != nil {
		log.Fatalf("Failed to read YAML file: %v", err)
	}
	var newConfig CodeSecurityConfig
	if err := yaml.Unmarshal(data, &newConfig); err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}

	// Read current config from GitHub
	githubEndpoint := os.Getenv("GITHUB_ENDPOINT")
	var url string
	switch githubEndpoint {
	case "GHEC":
		url = fmt.Sprintf("https://api.github.com/orgs/%s/code-security/configurations", *org)
	case "GHES":
		if *ghesURL == "" { log.Fatal("Set -ghes-url or GHES_URL when GITHUB_ENDPOINT=GHES") }
		url = fmt.Sprintf("%s/orgs/%s/code-security/configurations", *ghesURL, *org)
	default:
		log.Fatalf("GITHUB_ENDPOINT environment variable must be set to either GHEC or GHES, or left unset for GHES as default")
	}	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+*token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	client := &http.Client{}
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
	var currentConfigs []CodeSecurityConfig
	if err := json.Unmarshal(body, &currentConfigs); err != nil {
		log.Fatalf("Failed to parse current config JSON (expected array): %v", err)
	}
	if len(currentConfigs) == 0 {
		fmt.Println("No current configuration found in GitHub API response.")
	}
	// Find the config with the same name as newConfig for diff and update
	var currentConfig CodeSecurityConfig
	for _, cfg := range currentConfigs {
		if cfg.Name == newConfig.Name {
			currentConfig = cfg
			break
		}
	}


	// Show diff and highlight changes
	fmt.Println("--- Diff (lines starting with '>' are changed) ---")
	hasChanges := printYAMLDiff(currentConfig, newConfig)
	if !hasChanges {
		fmt.Println("No changes detected.")
		return
	}
	fmt.Print("Apply these changes? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	if answer != "y\n" && answer != "Y\n" {
		fmt.Println("Aborted.")
		return
	}


	// Remove ID from request body (must not be sent)
	type CodeSecurityConfigRequest struct {
		Name                               string      `json:"name"`
		Description                        string      `json:"description"`
		AdvancedSecurity                   string      `json:"advanced_security"`
		DependencyGraph                    string      `json:"dependency_graph"`
		DependencyGraphAutosubmitAction    string      `json:"dependency_graph_autosubmit_action"`
		DependencyGraphAutosubmitActionOptions DependencyGraphAutosubmitActionOptions `json:"dependency_graph_autosubmit_action_options"`
		DependabotAlerts                   string      `json:"dependabot_alerts"`
		DependabotSecurityUpdates          string      `json:"dependabot_security_updates"`
		CodeScanningDefaultSetup           string      `json:"code_scanning_default_setup"`
		CodeScanningDefaultSetupOptions    CodeScanningDefaultSetupOptions `json:"code_scanning_default_setup_options"`
		SecretScanning                     string      `json:"secret_scanning"`
		SecretScanningPushProtection       string      `json:"secret_scanning_push_protection"`
		SecretScanningValidityChecks       string      `json:"secret_scanning_validity_checks"`
		SecretScanningNonProviderPatterns  string      `json:"secret_scanning_non_provider_patterns"`
		SecretScanningGenericSecrets       string      `json:"secret_scanning_generic_secrets"`
		SecretScanningDelegatedBypass      string      `json:"secret_scanning_delegated_bypass"`
		SecretScanningDelegatedBypassOptions SecretScanningDelegatedBypassOptions `json:"secret_scanning_delegated_bypass_options"`
		SecretScanningDelegatedAlertDismissal string `json:"secret_scanning_delegated_alert_dismissal"`
		PrivateVulnerabilityReporting      string      `json:"private_vulnerability_reporting"`
		Enforcement                        string      `json:"enforcement"`
		DefaultForNewRepos                bool        `json:"default_for_new_repos"`
	}
	reqBody := CodeSecurityConfigRequest{
		Name: newConfig.Name,
		Description: newConfig.Description,
		AdvancedSecurity: newConfig.AdvancedSecurity,
		DependencyGraph: newConfig.DependencyGraph,
		DependencyGraphAutosubmitAction: newConfig.DependencyGraphAutosubmitAction,
		DependencyGraphAutosubmitActionOptions: newConfig.DependencyGraphAutosubmitActionOptions,
		DependabotAlerts: newConfig.DependabotAlerts,
		DependabotSecurityUpdates: newConfig.DependabotSecurityUpdates,
		CodeScanningDefaultSetup: newConfig.CodeScanningDefaultSetup,
		CodeScanningDefaultSetupOptions: newConfig.CodeScanningDefaultSetupOptions,
		SecretScanning: newConfig.SecretScanning,
		SecretScanningPushProtection: newConfig.SecretScanningPushProtection,
		SecretScanningValidityChecks: newConfig.SecretScanningValidityChecks,
		SecretScanningNonProviderPatterns: newConfig.SecretScanningNonProviderPatterns,
		SecretScanningGenericSecrets: newConfig.SecretScanningGenericSecrets,
		SecretScanningDelegatedBypass: newConfig.SecretScanningDelegatedBypass,
		SecretScanningDelegatedBypassOptions: newConfig.SecretScanningDelegatedBypassOptions,
		SecretScanningDelegatedAlertDismissal: newConfig.SecretScanningDelegatedAlertDismissal,
		PrivateVulnerabilityReporting: newConfig.PrivateVulnerabilityReporting,
		Enforcement: newConfig.Enforcement,
	DefaultForNewRepos: newConfig.DefaultForNewRepos,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatalf("Failed to marshal JSON: %v", err)
	}


	// Try to find a config with the same name and get its integer ID
	configID := 0
	for _, cfg := range currentConfigs {
		if cfg.Name == newConfig.Name {
			configID = cfg.ID
			break
		}
	}

	var updateReq *http.Request
	if configID != 0 {
		// PATCH to update existing config using integer ID
		var patchURL string
		switch githubEndpoint {
		case "GHEC":
			patchURL = fmt.Sprintf("https://api.github.com/orgs/%s/code-security/configurations/%d", *org, configID)
		case "GHES":
			patchURL = fmt.Sprintf("%s/orgs/%s/code-security/configurations/%d", *ghesURL, *org, configID)
		default:
			log.Fatalf("GITHUB_ENDPOINT environment variable must be set to either GHEC or GHES, or left unset for GHES as default")
		}
	} else {
		// POST to create new config
		updateReq, err = http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			log.Fatalf("Failed to create POST request: %v", err)
		}
	}

	updateReq.Header.Set("Accept", "application/vnd.github+json")
	updateReq.Header.Set("Authorization", "Bearer "+ token)
	updateReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp, err := client.Do(updateReq)
	if err != nil {
		log.Fatalf("Update request failed: %v", err)
	}
	defer updateResp.Body.Close()
	updateBody, _ := ioutil.ReadAll(updateResp.Body)
	if updateResp.StatusCode >= 200 && updateResp.StatusCode < 300 {
		fmt.Println("Configuration updated successfully:")
		fmt.Println(string(updateBody))
	} else {
		fmt.Fprintf(os.Stderr, "API error: %s\n%s\n", updateResp.Status, string(updateBody))
		os.Exit(1)
	}
}


func printYAMLDiff(a, b CodeSecurityConfig) bool {
	aMap := structToJSONMap(a)
	bMap := structToJSONMap(b)
	// Remove fields that should not be compared
	delete(aMap, "id")
	delete(bMap, "id")
	delete(aMap, "target_type")
	delete(bMap, "target_type")
	fmt.Println("  (Only fields with true value changes are shown)")
	return diffMapRecursive(aMap, bMap, "")
}

func structToJSONMap(s interface{}) map[string]interface{} {
	var m map[string]interface{}
	j, err := json.Marshal(s)
	if err != nil {
		return map[string]interface{}{}
	}
	if err := json.Unmarshal(j, &m); err != nil {
		return map[string]interface{}{}
	}
	return m
}

func diffMapRecursive(a, b map[string]interface{}, prefix string) bool {
	changes := false
	for k, bVal := range b {
		if k == "id" || k == "target_type" {
			continue
		}
		aVal, ok := a[k]
		if !ok {
			fmt.Printf("> %s%s: %v (added)\n", prefix, k, bVal)
			changes = true
			continue
		}
		switch bValTyped := bVal.(type) {
		case map[string]interface{}:
			aValMap, ok := aVal.(map[string]interface{})
			if ok {
				if diffMapRecursive(aValMap, bValTyped, prefix+k+": ") {
					changes = true
				}
			} else {
				fmt.Printf("> %s%s: %v (type changed)\n", prefix, k, bVal)
				changes = true
			}
		default:
			if !jsonValuesEqual(aVal, bVal) {
				fmt.Printf("> %s%s: %v -> %v\n", prefix, k, aVal, bVal)
				changes = true
			}
		}
	}
	return changes
}

func jsonValuesEqual(a, b interface{}) bool {
	// Handle nil/null
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	// Compare as strings for simple types
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

func structToMap(s interface{}) map[string]interface{} {
	var m map[string]interface{}
	// Marshal to YAML then unmarshal to map for normalization
	y, err := yaml.Marshal(s)
	if err != nil {
		return map[string]interface{}{}
	}
	if err := yaml.Unmarshal(y, &m); err != nil {
		return map[string]interface{}{}
	}
	return m
}

func trimYAMLLine(s string) string {
	// Remove leading/trailing whitespace and ignore empty lines
	return strings.TrimSpace(s)
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

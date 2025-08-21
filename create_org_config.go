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

// setDefaultForNewRepos sets the default code security configuration for new repositories in the org.
func setDefaultForNewRepos(apiURL, org, token string, configID int, defaultFor string) error {
       reqBody := map[string]string{"default_for_new_repos": defaultFor}
       jsonReq, err := json.Marshal(reqBody)
       if err != nil {
	       return fmt.Errorf("failed to marshal JSON: %w", err)
       }
       url := fmt.Sprintf("%s/orgs/%s/code-security/configurations/%d/defaults", apiURL, org, configID)
       req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonReq))
       if err != nil {
	       return fmt.Errorf("failed to create request: %w", err)
       }
       req.Header.Set("Accept", "application/vnd.github+json")
       req.Header.Set("Authorization", "Bearer "+token)
       req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
       req.Header.Set("Content-Type", "application/json")
       client := &http.Client{}
       resp, err := client.Do(req)
       if err != nil {
	       return fmt.Errorf("request failed: %w", err)
       }
       defer resp.Body.Close()
       respBody, _ := ioutil.ReadAll(resp.Body)
       if resp.StatusCode >= 200 && resp.StatusCode < 300 {
	       fmt.Println("Default for new repos set successfully:")
	       fmt.Println(string(respBody))
	       return nil
       } else {
	       return fmt.Errorf("API error: %s\n%s", resp.Status, string(respBody))
       }
}


// SetDefaultForNewReposRequest represents the request body for setting the default config for new repos
type SetDefaultForNewReposRequest struct {
	DefaultForNewRepos string `json:"default_for_new_repos"`
	ConfigurationID    int    `json:"configuration_id"`
}

// TLCConfigFile allows top-level default_for_new_repos in YAML
type TLCConfigFile struct {
	DefaultForNewRepos string `yaml:"default_for_new_repos" json:"default_for_new_repos"`
	CodeSecurityConfig `yaml:",inline"`
}

type CodeSecurityConfig struct {
	// TargetType string `yaml:"target_type" json:"target_type"`
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
	// SecretScanningGenericSecrets       string `yaml:"secret_scanning_generic_secrets" json:"secret_scanning_generic_secrets"`
	SecretScanningDelegatedBypass      string `yaml:"secret_scanning_delegated_bypass" json:"secret_scanning_delegated_bypass"`
	SecretScanningDelegatedBypassOptions struct {
		Reviewers []struct {
			ReviewerID   int    `yaml:"reviewer_id" json:"reviewer_id"`
			ReviewerType string `yaml:"reviewer_type" json:"reviewer_type"`
		} `yaml:"reviewers" json:"reviewers"`
	} `yaml:"secret_scanning_delegated_bypass_options" json:"secret_scanning_delegated_bypass_options"`
	// SecretScanningDelegatedAlertDismissal string `yaml:"secret_scanning_delegated_alert_dismissal" json:"secret_scanning_delegated_alert_dismissal"`
	PrivateVulnerabilityReporting      string `yaml:"private_vulnerability_reporting" json:"private_vulnerability_reporting"`
	Enforcement                        string `yaml:"enforcement" json:"enforcement"`
	// CodeScanningOptions struct {
	//     AllowAdvanced bool `yaml:"allow_advanced" json:"allow_advanced"`
	// } `yaml:"code_scanning_options" json:"code_scanning_options"`
}

func main() {


	yamlPath := flag.String("yaml", "", "Path to YAML file with configuration")
	token := flag.String("token", "", "GitHub API token")
	org := flag.String("org", "", "GitHub Organization name (e.g. my-org)")
	apiURL := flag.String("api-url", "https://api.github.com", "Base API URL (e.g. https://github.mycompany.com/api/v3)")
	flag.Parse()

	   if *yamlPath == "" || *token == "" || *org == "" {
		   log.Fatal("Usage: go run create_org_config.go -yaml config.yaml -token <redacted> -org <org>")
	   }

	data, err := ioutil.ReadFile(*yamlPath)
	if err != nil {
		log.Fatalf("Failed to read YAML file: %v", err)
	}

	var tlcFile TLCConfigFile
	if err := yaml.Unmarshal(data, &tlcFile); err != nil {
		log.Fatalf("Failed to parse YAML: %v", err)
	}
	config := tlcFile.CodeSecurityConfig

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
	   // Set the token header, but never print it
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

		   // If default_for_new_repos is set in the YAML, set as default for new repos
		   if tlcFile.DefaultForNewRepos != "" {
			   var respObj map[string]interface{}
			   if err := json.Unmarshal(body, &respObj); err == nil {
				   var configID int
				   if id, ok := respObj["id"]; ok {
					   switch v := id.(type) {
					   case float64:
						   configID = int(v)
					   }
				   } else if val, ok := respObj["value"]; ok {
					   if m, ok := val.(map[string]interface{}); ok {
						   if id, ok := m["id"]; ok {
							   if v, ok := id.(float64); ok {
								   configID = int(v)
							   }
						   }
					   }
				   }
				   if configID != 0 {
					   err := setDefaultForNewRepos(*apiURL, *org, *token, configID, tlcFile.DefaultForNewRepos)
					   if err != nil {
						   fmt.Fprintf(os.Stderr, "Failed to set default for new repos: %v\n", err)
					   }
				   } else {
					   fmt.Fprintf(os.Stderr, "Could not determine configuration ID to set as default.\n")
				   }
			   }
		   }
	   } else {
		   fmt.Fprintf(os.Stderr, "API error: %s\n%s\n", resp.Status, string(body))
		   os.Exit(1)
	   }
}

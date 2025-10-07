package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v56/github"
	"golang.org/x/oauth2"
)

func main() {
	var (
		token   = flag.String("token", "", "GitHub token (or set GITHUB_TOKEN env var)")
		ghesURL = flag.String("ghes-url", "", "Base URL for GHES api (ignored for GHEC)")
	)
	flag.Parse()

	// Allow GHES URL to be supplied via GHES_URL env var if -ghes-url not provided
	if *ghesURL == "" {
		if envURL := os.Getenv("GHES_URL"); envURL != "" {
			// Trim any trailing slash for consistency
			trimmed := strings.TrimRight(envURL, "/")
			ghesURL = &trimmed
		}
	}

	// Get token from flag or environment
	githubToken := *token
	if githubToken == "" {
		githubToken = os.Getenv("GITHUB_TOKEN")
	}
	if githubToken == "" {
		log.Fatal("GitHub token is required. Use -token flag or set GITHUB_TOKEN environment variable")
	}

	// Create authenticated client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	githubEndpoint := os.Getenv("GITHUB_ENDPOINT")
	var client *github.Client
	switch githubEndpoint {
	case "GHEC":
		client = github.NewClient(tc)
	case "GHES":
		if ghesURL == nil || *ghesURL == "" {
			log.Fatal("When GITHUB_ENDPOINT=GHES you must provide the server base URL via -ghes-url or GHES_URL environment variable")
		}
		enterpriseClient,err := github.NewEnterpriseClient(*ghesURL, *ghesURL, tc)
		if err != nil {
			log.Fatalf("Failed to create GHES client: %v", err)
		}
		client = enterpriseClient
	default:
		log.Fatalf("GITHUB_ENDPOINT environment variable must be set to either GHEC or GHES, or left unset for GHES as default")
	}

	fmt.Println("🏢 Enterprise & Organization Access Test")
	fmt.Println(strings.Repeat("=", 45))

	// Test 1: Get authenticated user info
	fmt.Println("1. 👤 Authenticated User Info:")
	user, _, err := client.Users.Get(ctx, "")
	if err != nil {
		fmt.Printf("   ❌ Failed: %v\n", err)
		return
	}
	fmt.Printf("   ✅ User: %s (ID: %d)\n", *user.Login, *user.ID)


	// Test 2: List organizations
	fmt.Println("\n2. 🏢 Organization Memberships:")
	orgs, _, err := client.Organizations.List(ctx, "", &github.ListOptions{PerPage: 100})
	if err != nil {
		fmt.Printf("   ❌ Failed to list organizations: %v\n", err)
	} else if len(orgs) == 0 {
		fmt.Println("   📋 No organization memberships found")
	} else {
		fmt.Printf("   ✅ Member of %d organization(s):\n", len(orgs))
		for _, org := range orgs {
			// Check membership level
			membership, _, err := client.Organizations.GetOrgMembership(ctx, "", *org.Login)
			membershipInfo := ""
			if err == nil {
				membershipInfo = fmt.Sprintf(" \n      - Role: %s,  \n      - State: %s", *membership.Role, *membership.State)
			}

			// Count repositories in the organization
			repos, _, err := client.Repositories.ListByOrg(ctx, *org.Login, &github.RepositoryListByOrgOptions{
				Type: "all", 
				ListOptions: github.ListOptions{PerPage: 100},
			})
			
			if err != nil {
				fmt.Printf("      - %s (unable to count repos)%s\n", *org.Login, membershipInfo)
			} else {
				repoCount := len(repos)
				// If we got exactly 100 repos, there might be more (pagination)
				if repoCount == 100 {
					fmt.Printf("      - %s (%d+ repos found)%s\n", *org.Login, repoCount, membershipInfo)
				} else {
					fmt.Printf("      - %s (%d repos found)%s\n", *org.Login, repoCount, membershipInfo)
				}
			}
		}
	}

	// Test 3: GHAS Security Analysis Settings
	fmt.Println("\n3. 🛡️  GHAS Security Analysis Settings:")

	// List organizations again to check settings for each
	if len(orgs) == 0 {
		fmt.Println("   ⚠️  No organizations found to check settings.")
	} else {
		for _, org := range orgs {
			fmt.Printf("   🔍 Organization: %s\n", *org.Login)
			// List all repositories for the organization
			repos, _, err := client.Repositories.ListByOrg(ctx, *org.Login, &github.RepositoryListByOrgOptions{Type: "all", ListOptions: github.ListOptions{PerPage: 100}})
			if err != nil {
				fmt.Printf("      ❌ Failed to list repositories: %v\n", err)
				continue
			}
			for _, repo := range repos {
				fmt.Printf(" -----------------------\n      📦 Repo: %s\n", *repo.Name)
				// Get detailed repository information including security analysis settings
				detailedRepo, _, err := client.Repositories.Get(ctx, *org.Login, *repo.Name)
				if err != nil {
					fmt.Printf("         ❌ Failed to get repository details: %v\n", err)
					continue
				}
				if detailedRepo == nil || detailedRepo.SecurityAndAnalysis == nil {
					fmt.Printf("         ℹ️  No security analysis settings found.\n")
					continue
				}
				// Secret Scanning
				if detailedRepo.SecurityAndAnalysis.SecretScanning != nil {
					status := *detailedRepo.SecurityAndAnalysis.SecretScanning.Status
					icon := "❌"
					if status == "enabled" {
						icon = "✅"
					}
					fmt.Printf("         Secret Scanning: %s %s\n", status, icon)
				}
				// Secret Scanning Push Protection
				if detailedRepo.SecurityAndAnalysis.SecretScanningPushProtection != nil {
					status := *detailedRepo.SecurityAndAnalysis.SecretScanningPushProtection.Status
					icon := "❌"
					if status == "enabled" {
						icon = "✅"
					}
					fmt.Printf("         Secret Protection: %s %s\n", status, icon)
				}
				// Advanced Security
				if detailedRepo.SecurityAndAnalysis.AdvancedSecurity != nil {
					status := *detailedRepo.SecurityAndAnalysis.AdvancedSecurity.Status
					icon := "❌"
					if status == "enabled" {
						icon = "✅"
					}
					fmt.Printf("         Advanced Security: %s %s\n", status, icon)
				}
				// Dependabot Security Updates
				if detailedRepo.SecurityAndAnalysis.DependabotSecurityUpdates != nil {
					status := *detailedRepo.SecurityAndAnalysis.DependabotSecurityUpdates.Status
					icon := "❌"
					if status == "enabled" {
						icon = "✅"
					}
					fmt.Printf("         Dependabot Security Updates: %s %s\n", status, icon)
				}

				// Try to check for Code Scanning by attempting to list alerts
				_, resp, err := client.CodeScanning.ListAlertsForRepo(ctx, *org.Login, *repo.Name, &github.AlertListOptions{
					ListOptions: github.ListOptions{PerPage: 1},
				})
				if err != nil {
					if resp != nil && resp.StatusCode == 404 {
						fmt.Printf("         Code Scanning (CodeQL): not configured ❌\n")
					} else {
						fmt.Printf("         Code Scanning (CodeQL): access denied or error ⚠️\n")
					}
				} else {
					fmt.Printf("         Code Scanning (CodeQL): configured ✅\n")
				}

				// Try to check for Dependabot by attempting to list alerts
				_, resp, err = client.Dependabot.ListRepoAlerts(ctx, *org.Login, *repo.Name, &github.ListAlertsOptions{
					ListOptions: github.ListOptions{PerPage: 1},
				})
				if err != nil {
					if resp != nil && resp.StatusCode == 404 {
						fmt.Printf("         Dependabot Scanning: not configured ❌\n")
					} else {
						fmt.Printf("         Dependabot Scanning: access denied or error ⚠️\n")
					}
				} else {
					fmt.Printf("         Dependabot Scanning: configured ✅\n")
				}
			}
		}
	}

	fmt.Printf("\n" + strings.Repeat("=", 45))
	fmt.Println("🏁 Organization review completed!")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    s[:len(substr)+1] == substr+"," || 
		    s[len(s)-len(substr)-1:] == ","+substr ||
		    findInMiddle(s, substr))
}

func findInMiddle(s, substr string) bool {
	target := "," + substr + ","
	for i := 0; i <= len(s)-len(target); i++ {
		if s[i:i+len(target)] == target {
			return true
		}
	}
	return false
}

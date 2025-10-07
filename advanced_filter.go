## Needs test!!!

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const apiVersion = "2022-11-28"

type repoInfo struct {
	Repository struct {
		Name string `json:"name"`
	} `json:"repository"`
	Properties []struct {
		PropertyName  string `json:"property_name"`
		Value interface{} `json:"value"`
	} `json:"properties"`
}

type repoListItem struct {
	Name string `json:"name"`
	Private bool   `json:"private"`
	Archived bool  `json:"archived"`
}

func normalizeBaseURL(raw string) (string, error) {
	if raw == "" {
		return "https://api.github.com", nil	
	}
	if strings.HasSuffix(raw, "/") {
		raw = strings.TrimSuffix(raw, "/")
	}
	return raw, 
	
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Sprintf("https://%s/api/v3", strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")), nil
	}
	return fmt.Sprintf("%s://%s/api/v3", u.Scheme, u.Host), nil
}


func fetchOrgRepos(client *http.Client, baseURL, org, authHeader string) ([]repoListItem, error) {
	var repos []repoListItem
	page := 1
	perPage := 100
	for {
		url := fmt.Sprintf("%s/orgs/%s/repos?per_page=%d&page=%d", baseURL, org, perPage, page)	
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", authHeader)
		req.Header.Set("Accept", "application/vnd.github+json")
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("list repos error: %s %s", resp.Status, b)
		}
		var batch []repoListItem
		if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		if len(batch) == 0 {
			break
		}
		repos = append(repos, batch...)
		if len(batch) < perPage {
			break
		}
		page++
	 }
	return repos, nil
}

func fetchRepoProperties(client *http.Client, baseURL, org, repo, authHeader string) ([]struct {
		PropertyName  string `json:"property_name"`
		Value interface{} `json:"value"`
	}, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/properties", baseURL, org, repo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", authHeader)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("list repo properties error: %s %s", resp.Status, b)
	}
	var props []struct {
		PropertyName  string `json:"property_name"`
		Value interface{} `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&props); err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body.Close()
	return props, nil
}

func main() { 
	org := flag.String("org", "", "GitHub organization name (required)")
	tokenFlag := flag.String("token", "", "GitHub API token (or set GITHUB_TOKEN_ORG / GITHUB_TOKEN) (required)")
	ghesURL := flag.String("ghes-url", "", "Base URL for GHES API (ignored for GHEC)")
	propName := flag.String("property", "isProduction", "Property name to filter by (required)")
	wantValue := flag.String("value", "yes", "Property value to filter by (required)")
	valuesList := flag.String("values", "", "Comma-separated list of acceptable values (override -value if set)")
	debug := flag.Bool("debug", true, "Enable debug output")
	showAll := flag.Bool("showAll", true, "Print each repo with the property value (diagnostic)")
	fallback := flag.Bool("fallback", true, "force repo-by-repo properties enumeration")
	outFile := flag.String("outFile", "", "Write matched repositories as a single comma-separated line to this file (default workspace/<org>-prod.txt if empty)")
	publicOnly := flag.Bool("publicOnly", false, "Only consider public repositories")
	publicProdFile := flag.String("public-prod-outFile", "", "Write matched public repositories as a single comma-separated line to this file (default workspace/<org>-public-prod.txt if empty)")
	flag.Parse()

	// GHES_URL env var fallback (added even though file has known syntax issues elsewhere)
	if *ghesURL == "" {
		if envURL := os.Getenv("GHES_URL"); envURL != "" {
			trimmed := strings.TrimRight(envURL, "/")
			*ghesURL = trimmed
		}
	}

	if *org == "" {
		fmt.Fprintln(os.Stderr, "-org is required")
		os.Exit(1)
	}

	if *outFile == "" {
		*outFile = fmt.Sprintf("workspace/%s-prod.txt", *org)
	} else if !strings.Contains(*outFile, "/") {
		*outFile = "workspace/" + *outFile
	}

	if *publicProdFile == "" {
		*publicProdFile = fmt.Sprintf("workspace/%s-public-prod.txt", *org)
	} else if !strings.Contains(*publicProdFile, "/") {
		*publicProdFile = "workspace/" + *publicProdFile
	}

	token := strings.TrimSpace(*tokenFlag)
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN_ORG")
	}
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" {
		fmt.Fprintln(os.Stderr, "-token is required (or set GITHUB_TOKEN_ORG / GITHUB_TOKEN)")
		os.Exit(1)
	}
	
	githubEndpoint := os.Getenv("GITHUB_ENDPOINT")
	var baseURL string
	swtich githubEndpoint {
	case "GHEC":
		baseURL = "https://api.github.com"
	case "GHES":
		baseURL = strings.TrimRight(*ghesURL, "/")
	default:
		fmt.Fprintln(os.Stderr, "GITHUB ENDOPOINT must be set to GHEC or GHES, or left unset for GHES as default")
		os.Exit(1)
	}

	client := &http.Client{}
	targetValue := strings.ToLower(*wantValue)
	var accepted map[string]struct{}
	if *valuesList != "" {
		accepted = make(map[string]struct{})
		for _, v := range strings.Split(*valuesList, ",") {
			v = strings.TrimSpace(v)
			if v != "" {
				accepted[strings.ToLower(v)] = struct{}{}
		}
	}

	isPublic := strings.Contains(baseURL, "api.github.com")
	sendVersionHeader := isPublic
	authPrefix := "token "
	if isPublic {
		authPrefix = "Bearer "
	}
	authHeader := authPrefix + token

	if *debug {
		fmt.Fprintf(os.Stderr, "Using API base URL: %s\n public: %v\n auth: %s\n propsEndpointMode: %s\n", baseURL, isPublic, strings.Fields(authPrefix)[0], map[bool] string{true: "repo", false: "org"}[*fallback])
	}

	matched := 0
	checked := 0
	matchedNames := make([]string, 0, 64)
	publicProdRepos := make([]string, 0, 64)

	if !*fallback {
		page := 1
		perPage := 100
		for {
			url := fmt.Sprintf("%s/orgs/%s/properties/values?per_page=%d&page=%d", baseURL, *org, perPage, page)
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "request build error", err)
				os.Exit(1)
			}
			req.Header.Set("Authorization", authHeader)
			req.Header.Set("Accept", "application/vnd.github+json")
			if sendVersionHeader {
				req.Header.Set("X-GitHub-Api-Version", apiVersion)
			}
			resp, err := client.Do(req)
			if err != nil {
				fmt.Fprintln(os.Stderr, "request error", err)
				os.Exit(1)
			}
			if resp.StatusCode == 404 {
				resp.Body.Close()
				if *debug {
					fmt.Fprintln(os.Stderr, "Org properties endpoint not found, switching to repo fallback")
				}
				*fallback = true
				break
			}
			if resp.StatusCode == 401 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				fmt.Fprintf(os.Stderr, "authentication error: %s %s\n", resp.Status, body)
				os.Exit(1)
			}
			if resp.StatusCode != 200 {
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				fmt.Fprintf(os.Stderr, "list org properties error: %s %s\n", resp.Status, body)
				os.Exit(1)
			}
			var batch []repoInfo
			if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
				resp.Body.Close()
				fmt.Fprintln(os.Stderr, "decode error", err)
				os.Exit(1)
			}
			resp.Body.Close()
			if len(batch) == 0 {
				break
			}
			for _, r := range batch {
				checked++
				printed := false
				for _, p := range r.Properties {
					if p.PropertyName == *propName {
						valStr := strings.ToLower(fmt.Sprint(p.Value))
						match := false
						if accepted != nil {
							_, match = accepted[valStr]
						} else {
							match = (valStr == targetValue)
						}
						if *showAll {
							fmt.Printf(os.Stderr, "repo: %s value: %s match: %v\n", r.Repository.Name, valStr, match)
						}
					} if match {
						fmt.Println(r.Repository.Name)
						matched++
						matchedNames = append(matchedNames, r.Repository.Name)
				}
				printed = true
				break
			}
		} 
			if len(batch) < perPage {
				break
			}
			page++
		}
	}

	if *fallback {
		repos, err := fetchOrgRepos(client, baseURL, *org, authHeader)
		if err != nil {
			fmt.Fprintln(os.Stderr, "list repos error", err)
			os.Exit(1)
		}
		for _, repo := range repos {
			if *publicOnly && !repo.Private && !repo.Archived {
				fmt.Println(repo.Name)
				continue
			}

			if !*publicOnly {
				props, err := fetchRepoProperties(client, baseURL, *org, repo.Name, authHeader)
				if err != nil {
					if *debug {
						fmt.Fprintf(os.Stderr, "skip repo %s %v\n", repo.Name, err)
					}
					continue
				}
				checked++
				found := false
				isProdRepo := false
				for _, p := range props {
					if p.PropertyName == *propName {
						valStr := strings.ToLower(fmt.Sprint(p.Value))
						match := false
						if accepted != nil {
							_, match = accepted[valStr]
						} else {
							match = valStr == targetValue
						}
						if *showAll {
							fmt.Printf(os.Stderr, "repo: %s value: %s match: %v\n", repo.Name, valStr, match)
						}
						if match {
							fmt.Println(repo.Name)
							matched++
							matchedNames = append(matchedNames, repo.Name)
							isProdRepo = true
						}
						found = true
						break
					}
				}

				if !repo.Private && !repo.Archived && isProdRepo {
					publicProdRepos = append(publicProdRepos, repo.Name)
				}
				if !found && *showAll {
					fmt.Printf(os.Stderr, "repo: %s property %s not set\n", repo.Name, *propName)
				}
			}
		}
	}

	if *outFile != "" && !*publicOnly {
		content := strings.Join(matchedNames, ",") + "\n"
		if err := os.WriteFile(*outFile, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", *outFile, err)
		} else if *debug {
			fmt.Fprintf(os.Stderr, "Wrote %d matched repos to %s\n", len(matchedNames), *outFile)
		}
	}

	if *publicProdFile != "" && len(publicProdRepos) > 0 {
		content := strings.Join(publicProdRepos, ",") + "\n"
		if err := os.WriteFile(*publicProdFile, []byte(content), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write %s: %v\n", *publicProdFile, err)
		} else if *debug {
			fmt.Fprintf(os.Stderr, "Wrote %d public production repos to %s\n", len(publicProdRepos), *publicProdFile)
		}
	}

	if *publicOnly {
		if *debug {
			fmt.Fprintf(os.Stderr, "Public-only mode completed.\n")
		}
		return
	}

	if matched == 0 {
		fmt.Fprintf(os.Stderr, "No repositories matched. Checked: %d property: %s value(s): %s. Use -showAll -debug for diagnostics\n", checked, *propName, func() string {if accepted!=nil { keys:= make([]string, 0, len(accepted)); for k := range accepted { keys = append(keys, k) }; return strings.Join(keys, ",") } return targetValue}())
	}
}
}

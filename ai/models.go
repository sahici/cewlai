package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type modelEntry struct {
	ID string `json:"id"`
}

type modelList struct {
	Data []modelEntry `json:"data"`
}

func ListModels(provider, apiKey, baseURL string) {
	p := strings.ToLower(provider)

	if p == "" {
		fmt.Fprintln(os.Stderr, "Error: -p (provider) is required with --list-models")
		os.Exit(1)
	}

	if preset, ok := providerPresets[p]; ok {
		if baseURL == "" {
			baseURL = preset.baseURL
		}
		if apiKey == "" {
			apiKey = os.Getenv(preset.envKey)
		}
	} else {
		switch p {
		case "anthropic":
			if baseURL == "" {
				baseURL = "https://api.anthropic.com/v1"
			}
			if apiKey == "" {
				apiKey = os.Getenv("ANTHROPIC_API_KEY")
			}
		case "openai":
			if baseURL == "" {
				baseURL = "https://api.openai.com/v1"
			}
			if apiKey == "" {
				apiKey = os.Getenv("OPENAI_API_KEY")
			}
		case "opencode":
			// opencode lists its own providers/models over /config/providers.
			listOpenCodeModels(baseURL)
			return
		}
	}

	url := strings.TrimRight(baseURL, "/") + "/models"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = resp.Body.Close() }()

	var models modelList
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Available models for %s:\n\n", provider)
	for _, m := range models.Data {
		fmt.Println(m.ID)
	}
}

// openCodeProvidersResp is the shaped subset of opencode's
// GET /config/providers response we care about.
type openCodeProvidersResp struct {
	Providers []struct {
		ID     string                     `json:"id"`
		Models map[string]openCodeModelID `json:"models"`
	} `json:"providers"`
	// default maps providerID -> modelID.
	Default map[string]string `json:"default"`
}

type openCodeModelID struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// listOpenCodeModels prints the providers/models exposed by a local opencode
// server via GET /config/providers.
func listOpenCodeModels(baseURL string) {
	if baseURL == "" {
		baseURL = DefaultOpenCodeBaseURL
	}
	url := strings.TrimRight(baseURL, "/") + "/config/providers"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to opencode at %s: %v\n", baseURL, err)
		os.Exit(1)
	}
	defer func() { _ = resp.Body.Close() }()

	var data openCodeProvidersResp
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing opencode response: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Available models for opencode (%s):\n\n", baseURL)
	shown := 0
	for _, pr := range data.Providers {
		if len(pr.Models) == 0 {
			continue
		}
		fmt.Printf("%s:\n", pr.ID)
		for id := range pr.Models {
			fmt.Printf("  %s/%s\n", pr.ID, id)
			shown++
		}
	}
	if shown == 0 {
		fmt.Println("(no provider model list returned; make sure opencode serve is running and providers are configured)")
	}
	if def, ok := data.Default["opencode"]; ok {
		fmt.Fprintf(os.Stderr, "\nDefault (opencode) model: %s\n", def)
	}
}

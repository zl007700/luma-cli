package commands

import (
	"fmt"
	"os"

	"github.com/luma-cli/lumer-cli/cloud"
	appconfig "github.com/luma-cli/lumer-cli/internal/config"
	"github.com/luma-cli/lumer-cli/internal/output"
)

const (
	prodAPIURL    = "https://api.pikgeo.com"
	envTestAPIURL = "LUMA_TEST_API_URL"
)

func printVersion() {
	apiURL := cloud.BaseURL()
	envName := activeEnvironmentName(apiURL)
	fmt.Printf("luma-cli version %s\n", version)
	if envName == "test" {
		fmt.Printf("backend: TEST (%s)\n", apiURL)
		return
	}
	fmt.Printf("backend: %s\n", apiURL)
}

func cmdEnv(args []string) error {
	if len(args) == 0 || args[0] == "show" {
		return showEnv()
	}

	switch args[0] {
	case "use":
		if len(args) < 2 {
			return output.ErrValidation("usage: luma-cli env use test|prod|<url>")
		}
		envName, apiURL := resolveEnvTarget(args[1])
		if envName == "test" && apiURL == "" {
			return output.ErrValidation("set %s before running `luma-cli env use test`, or run `luma-cli env use <url>`", envTestAPIURL)
		}
		if err := appconfig.SaveEnvironment(envName, apiURL); err != nil {
			return output.ErrSystem("write config: %v", err)
		}
		if runtimeOpts.JSON {
			return output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{
				"environment": envName,
				"api_url":     apiURL,
			}})
		}
		fmt.Printf("Luma backend -> %s (%s)\n", envName, apiURL)
		if envName == "test" {
			fmt.Println("Notice: test backend is active.")
		}
		return nil
	default:
		return output.ErrValidation("usage: luma-cli env show | luma-cli env use test|prod|<url>")
	}
}

func showEnv() error {
	apiURL := cloud.BaseURL()
	envName := activeEnvironmentName(apiURL)
	if runtimeOpts.JSON {
		return output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{
			"environment": envName,
			"api_url":     apiURL,
		}})
	}
	fmt.Printf("Backend: %s (%s)\n", envName, apiURL)
	if envName == "test" {
		fmt.Println("Notice: test backend is active.")
	}
	return nil
}

func resolveEnvTarget(target string) (string, string) {
	switch target {
	case "prod", "production":
		return "prod", prodAPIURL
	case "test", "testing":
		return "test", os.Getenv(envTestAPIURL)
	default:
		apiURL := target
		return appconfig.EnvironmentName(apiURL, ""), apiURL
	}
}

func activeEnvironmentName(apiURL string) string {
	cfg, err := appconfig.Load()
	if err == nil && cfg != nil && cfg.APIURL == apiURL {
		return appconfig.EnvironmentName(apiURL, cfg.Environment)
	}
	return appconfig.EnvironmentName(apiURL, "")
}

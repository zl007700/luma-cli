package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/luma-cli/lumer-cli/cloud"
	appconfig "github.com/luma-cli/lumer-cli/internal/config"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func cmdAuth(args []string) error {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli auth login       (browser-based login)")
		fmt.Println("       luma-cli auth login <account> (browser-based login with account hint)")
		fmt.Println("       luma-cli auth login --account <account>")
		fmt.Println("       luma-cli auth login --key <key> (paste API key or legacy card key directly)")
		fmt.Println("       luma-cli auth login <key> (paste API key or legacy card key directly)")
		fmt.Println("       luma-cli auth status      (show login status)")
		return nil
	}

	switch args[0] {
	case "login":
		loginOpts := parseAuthLoginArgs(args[1:])
		if loginOpts.Help {
			printAuthUsage()
			return nil
		}
		if loginOpts.DirectKey != "" {
			if err := appconfig.SaveCardKey(loginOpts.DirectKey); err != nil {
				return output.ErrSystem("write config: %v\n", err)
			}
			fmt.Println("Login saved.")
			return nil
		}
		return deviceFlowLogin(loginOpts.Account)

	case "status":
		cfg := loadConfig()
		loggedIn := cfg != nil && cfg.CardKey != ""
		if runtimeOpts.JSON {
			data := map[string]any{"logged_in": loggedIn, "api_url": cloud.BaseURL()}
			if loggedIn {
				data["key"] = appconfig.MaskKey(cfg.CardKey)
			}
			if err := output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: data}); err != nil {
				return output.ErrSystem("%v", err)
			}
			return nil
		}
		if !loggedIn {
			fmt.Println("Not logged in. Run: luma-cli auth login")
		} else {
			fmt.Printf("Logged in. Key: %s\n", appconfig.MaskKey(cfg.CardKey))
		}
		fmt.Printf("Backend: %s\n", cloud.BaseURL())
		return nil

	default:
		printAuthUsage()
	}
	return nil
}

type authLoginOptions struct {
	Account   string
	DirectKey string
	Help      bool
}

func printAuthUsage() {
	fmt.Println("usage: luma-cli auth login")
	fmt.Println("       luma-cli auth login <account>")
	fmt.Println("       luma-cli auth login --account <account>")
	fmt.Println("       luma-cli auth login --key <key>")
	fmt.Println("       luma-cli auth status")
}

func parseAuthLoginArgs(args []string) authLoginOptions {
	var opts authLoginOptions
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		switch arg {
		case "-h", "--help":
			opts.Help = true
		case "--account", "--phone", "--email":
			if i+1 < len(args) {
				opts.Account = strings.TrimSpace(args[i+1])
				i++
			}
		case "--key", "--token", "--card-key":
			if i+1 < len(args) {
				opts.DirectKey = strings.TrimSpace(args[i+1])
				i++
			}
		default:
			if opts.Account == "" && opts.DirectKey == "" {
				if looksLikeAccount(arg) {
					opts.Account = arg
				} else {
					opts.DirectKey = arg
				}
			}
		}
	}
	return opts
}

func looksLikeAccount(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	if strings.Contains(value, "@") {
		return true
	}
	digits := 0
	for _, ch := range value {
		if ch >= '0' && ch <= '9' {
			digits++
		} else if ch != '+' && ch != '-' && ch != ' ' {
			return false
		}
	}
	return digits >= 6
}

func deviceFlowLogin(account string) error {
	base := cloud.BaseURL()

	// 1. Activate device
	type activateResp struct {
		DeviceCode string `json:"device_code"`
		UserCode   string `json:"user_code"`
		VerifyURL  string `json:"verify_url"`
		ExpiresIn  int    `json:"expires_in"`
	}
	var act activateResp
	body := map[string]string{"client_name": "Luma CLI"}
	if strings.TrimSpace(account) != "" {
		body["account"] = strings.TrimSpace(account)
	}
	if err := postJSON(base+"/api/auth/device/activate", body, &act); err != nil {
		return output.ErrNetwork("activate device: %v", err)
	}

	// 2. Open browser
	fmt.Printf("Opening browser to: %s\n", act.VerifyURL)
	fmt.Printf("Your code: %s\n\n", act.UserCode)
	if strings.TrimSpace(account) != "" {
		fmt.Printf("Account hint: %s\n", strings.TrimSpace(account))
	}
	fmt.Println("If the browser does not open, visit the URL above.")
	openBrowser(act.VerifyURL)

	// 3. Poll for token
	fmt.Print("Waiting for authorization")
	deadline := time.Now().Add(time.Duration(act.ExpiresIn) * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(2 * time.Second)
		fmt.Print(".")

		type pollResp struct {
			Status      string `json:"status"`
			AccessToken string `json:"access_token"`
			APIKey      string `json:"api_key"`
			MaskedKey   string `json:"masked_key"`
		}
		var pr pollResp
		if err := postJSON(base+"/api/auth/device/poll", map[string]string{
			"device_code": act.DeviceCode,
		}, &pr); err != nil {
			continue
		}

		if pr.Status == "success" {
			fmt.Println("\nAuthorized!")
			key := strings.TrimSpace(pr.APIKey)
			if key == "" {
				key = strings.TrimSpace(pr.AccessToken)
			}
			if key == "" {
				return output.ErrSystem("authorization response did not include an API key")
			}
			if err := appconfig.SaveCardKey(key); err != nil {
				return output.ErrSystem("save key: %v", err)
			}
			if pr.MaskedKey != "" {
				fmt.Printf("API key saved: %s\n", pr.MaskedKey)
			} else {
				fmt.Println("Login saved.")
			}
			return nil
		}
		if pr.Status == "expired" {
			fmt.Println("\nAuthorization expired. Run luma-cli auth login again.")
			return nil
		}
	}
	fmt.Println("\nAuthorization timed out. Run luma-cli auth login again.")
	return nil
}

func postJSON(url string, body any, result any) error {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = strings.NewReader(string(b))
	}
	req, err := http.NewRequest("POST", url, bodyReader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(data))
	}
	if result != nil {
		return json.Unmarshal(data, result)
	}
	return nil
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}

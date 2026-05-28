package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
		fmt.Println("       luma-cli auth complete    (finish after the user approves)")
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
		return startDeviceFlowLogin(loginOpts.Account)

	case "complete", "finish":
		deviceCode := ""
		if len(args) >= 2 {
			deviceCode = strings.TrimSpace(args[1])
		}
		return completeDeviceFlowLogin(deviceCode)

	case "status":
		cfg := loadConfig()
		loggedIn := cfg != nil && cfg.CardKey != ""
		if runtimeOpts.JSON {
			data := map[string]any{"logged_in": loggedIn, "api_url": cloud.BaseURL()}
			if loggedIn {
				data["key"] = appconfig.MaskKey(cfg.CardKey)
			}
			if cfg != nil && cfg.PendingAuthDeviceCode != "" {
				data["pending_auth"] = true
				data["verify_url"] = cfg.PendingAuthVerifyURL
				data["code"] = cfg.PendingAuthUserCode
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
		if cfg != nil && cfg.PendingAuthDeviceCode != "" {
			fmt.Printf("Pending authorization: %s\n", cfg.PendingAuthVerifyURL)
			fmt.Println("After approval, run: luma-cli auth complete")
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
	fmt.Println("       luma-cli auth complete")
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

func startDeviceFlowLogin(account string) error {
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
	if err := appconfig.SavePendingDeviceAuth(act.DeviceCode, act.UserCode, act.VerifyURL); err != nil {
		return output.ErrSystem("save pending authorization: %v", err)
	}

	// 2. Print authorization instructions. Do not open a local browser because
	// this command often runs inside a cloud-hosted agent environment.
	fmt.Printf("Authorization URL: %s\n", act.VerifyURL)
	fmt.Printf("Code: %s\n\n", act.UserCode)
	if strings.TrimSpace(account) != "" {
		fmt.Printf("Account hint: %s\n", strings.TrimSpace(account))
	}
	fmt.Println("Open the URL in your own browser, sign in, and approve the authorization.")
	fmt.Println("After approval, run: luma-cli auth complete")
	return nil
}

func completeDeviceFlowLogin(deviceCode string) error {
	if strings.TrimSpace(deviceCode) == "" {
		cfg := loadConfig()
		if cfg != nil {
			deviceCode = cfg.PendingAuthDeviceCode
		}
	}
	deviceCode = strings.TrimSpace(deviceCode)
	if deviceCode == "" {
		return output.ErrAuth("no pending authorization. Run: luma-cli auth login <account>")
	}

	type pollResp struct {
		Status      string `json:"status"`
		AccessToken string `json:"access_token"`
		APIKey      string `json:"api_key"`
		MaskedKey   string `json:"masked_key"`
	}
	var pr pollResp
	if err := postJSON(cloud.BaseURL()+"/api/auth/device/poll", map[string]string{
		"device_code": deviceCode,
	}, &pr); err != nil {
		return output.ErrNetwork("complete authorization: %v", err)
	}

	switch pr.Status {
	case "success":
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
		if err := appconfig.ClearPendingDeviceAuth(); err != nil {
			return output.ErrSystem("clear pending authorization: %v", err)
		}
		if pr.MaskedKey != "" {
			fmt.Printf("API key saved: %s\n", pr.MaskedKey)
		} else {
			fmt.Println("Login saved.")
		}
	case "pending":
		fmt.Println("Authorization is not approved yet. Ask the user to open the URL and approve, then run: luma-cli auth complete")
	case "expired":
		_ = appconfig.ClearPendingDeviceAuth()
		fmt.Println("Authorization expired. Run: luma-cli auth login again.")
	default:
		return output.ErrSystem("unexpected authorization status: %s", pr.Status)
	}
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
		if os.IsTimeout(err) {
			return fmt.Errorf("request timed out after 60s; the backend may be slow or unreachable: %w", err)
		}
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

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
		fmt.Println("       luma-cli auth login <key> (paste card key or token directly)")
		fmt.Println("       luma-cli auth status      (show login status)")
		return nil
	}

	switch args[0] {
	case "login":
		if len(args) >= 2 {
			// Direct key/token save
			if err := appconfig.SaveCardKey(args[1]); err != nil {
				fmt.Printf("Error: write config: %v\n", err)
				return nil
			}
			fmt.Println("Login saved.")
			return nil
		}
		// Device flow login
		deviceFlowLogin()
		return nil

	case "status":
		cfg := loadConfig()
		if runtimeOpts.JSON {
			data := map[string]any{"logged_in": cfg != nil}
			if cfg != nil {
				data["key"] = appconfig.MaskKey(cfg.CardKey)
			}
			if err := output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: data}); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			return nil
		}
		if cfg == nil {
			fmt.Println("Not logged in. Run: luma-cli auth login")
		} else {
			fmt.Printf("Logged in. Key: %s\n", appconfig.MaskKey(cfg.CardKey))
		}
		return nil

	default:
		fmt.Println("usage: luma-cli auth login [key]")
		fmt.Println("       luma-cli auth status")
	}
	return nil
}

func deviceFlowLogin() {
	base := cloud.BaseURL()

	// 1. Activate device
	type activateResp struct {
		DeviceCode string `json:"device_code"`
		UserCode   string `json:"user_code"`
		VerifyURL  string `json:"verify_url"`
		ExpiresIn  int    `json:"expires_in"`
	}
	var act activateResp
	if err := postJSON(base+"/api/auth/device/activate", nil, &act); err != nil {
		fmt.Printf("Error: activate device: %v\n", err)
		return
	}

	// 2. Open browser
	fmt.Printf("Opening browser to: %s\n", act.VerifyURL)
	fmt.Printf("Your code: %s\n\n", act.UserCode)
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
		}
		var pr pollResp
		if err := postJSON(base+"/api/auth/device/poll", map[string]string{
			"device_code": act.DeviceCode,
		}, &pr); err != nil {
			continue
		}

		if pr.Status == "success" {
			fmt.Println("\nAuthorized!")
			if err := appconfig.SaveCardKey(pr.AccessToken); err != nil {
				fmt.Printf("Error: save token: %v\n", err)
				return
			}
			fmt.Println("Login saved.")
			return
		}
		if pr.Status == "expired" {
			fmt.Println("\nAuthorization expired. Run luma-cli auth login again.")
			return
		}
	}
	fmt.Println("\nAuthorization timed out. Run luma-cli auth login again.")
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
	client := &http.Client{Timeout: 10 * time.Second}
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

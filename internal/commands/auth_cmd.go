package commands

import (
	"fmt"
	"os"

	appconfig "github.com/luma-cli/lumer-cli/internal/config"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func cmdAuth(args []string) {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli auth login <card_key>")
		return
	}
	if args[0] == "login" && len(args) >= 2 {
		if err := appconfig.SaveCardKey(args[1]); err != nil {
			fmt.Printf("Error: write config: %v\n", err)
			return
		}
		fmt.Println("Login saved.")
		return
	}
	if args[0] == "status" {
		cfg := loadConfig()
		if runtimeOpts.JSON {
			data := map[string]any{
				"logged_in": cfg != nil,
			}
			if cfg != nil {
				data["key"] = appconfig.MaskKey(cfg.CardKey)
			}
			if err := output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: data}); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			return
		}
		if cfg == nil {
			fmt.Println("Not logged in. Run: luma-cli auth login <card_key>")
		} else {
			fmt.Printf("Logged in. Key: %s\n", appconfig.MaskKey(cfg.CardKey))
		}
		return
	}
	fmt.Println("usage: luma-cli auth login <card_key>")
	fmt.Println("       luma-cli auth status")
}

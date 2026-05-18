package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
)

func cmdAsset(args []string) {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli asset upload <file> [--group <name>]")
		fmt.Println("       luma-cli asset list [group]")
		return
	}

	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}

	switch args[0] {
	case "upload":
		parsed := cmdutil.Parse(args[1:])
		filePath := parsed.Pos(0)
		if filePath == "" {
			fmt.Println("usage: luma-cli asset upload <file> [--group <name>]")
			return
		}
		group := parsed.String("group", "default")
		objectKey, err := cloud.UploadFile(filePath, cfg.CardKey, group)
		if err != nil {
			fmt.Printf("Error: upload failed: %v\n", err)
			return
		}
		fmt.Printf("Uploaded: %s\n", objectKey)

	case "list":
		parsed := cmdutil.Parse(args[1:])
		group := "default"
		if parsed.Pos(0) != "" {
			group = parsed.Pos(0)
		}
		items, err := cloud.AssetList(group, cfg.CardKey)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
		if len(items) == 0 {
			fmt.Println("No assets found.")
			return
		}
		fmt.Printf("%-8s %-20s %s\n", "TYPE", "NAME", "OBJECT_KEY")
		fmt.Println(strings.Repeat("-", 80))
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				data, _ := json.Marshal(item)
				fmt.Println(string(data))
				continue
			}
			rtype, _ := m["resource_type"].(string)
			key, _ := m["object_key"].(string)
			name := atom.AssetFriendlyName(key)
			fmt.Printf("%-8s %-20s %s\n", rtype, name, key)
		}

	default:
		fmt.Printf("unknown asset subcommand: %s\n", args[0])
	}
}

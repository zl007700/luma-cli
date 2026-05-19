package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

type assetView struct {
	Type      string `json:"type,omitempty"`
	Name      string `json:"name"`
	Group     string `json:"group,omitempty"`
	ObjectKey string `json:"object_key,omitempty"`
}

func cmdAsset(args []string) {
	if len(args) < 1 {
		printAssetUsage()
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
		verbose := parsed.Has("verbose")
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
		view := assetView{
			Name:      atom.AssetFriendlyName(objectKey),
			Group:     group,
			ObjectKey: objectKey,
		}
		if runtimeOpts.JSON {
			if !verbose {
				view.ObjectKey = ""
			}
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: view})
			return
		}
		fmt.Printf("Uploaded: %s\n", view.Name)
		fmt.Printf("Group: %s\n", group)
		if verbose {
			fmt.Printf("Object key: %s\n", objectKey)
		}

	case "list":
		parsed := cmdutil.Parse(args[1:])
		verbose := parsed.Has("verbose")
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
		views := make([]assetView, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			rtype, _ := m["resource_type"].(string)
			key := atom.ResourceKeyFromMap(m, cfg.CardKey)
			view := assetView{
				Type:      rtype,
				Name:      atom.AssetFriendlyName(key),
				Group:     group,
				ObjectKey: key,
			}
			if !verbose {
				view.ObjectKey = ""
			}
			views = append(views, view)
		}
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"items": views}})
			return
		}
		if verbose {
			fmt.Printf("%-8s %-20s %s\n", "TYPE", "NAME", "OBJECT_KEY")
			fmt.Println(strings.Repeat("-", 80))
			for _, item := range views {
				fmt.Printf("%-8s %-20s %s\n", item.Type, item.Name, item.ObjectKey)
			}
			return
		}
		fmt.Printf("%-8s %s\n", "TYPE", "NAME")
		fmt.Println(strings.Repeat("-", 32))
		for _, item := range views {
			fmt.Printf("%-8s %s\n", item.Type, item.Name)
		}

	default:
		fmt.Printf("unknown asset subcommand: %s\n", args[0])
	}
}

func printAssetUsage() {
	fmt.Println("usage: luma-cli asset upload <file> [--group <name>] [--verbose]")
	fmt.Println("       luma-cli asset list [group] [--verbose]")
}

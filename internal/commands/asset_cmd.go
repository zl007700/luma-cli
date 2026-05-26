package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

func cmdAsset(args []string) error {
	if len(args) < 1 {
		printAssetUsage()
		return nil
	}

	cfg, err := requireConfig()
	if err != nil {
		return err
	}

	switch args[0] {
	case "upload":
		parsed := cmdutil.Parse(args[1:])
		verbose := parsed.Has("verbose")
		filePath := parsed.Pos(0)
		if filePath == "" {
			fmt.Println("usage: luma-cli asset upload <file> [--group <name>]")
			return nil
		}
		group := parsed.String("group", "default")
		objectKey, err := cloud.UploadFile(filePath, cfg.CardKey, group)
		if err != nil {
			return output.ErrNetwork(fmt.Sprintf("upload failed: %v\n", err))
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
			return nil
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
			return output.ErrSystem(fmt.Sprintf("%v\n", err))
		}
		if len(items) == 0 {
			fmt.Println("No assets found.")
			return nil
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
			return nil
		}
		if verbose {
			fmt.Printf("%-8s %-20s %s\n", "TYPE", "NAME", "OBJECT_KEY")
			fmt.Println(strings.Repeat("-", 80))
			for _, item := range views {
				fmt.Printf("%-8s %-20s %s\n", item.Type, item.Name, item.ObjectKey)
			}
			return nil
		}
		fmt.Printf("%-8s %s\n", "TYPE", "NAME")
		fmt.Println(strings.Repeat("-", 32))
		for _, item := range views {
			fmt.Printf("%-8s %s\n", item.Type, item.Name)
		}

	case "understand":
		parsed := cmdutil.Parse(args[1:])
		group := parsed.String("group", "default")
		objectName := parsed.String("object", "")
		if objectName == "" {
			objectName = parsed.Pos(0)
		}
		if objectName == "" {
			fmt.Println("usage: luma-cli asset understand <object_name> [--group <name>] [--output meta.json]")
			return nil
		}
		result, err := cloud.UnderstandResource(group, objectName, cfg.CardKey)
		if err != nil {
			return output.ErrNetwork(fmt.Sprintf("understand failed: %v\n", err))
		}
		outputPath := parsed.String("output", "")
		if outputPath != "" {
			abs, err := absoluteOutputPath(outputPath)
			if err != nil {
				return output.ErrValidation(fmt.Sprintf("bad output path: %v\n", err))
			}
			if err := os.MkdirAll(filepath.Dir(abs), 0755); err != nil {
				return output.ErrSystem(fmt.Sprintf("create output dir failed: %v\n", err))
			}
			data, _ := json.MarshalIndent(result, "", "  ")
			if err := os.WriteFile(abs, data, 0644); err != nil {
				return output.ErrSystem(fmt.Sprintf("write output failed: %v\n", err))
			}
			result["output_path"] = abs
		}
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: result})
			return nil
		}
		fmt.Println("Analyzed resource.")
		if outputPath != "" {
			fmt.Printf("Saved to: %s\n", result["output_path"])
		}

	default:
		fmt.Printf("unknown asset subcommand: %s\n", args[0])
	}
	return nil
}

func printAssetUsage() {
	fmt.Println("usage: luma-cli asset upload <file> [--group <name>] [--verbose]")
	fmt.Println("       luma-cli asset list [group] [--verbose]")
	fmt.Println("       luma-cli asset understand <object_name> [--group <name>] [--output meta.json]")
}

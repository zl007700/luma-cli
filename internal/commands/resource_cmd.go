package commands

import (
	"encoding/json"
	"fmt"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/clientruntime"
)

func cmdResource(args []string) {
	if len(args) < 1 {
		printResourceUsage()
		return
	}
	switch args[0] {
	case "list":
		cmdResourceList(args[1:])
	case "cache":
		cmdResourceCache(args[1:])
	case "path":
		cmdResourcePath(args[1:])
	default:
		fmt.Printf("unknown resource subcommand: %s\n\n", args[0])
		printResourceUsage()
	}
}

func cmdResourceList(args []string) {
	resourceType := ""
	tag := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) {
				resourceType = args[i+1]
				i++
			}
		case "--tag":
			if i+1 < len(args) {
				tag = args[i+1]
				i++
			}
		default:
			if resourceType == "" {
				resourceType = args[i]
			}
		}
	}
	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}
	items, err := cloud.ListClientResources(resourceType, tag, cfg.CardKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if runtimeOpts.JSON {
		data, _ := json.MarshalIndent(map[string]any{"items": items}, "", "  ")
		fmt.Println(string(data))
		return
	}
	if len(items) == 0 {
		fmt.Println("No client resources found.")
		return
	}
	fmt.Printf("%-24s %-10s %-12s %s\n", "ID", "TYPE", "VERSION", "NAME")
	for _, item := range items {
		fmt.Printf("%-24s %-10s %-12s %s\n", item.ID, item.Type, item.Version, item.Name)
	}
}

func cmdResourceCache(args []string) {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli resource cache <resource_id>")
		return
	}
	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}
	cached, err := clientruntime.CacheResource(cfg.CardKey, args[0])
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if runtimeOpts.JSON {
		data, _ := json.MarshalIndent(cached, "", "  ")
		fmt.Println(string(data))
		return
	}
	fmt.Printf("Cached %s\n", cached.ID)
	fmt.Printf("Path: %s\n", cached.Path)
}

func cmdResourcePath(args []string) {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli resource path <resource_id>")
		return
	}
	cached, err := clientruntime.CurrentResource(args[0])
	if err != nil {
		fmt.Printf("Error: resource not cached: %s\n", args[0])
		return
	}
	if runtimeOpts.JSON {
		data, _ := json.MarshalIndent(cached, "", "  ")
		fmt.Println(string(data))
		return
	}
	fmt.Println(cached.Path)
}

func printResourceUsage() {
	fmt.Println("luma-cli resource <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  list [type] [--tag <tag>]       List cloud-managed client resources")
	fmt.Println("  cache <resource_id>             Download and cache one signed resource")
	fmt.Println("  path <resource_id>              Print cached resource path")
}

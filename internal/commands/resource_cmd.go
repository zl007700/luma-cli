package commands

import (
	"github.com/luma-cli/lumer-cli/internal/output"
	"encoding/json"
	"fmt"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/clientruntime"
)

func cmdResource(args []string) error {
	if len(args) < 1 {
		printResourceUsage()
		return nil
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
	return nil
}

func cmdResourceList(args []string) error {
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
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	items, err := cloud.ListClientResources(resourceType, tag, cfg.CardKey)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	if runtimeOpts.JSON {
		data, _ := json.MarshalIndent(map[string]any{"items": items}, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	if len(items) == 0 {
		fmt.Println("No client resources found.")
		return nil
	}
	fmt.Printf("%-24s %-10s %-12s %s\n", "ID", "TYPE", "VERSION", "NAME")
	for _, item := range items {
		fmt.Printf("%-24s %-10s %-12s %s\n", item.ID, item.Type, item.Version, item.Name)
	}
	return nil
}

func cmdResourceCache(args []string) error {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli resource cache <resource_id>")
		return nil
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	cached, err := clientruntime.CacheResource(cfg.CardKey, args[0])
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	if runtimeOpts.JSON {
		data, _ := json.MarshalIndent(cached, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Cached %s\n", cached.ID)
	fmt.Printf("Path: %s\n", cached.Path)
	return nil
}

func cmdResourcePath(args []string) error {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli resource path <resource_id>")
		return nil
	}
	cached, err := clientruntime.CurrentResource(args[0])
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("resource not cached: %s\n", args[0]))
	}
	if runtimeOpts.JSON {
		data, _ := json.MarshalIndent(cached, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	fmt.Println(cached.Path)
	return nil
}

func printResourceUsage() {
	fmt.Println("luma-cli resource <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  list [type] [--tag <tag>]       List cloud-managed client resources")
	fmt.Println("  cache <resource_id>             Download and cache one signed resource")
	fmt.Println("  path <resource_id>              Print cached resource path")
}

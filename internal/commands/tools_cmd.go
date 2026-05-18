package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/luma-cli/lumer-cli/internal/output"
	"github.com/luma-cli/lumer-cli/shortcuts/registry"
)

func cmdTools(args []string) {
	if len(args) < 1 {
		printToolsUsage()
		return
	}

	switch args[0] {
	case "list", "ls":
		cmdToolsList()
	case "describe", "show":
		if len(args) < 2 {
			fmt.Println("usage: luma-cli tools describe <tool_id>")
			return
		}
		cmdToolsDescribe(args[1])
	default:
		fmt.Printf("unknown tools subcommand: %s\n", args[0])
		printToolsUsage()
	}
}

func printToolsUsage() {
	fmt.Println("luma-cli tools <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  list                 List agent-callable atomic tools")
	fmt.Println("  describe <tool_id>   Show one atomic tool contract")
}

func cmdToolsList() {
	items := registry.List()
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"tools": items}})
		return
	}

	fmt.Printf("%-22s %-10s %-8s %s\n", "ID", "SERVICE", "RISK", "DESCRIPTION")
	for _, item := range items {
		fmt.Printf("%-22s %-10s %-8s %s\n", item.ID, item.Service, item.Risk, item.Description)
	}
}

func cmdToolsDescribe(id string) {
	item, ok := registry.Find(id)
	if !ok {
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: false, Code: "tool_not_found", Error: "tool not found"})
			return
		}
		fmt.Printf("Error: tool not found: %s\n", id)
		return
	}

	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: item})
		return
	}

	data, _ := json.MarshalIndent(item, "", "  ")
	fmt.Println(string(data))
}

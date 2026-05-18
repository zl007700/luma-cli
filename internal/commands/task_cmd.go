package commands

import (
	"encoding/json"
	"fmt"

	"github.com/luma-cli/lumer-cli/cloud"
)

func cmdTask(args []string) {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli task status <task_id>")
		fmt.Println("       luma-cli task list")
		return
	}

	switch args[0] {
	case "status":
		if len(args) < 2 {
			fmt.Println("usage: luma-cli task status <task_id>")
			return
		}
		cmdTaskStatus(args[1])
	default:
		fmt.Printf("unknown task subcommand: %s\n", args[0])
	}
}

func cmdTaskStatus(taskID string) {
	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}

	status, err := cloud.PollTask(taskID, cfg.CardKey)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	data, _ := json.MarshalIndent(status, "", "  ")
	fmt.Println(string(data))
}

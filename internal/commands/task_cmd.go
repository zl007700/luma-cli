package commands

import (
	"github.com/luma-cli/lumer-cli/internal/output"
	"encoding/json"
	"fmt"

	"github.com/luma-cli/lumer-cli/cloud"
)

func cmdTask(args []string) error {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli task status <task_id>")
		fmt.Println("       luma-cli task list")
		return nil
	}

	switch args[0] {
	case "status":
		if len(args) < 2 {
			fmt.Println("usage: luma-cli task status <task_id>")
			return nil
		}
		cmdTaskStatus(args[1])
	default:
		fmt.Printf("unknown task subcommand: %s\n", args[0])
	}
	return nil
}

func cmdTaskStatus(taskID string) error {
	cfg := loadConfig()
	if cfg == nil {
		return output.ErrAuth("not logged in. Run: luma-cli auth login <card_key>")
	}

	status, err := cloud.PollTask(taskID, cfg.CardKey)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}

	data, _ := json.MarshalIndent(status, "", "  ")
	fmt.Println(string(data))
	return nil
}

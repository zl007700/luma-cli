package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/luma-cli/lumer-cli/internal/clientruntime"
)

func cmdRuntime(args []string) {
	if len(args) < 1 {
		printRuntimeUsage()
		return
	}
	switch args[0] {
	case "install":
		cmdRuntimeInstall(args[1:])
	case "path":
		cmdRuntimePath(args[1:])
	case "doctor":
		cmdRuntimeDoctor()
	default:
		fmt.Printf("unknown runtime subcommand: %s\n\n", args[0])
		printRuntimeUsage()
	}
}

func cmdRuntimeInstall(args []string) {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli runtime install ffmpeg [--version <version>]")
		return
	}
	name := args[0]
	version := "auto"
	for i := 1; i < len(args); i++ {
		if args[i] == "--version" && i+1 < len(args) {
			version = args[i+1]
			i++
		}
	}
	cfg := loadConfig()
	if cfg == nil {
		fmt.Println("Error: not logged in. Run: luma-cli auth login <card_key>")
		return
	}
	fmt.Printf("Installing runtime: %s\n", name)
	installed, err := clientruntime.InstallRuntime(cfg.CardKey, name, version)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	if runtimeOpts.JSON {
		data, _ := json.MarshalIndent(installed, "", "  ")
		fmt.Println(string(data))
		return
	}
	fmt.Printf("Installed %s %s\n", installed.Name, installed.Version)
	fmt.Printf("Executable: %s\n", installed.ExecutablePath)
	if installed.FFProbePath != "" {
		fmt.Printf("FFprobe: %s\n", installed.FFProbePath)
	}
}

func cmdRuntimePath(args []string) {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli runtime path ffmpeg")
		return
	}
	installed, err := clientruntime.CurrentRuntime(args[0])
	if err != nil {
		fmt.Printf("Error: runtime not installed: %s\n", args[0])
		return
	}
	if runtimeOpts.JSON {
		data, _ := json.MarshalIndent(installed, "", "  ")
		fmt.Println(string(data))
		return
	}
	fmt.Println(installed.ExecutablePath)
}

func cmdRuntimeDoctor() {
	for _, name := range []string{"ffmpeg"} {
		installed, err := clientruntime.CurrentRuntime(name)
		if err != nil {
			fmt.Printf("%s: not installed\n", name)
			continue
		}
		if _, err := os.Stat(installed.ExecutablePath); err != nil {
			fmt.Printf("%s: missing (%s)\n", name, installed.ExecutablePath)
			continue
		}
		if installed.FFProbePath != "" {
			if _, err := os.Stat(installed.FFProbePath); err != nil {
				fmt.Printf("%s: ffprobe missing (%s)\n", name, installed.FFProbePath)
				continue
			}
		}
		fmt.Printf("%s: ok (%s)\n", name, installed.ExecutablePath)
	}
}

func printRuntimeUsage() {
	fmt.Println("luma-cli runtime <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  install ffmpeg [--version <version>]  Install a signed runtime from Luma cloud")
	fmt.Println("  path ffmpeg                           Print installed runtime executable path")
	fmt.Println("  doctor                                Check local runtime status")
}

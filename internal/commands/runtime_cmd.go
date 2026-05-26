package commands

import (
	"github.com/luma-cli/lumer-cli/internal/output"
	"encoding/json"
	"fmt"
	"os"

	"github.com/luma-cli/lumer-cli/internal/clientruntime"
)

func cmdRuntime(args []string) error {
	if len(args) < 1 {
		printRuntimeUsage()
		return nil
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
	return nil
}

func cmdRuntimeInstall(args []string) error {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli runtime install ffmpeg [--version <version>]")
		return nil
	}
	name := args[0]
	version := "auto"
	for i := 1; i < len(args); i++ {
		if args[i] == "--version" && i+1 < len(args) {
			version = args[i+1]
			i++
		}
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	fmt.Printf("Installing runtime: %s\n", name)
	installed, err := clientruntime.InstallRuntime(cfg.CardKey, name, version)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("%v\n", err))
	}
	if runtimeOpts.JSON {
		data, _ := json.MarshalIndent(installed, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	fmt.Printf("Installed %s %s\n", installed.Name, installed.Version)
	fmt.Printf("Executable: %s\n", installed.ExecutablePath)
	if installed.FFProbePath != "" {
		fmt.Printf("FFprobe: %s\n", installed.FFProbePath)
	}
	return nil
}

func cmdRuntimePath(args []string) error {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli runtime path ffmpeg")
		return nil
	}
	installed, err := clientruntime.CurrentRuntime(args[0])
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("runtime not installed: %s\n", args[0]))
	}
	if runtimeOpts.JSON {
		data, _ := json.MarshalIndent(installed, "", "  ")
		fmt.Println(string(data))
		return nil
	}
	fmt.Println(installed.ExecutablePath)
	return nil
}

func cmdRuntimeDoctor() error {
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
	return nil
}

func printRuntimeUsage() {
	fmt.Println("luma-cli runtime <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  install ffmpeg [--version <version>]  Install a signed runtime from Luma cloud")
	fmt.Println("  path ffmpeg                           Print installed runtime executable path")
	fmt.Println("  doctor                                Check local runtime status")
}

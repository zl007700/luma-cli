package commands

import (
	"fmt"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/cmdutil"
)

func cmdMaterialLibrary(raw []string) {
	if len(raw) < 1 {
		printMaterialLibraryUsage()
		return
	}
	switch raw[0] {
	case "path":
		fmt.Println(defaultMaterialLibraryRoot())
	case "import":
		cmdMaterialLibraryImport(raw[1:])
	default:
		printMaterialLibraryUsage()
	}
}

func cmdMaterialLibraryImport(raw []string) {
	args := cmdutil.Parse(raw)
	sourcePath := strings.TrimSpace(args.Pos(0))
	if sourcePath == "" {
		sourcePath = strings.TrimSpace(args.String("source", ""))
	}
	if sourcePath == "" {
		fmt.Println("usage: luma-cli material library import <group_dir> [--name <group_name>] [--replace]")
		return
	}
	name := strings.TrimSpace(args.String("name", ""))
	replace, err := args.Bool("replace", false)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	imported, err := importMaterialGroup(sourcePath, name, replace)
	if err != nil {
		fmt.Printf("Error: import material group failed: %v\n", err)
		return
	}
	writeSimpleResult(map[string]any{"group_path": imported})
}

func printMaterialLibraryUsage() {
	fmt.Println("luma-cli material library <subcommand>")
	fmt.Println("  path                                            Show default local material library path")
	fmt.Println("  import <group_dir> [--name <group_name>] [--replace]  Copy a material group into the default library")
}

package commands

import (
	"fmt"
	"github.com/luma-cli/lumer-cli/internal/output"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/cmdutil"
)

func cmdMaterialLibrary(raw []string) error {
	if len(raw) < 1 {
		printMaterialLibraryUsage()
		return nil
	}
	switch raw[0] {
	case "path":
		fmt.Println(defaultMaterialLibraryRoot())
	case "import":
		cmdMaterialLibraryImport(raw[1:])
	default:
		printMaterialLibraryUsage()
	}
	return nil
}

func cmdMaterialLibraryImport(raw []string) error {
	args := cmdutil.Parse(raw)
	sourcePath := strings.TrimSpace(args.Pos(0))
	if sourcePath == "" {
		sourcePath = strings.TrimSpace(args.String("source", ""))
	}
	if sourcePath == "" {
		fmt.Println("usage: luma-cli material library import <group_dir> [--name <group_name>] [--replace]")
		return nil
	}
	name := strings.TrimSpace(args.String("name", ""))
	replace, err := args.Bool("replace", false)
	if err != nil {
		return output.ErrSystem("%v\n", err)
	}
	imported, err := importMaterialGroup(sourcePath, name, replace)
	if err != nil {
		return output.ErrSystem("import material group failed: %v\n", err)
	}
	writeSimpleResult(map[string]any{"group_path": imported})
	return nil
}

func printMaterialLibraryUsage() {
	fmt.Println("luma-cli material library <subcommand>")
	fmt.Println("  path                                            Show default local material library path")
	fmt.Println("  import <group_dir> [--name <group_name>] [--replace]  Copy a material group into the default library")
}

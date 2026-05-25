package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func cmdMaterialGroup(raw []string) {
	if len(raw) < 1 {
		printMaterialUsage()
		return
	}
	switch raw[0] {
	case "list", "ls":
		cmdMaterialGroupList(raw[1:])
	case "describe":
		cmdMaterialGroupDescribe(raw[1:])
	default:
		printMaterialUsage()
	}
}

func cmdMaterialGroupList(raw []string) {
	args := cmdutil.Parse(raw)
	rootPath := strings.TrimSpace(args.Pos(0))
	if rootPath == "" {
		rootPath = strings.TrimSpace(args.String("root", ""))
	}
	if rootPath == "" {
		rootPath = defaultMaterialGroupsRoot()
	}
	groups, err := listMaterialGroups(rootPath)
	if err != nil {
		fmt.Printf("Error: list material groups failed: %v\n", err)
		return
	}
	outputPath := strings.TrimSpace(args.String("output", ""))
	savedPath := ""
	if outputPath != "" {
		abs, err := absoluteOutputPath(outputPath)
		if err != nil {
			fmt.Printf("Error: bad output path: %v\n", err)
			return
		}
		if err := writeJSONFile(abs, map[string]any{"groups": groups}); err != nil {
			fmt.Printf("Error: write output failed: %v\n", err)
			return
		}
		savedPath = abs
		recordProjectArtifact("material_groups", savedPath, "material.group.list")
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"groups": groups, "output_path": savedPath}})
		return
	}
	fmt.Printf("Material groups: %d\n", len(groups))
	for _, group := range groups {
		name := group.Name
		if name == "" {
			name = group.ID
		}
		fmt.Printf("  %-24s %4d materials  %s\n", name, group.MaterialCount, group.Path)
	}
	if savedPath != "" {
		fmt.Printf("Saved to: %s\n", savedPath)
	}
}

func cmdMaterialGroupDescribe(raw []string) {
	args := cmdutil.Parse(raw)
	groupPath := strings.TrimSpace(args.Pos(0))
	if groupPath == "" {
		groupPath = strings.TrimSpace(args.String("group", ""))
	}
	if groupPath == "" {
		fmt.Println("usage: luma-cli material group describe <group_name_or_dir> [--output materials.json]")
		fmt.Printf("Default library: %s\n", defaultMaterialGroupsRoot())
		return
	}
	groupPath = resolveMaterialGroupPath(groupPath)
	group, err := describeMaterialGroup(groupPath)
	if err != nil {
		fmt.Printf("Error: describe material group failed: %v\n", err)
		return
	}
	outputPath := strings.TrimSpace(args.String("output", "materials.json"))
	savedPath := ""
	if outputPath != "" {
		abs, err := absoluteOutputPath(outputPath)
		if err != nil {
			fmt.Printf("Error: bad output path: %v\n", err)
			return
		}
		groupView := group
		groupView.Materials = nil
		if err := writeJSONFile(abs, map[string]any{"group": groupView, "materials": group.Materials}); err != nil {
			fmt.Printf("Error: write output failed: %v\n", err)
			return
		}
		savedPath = abs
		recordProjectArtifact("materials", savedPath, "material.group.describe")
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"group": group, "materials": group.Materials, "output_path": savedPath}})
		return
	}
	fmt.Printf("Group: %s\n", group.ID)
	fmt.Printf("Materials: %d\n", len(group.Materials))
	if savedPath != "" {
		fmt.Printf("Saved to: %s\n", savedPath)
	}
}

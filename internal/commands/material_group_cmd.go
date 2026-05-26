package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func cmdMaterialGroup(raw []string) error {
	if len(raw) < 1 {
		printMaterialUsage()
		return nil
	}
	switch raw[0] {
	case "list", "ls":
		cmdMaterialGroupList(raw[1:])
	case "describe":
		cmdMaterialGroupDescribe(raw[1:])
	default:
		printMaterialUsage()
	}
	return nil
}

func cmdMaterialGroupList(raw []string) error {
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
		return output.ErrSystem(fmt.Sprintf("list material groups failed: %v\n", err))
	}
	outputPath := strings.TrimSpace(args.String("output", ""))
	savedPath := ""
	if outputPath != "" {
		abs, err := ensureOutputDir(outputPath)
		if err != nil {
			return nil
		}
		if err := writeJSONFile(abs, map[string]any{"groups": groups}); err != nil {
			return output.ErrSystem(fmt.Sprintf("write output failed: %v\n", err))
		}
		savedPath = abs
		recordProjectArtifact("material_groups", savedPath, "material.group.list")
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"groups": groups, "output_path": savedPath}})
		return nil
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
	return nil
}

func cmdMaterialGroupDescribe(raw []string) error {
	args := cmdutil.Parse(raw)
	groupPath := strings.TrimSpace(args.Pos(0))
	if groupPath == "" {
		groupPath = strings.TrimSpace(args.String("group", ""))
	}
	if groupPath == "" {
		fmt.Println("usage: luma-cli material group describe <group_name_or_dir> [--output materials.json]")
		fmt.Printf("Default library: %s\n", defaultMaterialGroupsRoot())
		return nil
	}
	groupPath = resolveMaterialGroupPath(groupPath)
	group, err := describeMaterialGroup(groupPath)
	if err != nil {
		return output.ErrSystem(fmt.Sprintf("describe material group failed: %v\n", err))
	}
	outputPath := strings.TrimSpace(args.String("output", "materials.json"))
	savedPath := ""
	if outputPath != "" {
		abs, err := ensureOutputDir(outputPath)
		if err != nil {
			return nil
		}
		groupView := group
		groupView.Materials = nil
		if err := writeJSONFile(abs, map[string]any{"group": groupView, "materials": group.Materials}); err != nil {
			return output.ErrSystem(fmt.Sprintf("write output failed: %v\n", err))
		}
		savedPath = abs
		recordProjectArtifact("materials", savedPath, "material.group.describe")
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"group": group, "materials": group.Materials, "output_path": savedPath}})
		return nil
	}
	fmt.Printf("Group: %s\n", group.ID)
	fmt.Printf("Materials: %d\n", len(group.Materials))
	if savedPath != "" {
		fmt.Printf("Saved to: %s\n", savedPath)
	}
	return nil
}

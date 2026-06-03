package commands

import (
	"fmt"
	"github.com/luma-cli/lumer-cli/internal/output"
	"os"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/skillsync"
)

func cmdUpdate(args []string) error {
	opts := parseUpdateOptions(args)
	target := opts.Version
	installVersion := opts.Version
	registry := opts.Registry
	if target == "" || target == "latest" {
		resolved, err := skillsync.LatestNpmVersion()
		if err != nil {
			fmt.Printf("Warning: failed to resolve latest npm version: %v\n", err)
			target = "latest"
		} else {
			target = resolved
		}
		if opts.Registry == "" {
			if official, err := skillsync.LatestNpmVersionFromRegistry(skillsync.OfficialNpmRegistry); err == nil && skillsync.CompareSemver(official, target) > 0 {
				fmt.Printf("@lumageo/luma-cli latest on current npm registry is %s, official registry latest is %s. Using official registry for this update.\n", target, official)
				target = official
				installVersion = official
				registry = skillsync.OfficialNpmRegistry
			}
		}
	}
	if opts.Version != "" && opts.Version != "latest" {
		target = opts.Version
		installVersion = opts.Version
	}

	fmt.Printf("Updating @lumageo/luma-cli via npm (%s)...\n", target)
	if err := skillsync.RunNpmInstall(skillsync.NpmInstallOptions{
		Version:  installVersion,
		Registry: registry,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
	}); err != nil {
		return output.ErrSystem("failed to update luma-cli: %v\n", err)
	}

	if opts.SkipSkills {
		fmt.Println("Skipped skills sync.")
		return nil
	}

	syncOpts := skillsync.SyncOptions{
		Source:  opts.Source,
		Global:  true,
		Yes:     true,
		Version: version,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
	if syncOpts.Source == "" {
		syncOpts.Source = skillsync.SourceFromEnv()
	}
	fmt.Printf("Syncing Luma skills from %s...\n", syncOpts.Source)
	if err := skillsync.RunSkillsAdd(syncOpts); err != nil {
		fmt.Printf("Error: updated CLI, but failed to sync skills: %v\n", err)
		fmt.Println("Try: luma-cli skills sync")
		return nil
	}
	stampVersion := version
	if target != "" && target != "latest" {
		stampVersion = target
	}
	if stampVersion == "dev" || stampVersion == "local" {
		stampVersion = target
	}
	if err := skillsync.WriteStamp(stampVersion, syncOpts.Source); err != nil {
		fmt.Printf("Warning: synced skills, but failed to write stamp: %v\n", err)
		return nil
	}
	fmt.Println("Done. luma-cli and Luma skills are updated.")
	return nil
}

type updateOptions struct {
	Version    string
	Source     string
	Registry   string
	SkipSkills bool
}

func parseUpdateOptions(args []string) updateOptions {
	var opts updateOptions
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--version":
			if i+1 < len(args) {
				opts.Version = strings.TrimSpace(args[i+1])
				i++
			}
		case "--source":
			if i+1 < len(args) {
				opts.Source = strings.TrimSpace(args[i+1])
				i++
			}
		case "--registry":
			if i+1 < len(args) {
				opts.Registry = strings.TrimSpace(args[i+1])
				i++
			}
		case "--skip-skills":
			opts.SkipSkills = true
		}
	}
	return opts
}

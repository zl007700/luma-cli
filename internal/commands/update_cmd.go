package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/skillsync"
)

func cmdUpdate(args []string) {
	opts := parseUpdateOptions(args)
	target := opts.Version
	if target == "" || target == "latest" {
		resolved, err := skillsync.LatestNpmVersion()
		if err != nil {
			fmt.Printf("Warning: failed to resolve latest npm version: %v\n", err)
			target = "latest"
		} else {
			target = resolved
		}
	}
	if opts.Version != "" && opts.Version != "latest" {
		target = opts.Version
	}

	fmt.Printf("Updating luma-cli via npm (%s)...\n", target)
	if err := skillsync.RunNpmInstall(skillsync.NpmInstallOptions{
		Version: opts.Version,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}); err != nil {
		fmt.Printf("Error: failed to update luma-cli: %v\n", err)
		return
	}

	if opts.SkipSkills {
		fmt.Println("Skipped skills sync.")
		return
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
		return
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
		return
	}
	fmt.Println("Done. luma-cli and Luma skills are updated.")
}

type updateOptions struct {
	Version    string
	Source     string
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
		case "--skip-skills":
			opts.SkipSkills = true
		}
	}
	return opts
}

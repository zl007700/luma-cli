package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/output"
	"github.com/luma-cli/lumer-cli/internal/skillsync"
)

func cmdSkills(args []string) error {
	if len(args) < 1 {
		printSkillsUsage()
		return nil
	}

	switch args[0] {
	case "list", "ls":
		cmdSkillsList()
	case "sync", "install":
		cmdSkillsSync(args[1:])
	case "status":
		cmdSkillsStatus()
	default:
		fmt.Printf("unknown skills subcommand: %s\n", args[0])
		printSkillsUsage()
	}
	return nil
}

func printSkillsUsage() {
	fmt.Println("luma-cli skills <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  list                                    List Luma skill names")
	fmt.Println("  sync [--source <source>] [-s <skill>]  Install or update Luma skills")
	fmt.Println("  install [--source <source>] [-s <skill>]")
	fmt.Println("  status                                  Show local skills sync status")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  luma-cli skills list")
	fmt.Println("  luma-cli skills sync")
	fmt.Println("  luma-cli skills sync -s luma-workflow-viral-remix")
}

func cmdSkillsList() error {
	items := []map[string]string{
		{"name": "luma-shared", "layer": "shared", "description": "Common Luma agent rules"},
		{"name": "luma-maintenance", "layer": "maintenance", "description": "Update Luma CLI and sync agent skills"},
		{"name": "luma-profile-onboarding", "layer": "capability", "description": "Create a profile via avatar-persona and bind media defaults"},
		{"name": "luma-material", "layer": "capability", "description": "Local material groups, search, and PIP matching"},
		{"name": "luma-original-script", "layer": "workflow", "description": "Run the avatar-persona-based original script pipeline"},
		{"name": "luma-assets", "layer": "capability", "description": "Cloud assets and reusable resources"},
		{"name": "luma-digital-human", "layer": "capability", "description": "Voice clone, TTS, avatar, and lip-sync"},
		{"name": "luma-ppt-video", "layer": "capability", "description": "Render align.json into PPT-style MP4, with optional avatar PiP"},
		{"name": "luma-subtitle", "layer": "capability", "description": "Subtitle generation and rendering"},
		{"name": "luma-workflow-original-ip-talk", "layer": "workflow", "description": "Original persona-to-video workflow with digital human and PPT visuals"},
		{"name": "luma-workflow-viral-remix", "layer": "workflow", "description": "Research-to-video viral remix workflow"},
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"skills": items}})
		return nil
	}

	fmt.Printf("%-30s %-12s %s\n", "NAME", "LAYER", "DESCRIPTION")
	for _, item := range items {
		fmt.Printf("%-30s %-12s %s\n", item["name"], item["layer"], item["description"])
	}
	return nil
}

func cmdSkillsSync(args []string) error {
	opts := parseSkillsSyncOptions(args)
	if opts.Source == "" {
		opts.Source = skillsync.SourceFromEnv()
	}
	opts.Version = version
	opts.Stdout = os.Stdout
	opts.Stderr = os.Stderr

	fmt.Printf("Syncing Luma skills from %s...\n", opts.Source)
	if opts.Skill != "" {
		fmt.Printf("Skill: %s\n", opts.Skill)
	}

	if err := skillsync.RunSkillsAdd(opts); err != nil {
		return output.ErrSystem("failed to sync skills: %v\n", err)
	}
	removed, cleanupErr := skillsync.CleanupDeprecatedSkills()
	if cleanupErr != nil {
		fmt.Printf("Warning: failed to clean deprecated skills: %v\n", cleanupErr)
	}
	if err := skillsync.WriteStamp(version, opts.Source); err != nil {
		fmt.Printf("Warning: synced skills, but failed to write stamp: %v\n", err)
		return nil
	}
	if len(removed) > 0 {
		fmt.Printf("Removed deprecated skills: %s\n", strings.Join(removed, ", "))
	}
	fmt.Println("Done. Luma skills are synced.")
	return nil
}

func cmdSkillsStatus() error {
	stamp, err := skillsync.ReadStamp()
	if err != nil {
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: false, Code: "skills_status_error", Error: err.Error()})
			return nil
		}
		return output.ErrSystem("%v\n", err)
	}

	data := map[string]any{
		"cli_version": version,
		"synced":      stamp != nil && !skillsync.IsVersionDrift(version, stamp),
		"stamp":       stamp,
		"source":      skillsync.SourceFromEnv(),
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: data})
		return nil
	}

	if stamp == nil {
		fmt.Println("Skills status: no local sync stamp found")
		fmt.Printf("CLI version: %s\n", version)
		fmt.Println("Run: luma-cli skills sync")
		return nil
	}
	fmt.Printf("Skills status: %s\n", statusLabel(!skillsync.IsVersionDrift(version, stamp)))
	fmt.Printf("CLI version: %s\n", version)
	fmt.Printf("Skills version: %s\n", stamp.Version)
	fmt.Printf("Source: %s\n", stamp.Source)
	fmt.Printf("Synced at: %s\n", stamp.SyncedAt.Format("2006-01-02 15:04:05"))
	if skillsync.IsVersionDrift(version, stamp) {
		fmt.Println("Run: luma-cli update")
	}
	return nil
}

func parseSkillsSyncOptions(args []string) skillsync.SyncOptions {
	opts := skillsync.SyncOptions{Global: true, Yes: true}
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--source":
			if i+1 < len(args) {
				opts.Source = strings.TrimSpace(args[i+1])
				i++
			}
		case "-s", "--skill":
			if i+1 < len(args) {
				opts.Skill = strings.TrimSpace(args[i+1])
				i++
			}
		case "--local":
			opts.Global = false
		case "--no-yes":
			opts.Yes = false
		}
	}
	return opts
}

func statusLabel(ok bool) string {
	if ok {
		return "synced"
	}
	return "out of sync"
}

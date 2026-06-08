package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luma-cli/lumer-cli/internal/output"
	"github.com/luma-cli/lumer-cli/project"
)

func cmdProjectContext(args []string) error {
	if len(args) < 1 {
		printProjectContextUsage()
		return nil
	}
	switch args[0] {
	case "snapshot-profile":
		return cmdProjectContextSnapshotProfile(args[1:])
	default:
		printProjectContextUsage()
		return nil
	}
}

func printProjectContextUsage() {
	fmt.Println("luma-cli project context <subcommand>")
	fmt.Println("  snapshot-profile [profile_id] [--project <name>]")
}

func cmdProjectContextSnapshotProfile(args []string) error {
	profileID := ""
	for _, item := range args {
		if !strings.HasPrefix(item, "--") && profileID == "" {
			profileID = strings.TrimSpace(item)
		}
	}
	if profileID == "" {
		state, _ := loadProfileState()
		profileID = state.CurrentProfile
	}
	if profileID == "" {
		return output.ErrValidation("profile id is required")
	}
	view, err := loadProfileView(profileID)
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	p, err := resolveProject(args)
	if err != nil {
		return output.ErrSystem("%v\n", err)
	}
	contextDir := filepath.Join(p.Path, "context")
	profilePath := filepath.Join(contextDir, "profile.snapshot.json")
	extraPath := filepath.Join(contextDir, "profile_extra.snapshot.md")
	if err := writeJSONFile(profilePath, view.lumaProfile); err != nil {
		return output.ErrSystem("write profile snapshot: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(extraPath), 0755); err != nil {
		return output.ErrSystem("create context dir: %v", err)
	}
	if err := os.WriteFile(extraPath, []byte(view.Extra), 0644); err != nil {
		return output.ErrSystem("write profile extra snapshot: %v", err)
	}
	now := time.Now()
	if err := p.AddArtifact(project.Artifact{
		ID:        "profile_snapshot_" + now.Format("20060102150405"),
		Type:      "profile_snapshot",
		Path:      profilePath,
		Step:      "project.context.snapshot-profile",
		Ability:   "project.context.snapshot-profile",
		Outputs:   []string{profilePath, extraPath},
		Status:    "completed",
		CreatedAt: now,
		Meta: map[string]any{
			"profile_id": profileID,
			"extra_path": extraPath,
		},
	}); err != nil {
		return output.ErrSystem("add profile snapshot artifact: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{
			"profile_id": profileID,
			"profile":    profilePath,
			"extra":      extraPath,
			"project":    p.Name,
		}})
		return nil
	}
	fmt.Printf("Profile snapshot saved: %s\n", profilePath)
	fmt.Printf("Profile extra snapshot saved: %s\n", extraPath)
	return nil
}

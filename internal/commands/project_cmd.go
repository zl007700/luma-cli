package commands

import (
	"fmt"
	"github.com/luma-cli/lumer-cli/internal/output"
	"os"
	"path/filepath"
	"time"

	"github.com/luma-cli/lumer-cli/project"
)

func cmdProject(args []string) error {
	if len(args) < 1 {
		printProjectUsage()
		return nil
	}

	switch args[0] {
	case "create":
		cmdProjectCreate(args[1:])
	case "list", "ls":
		cmdProjectList(args[1:])
	case "use":
		cmdProjectUse(args[1:])
	case "info":
		cmdProjectInfo(args[1:])
	case "clean":
		cmdProjectClean(args[1:])
	case "manifest":
		cmdProjectManifest(args[1:])
	case "artifact":
		cmdProjectArtifact(args[1:])
	case "context":
		cmdProjectContext(args[1:])
	default:
		fmt.Printf("unknown project subcommand: %s\n\n", args[0])
		printProjectUsage()
	}
	return nil
}

func printProjectUsage() {
	fmt.Println("luma-cli project <subcommand> [options]")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  create <name> [--dir <path>]   Create a video project")
	fmt.Println("  list                           List projects")
	fmt.Println("  use <name>                     Switch active project")
	fmt.Println("  info                           Show active project details")
	fmt.Println("  clean                          Clean project temp files")
	fmt.Println("  manifest                       Print project manifest JSON")
	fmt.Println("  artifact list                  List project artifacts")
	fmt.Println("  artifact schema                Print artifact schema")
	fmt.Println("  artifact add <path> --type <t> Add an artifact to project manifest")
	fmt.Println("  context snapshot-profile [id]  Snapshot active/cloud profile into this project")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  luma-cli project create my-video")
	fmt.Println("  luma-cli project create my-video --dir ~/videos")
	fmt.Println("  luma-cli project list")
	fmt.Println("  luma-cli project use my-video")
}

func cmdProjectCreate(args []string) error {
	if len(args) < 1 || args[0] == "--help" {
		fmt.Println("usage: luma-cli project create <name> [--dir <path>]")
		fmt.Println("")
		fmt.Println("Creates a video project with standard subdirectories:")
		fmt.Println("  source/    - source media files")
		fmt.Println("  audio/     - extracted or generated audio")
		fmt.Println("  subtitles/ - SRT and ASS subtitle files")
		fmt.Println("  effects/   - effect overlay files")
		fmt.Println("  output/    - final outputs")
		fmt.Println("  tmp/       - temporary files")
		return nil
	}

	name := args[0]
	var basePath string

	for i := 1; i < len(args); i++ {
		if args[i] == "--dir" && i+1 < len(args) {
			basePath = args[i+1]
			i++
		}
	}

	if basePath != "" {
		absPath, err := filepath.Abs(basePath)
		if err != nil {
			return output.ErrValidation("invalid dir path: %v\n", err)
		}
		basePath = absPath
	}

	p, err := project.Create(name, basePath)
	if err != nil {
		return output.ErrSystem("%v\n", err)
	}

	fmt.Printf("Project created: %s\n", p.Path)
	fmt.Println("")
	fmt.Println("Directories:")
	for _, sub := range []string{project.DirSource, project.DirAudio, project.DirSubtitles, project.DirEffects, project.DirOutput, project.DirTmp} {
		fmt.Printf("  %s/\n", sub)
	}
	fmt.Println("")
	fmt.Printf("Use 'luma-cli project use %s' to activate.\n", name)
	return nil
}

func cmdProjectList(args []string) error {
	projects, err := project.ListProjects()
	if err != nil {
		return output.ErrSystem("%v\n", err)
	}

	if len(projects) == 0 {
		fmt.Println("No projects found. Create one with: luma-cli project create <name>")
		return nil
	}

	activeName, _ := project.ActiveProjectName()

	fmt.Println("Projects:")
	fmt.Println("")
	for _, p := range projects {
		marker := "  "
		if p.Name == activeName {
			marker = "* "
		}
		sourceInfo := ""
		if p.Source != "" {
			sourceInfo = fmt.Sprintf("  source: %s", filepath.Base(p.Source))
		}
		stepCount := len(p.History)
		fmt.Printf("%s%s  %s%s  (%d steps)\n", marker, p.Name, formatRelativeTime(p.CreatedAt), sourceInfo, stepCount)
	}
	fmt.Println("")
	if activeName != "" {
		fmt.Printf("Active project: %s\n", activeName)
	}
	return nil
}

func cmdProjectUse(args []string) error {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli project use <name>")
		return nil
	}

	name := args[0]
	if err := project.SetActiveProject(name); err != nil {
		return output.ErrSystem("%v\n", err)
	}

	p, err := project.FindByName(name)
	if err != nil {
		return output.ErrSystem("%v\n", err)
	}

	fmt.Printf("Switched to project: %s\n", name)
	fmt.Printf("  Path: %s\n", p.Path)
	return nil
}

func cmdProjectInfo(args []string) error {
	p, err := resolveProject(args)
	if err != nil {
		return output.ErrSystem("%v\n", err)
	}

	fmt.Printf("Project: %s\n", p.Name)
	fmt.Printf("  Path:      %s\n", p.Path)
	fmt.Printf("  Created:   %s\n", p.CreatedAt.Format("2006-01-02 15:04:05"))
	if p.Source != "" {
		fmt.Printf("  Source:    %s\n", p.Source)
	}

	fmt.Println("")
	fmt.Println("Directories:")
	for _, sub := range []string{project.DirSource, project.DirAudio, project.DirSubtitles, project.DirEffects, project.DirOutput, project.DirTmp} {
		dirPath := p.SubDir(sub)
		count, size := dirStats(dirPath)
		fmt.Printf("  %-12s %d files, %s\n", sub+"/", count, formatSize(size))
	}

	if len(p.History) > 0 {
		fmt.Println("")
		fmt.Println("History:")
		for i, step := range p.History {
			fmt.Printf("  %d. [%s] %s", i+1, step.Timestamp.Format("01-02 15:04"), step.Command)
			if step.Input != "" {
				fmt.Printf("  in: %s", filepath.Base(step.Input))
			}
			if step.Output != "" {
				fmt.Printf("  out: %s", filepath.Base(step.Output))
			}
			fmt.Println()
		}
	}
	return nil
}

func cmdProjectClean(args []string) error {
	p, err := resolveProject(args)
	if err != nil {
		return output.ErrSystem("%v\n", err)
	}

	if err := p.CleanTmp(); err != nil {
		return output.ErrSystem("%v\n", err)
	}

	fmt.Printf("Cleaned tmp directory: %s\n", p.SubDir(project.DirTmp))
	return nil
}

func resolveProject(args []string) (*project.Project, error) {
	for i := 0; i < len(args); i++ {
		if args[i] == "--project" && i+1 < len(args) {
			return project.FindByName(args[i+1])
		}
	}

	p, err := project.GetActiveProject()
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, fmt.Errorf("no active project. Use 'luma-cli project use <name>' or pass --project <name>")
	}
	return p, nil
}

func dirStats(dir string) (count int, size int64) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				count++
				size += info.Size()
			}
		}
	}
	return
}

func formatSize(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatRelativeTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

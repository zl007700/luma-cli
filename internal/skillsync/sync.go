package skillsync

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/luma-cli/lumer-cli/internal/config"
)

const (
	DefaultSource = "zl007700/luma-cli"

	envSource = "LUMA_SKILLS_SOURCE"
)

// Stamp records the last successful skills sync.
type Stamp struct {
	Version  string    `json:"version"`
	Source   string    `json:"source"`
	SyncedAt time.Time `json:"synced_at"`
}

type SyncOptions struct {
	Source  string
	Skill   string
	Global  bool
	Yes     bool
	Version string
	Stdout  io.Writer
	Stderr  io.Writer
}

type NpmInstallOptions struct {
	Version string
	Stdout  io.Writer
	Stderr  io.Writer
}

func SourceFromEnv() string {
	if source := strings.TrimSpace(os.Getenv(envSource)); source != "" {
		return source
	}
	return DefaultSource
}

func StampPath() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "skills.stamp.json"), nil
}

func ReadStamp() (*Stamp, error) {
	path, err := StampPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read skills stamp: %w", err)
	}
	var stamp Stamp
	if err := json.Unmarshal(data, &stamp); err != nil {
		return nil, fmt.Errorf("parse skills stamp: %w", err)
	}
	return &stamp, nil
}

func WriteStamp(version, source string) error {
	path, err := StampPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	stamp := Stamp{
		Version:  strings.TrimSpace(version),
		Source:   strings.TrimSpace(source),
		SyncedAt: time.Now(),
	}
	data, err := json.MarshalIndent(stamp, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal skills stamp: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

func IsVersionDrift(current string, stamp *Stamp) bool {
	current = strings.TrimSpace(current)
	if current == "" || current == "dev" || current == "local" || stamp == nil {
		return false
	}
	return strings.TrimSpace(stamp.Version) != current
}

func RunSkillsAdd(opts SyncOptions) error {
	source := strings.TrimSpace(opts.Source)
	if source == "" {
		source = SourceFromEnv()
	}
	args := BuildSkillsAddArgs(source, opts.Skill, opts.Global, opts.Yes)
	return runCommand(npxCommand(), args, opts.Stdout, opts.Stderr)
}

func RunNpmInstall(opts NpmInstallOptions) error {
	pkg := "@lumageo/luma-cli"
	if version := strings.TrimSpace(opts.Version); version != "" && version != "latest" {
		pkg += "@" + version
	} else if strings.TrimSpace(opts.Version) == "latest" {
		pkg += "@latest"
	}
	return runCommand(npmCommand(), []string{"install", "-g", pkg}, opts.Stdout, opts.Stderr)
}

func LatestNpmVersion() (string, error) {
	out, err := exec.Command(npmCommand(), "view", "@lumageo/luma-cli", "version").Output()
	if err != nil {
		return "", fmt.Errorf("npm view @lumageo/luma-cli version: %w", err)
	}
	version := strings.TrimSpace(string(out))
	if version == "" {
		return "", fmt.Errorf("npm view @lumageo/luma-cli version returned empty version")
	}
	return version, nil
}

func BuildSkillsAddArgs(source, skill string, global, yes bool) []string {
	args := []string{"-y", "skills", "add", source}
	if skill = strings.TrimSpace(skill); skill != "" {
		args = append(args, "-s", skill)
	}
	if global {
		args = append(args, "-g")
	}
	if yes {
		args = append(args, "-y")
	}
	return args
}

func runCommand(name string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.Command(name, args...)
	if stdout != nil {
		cmd.Stdout = stdout
	}
	if stderr != nil {
		cmd.Stderr = stderr
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

func npmCommand() string {
	if runtime.GOOS == "windows" {
		return "npm.cmd"
	}
	return "npm"
}

func npxCommand() string {
	if runtime.GOOS == "windows" {
		return "npx.cmd"
	}
	return "npx"
}

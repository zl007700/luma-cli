package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	appconfig "github.com/luma-cli/lumer-cli/internal/config"
)

// Project represents a video project with organized directory structure.
type Project struct {
	Name         string     `json:"name"`
	Path         string     `json:"path"`
	CreatedAt    time.Time  `json:"created_at"`
	Source       string     `json:"source,omitempty"`
	LatestTTSKey string     `json:"latest_tts_key,omitempty"` // cloud object key for most recent TTS output
	History      []Step     `json:"history,omitempty"`
	Artifacts    []Artifact `json:"artifacts,omitempty"`
}

// Step records a processing step executed in the project.
type Step struct {
	Command   string    `json:"command"`
	Timestamp time.Time `json:"timestamp"`
	Input     string    `json:"input,omitempty"`
	Output    string    `json:"output,omitempty"`
}

type Artifact struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Path      string    `json:"path"`
	Step      string    `json:"step,omitempty"`
	Ability   string    `json:"ability,omitempty"`
	Command   string    `json:"command,omitempty"`
	Inputs    []string  `json:"inputs,omitempty"`
	Outputs   []string  `json:"outputs,omitempty"`
	Status    string    `json:"status,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Meta      any       `json:"meta,omitempty"`
}

// Subdirectory names within a project.
const (
	DirSource    = "source"
	DirAudio     = "audio"
	DirSubtitles = "subtitles"
	DirEffects   = "effects"
	DirOutput    = "output"
	DirTmp       = "tmp"
)

// subDirs is the list of all subdirectories to create.
var subDirs = []string{DirSource, DirAudio, DirSubtitles, DirEffects, DirOutput, DirTmp}

// registryFile is the global project registry file name.
const registryFile = "projects.json"

// Registry holds the list of projects and the active project name.
type Registry struct {
	ActiveProject string   `json:"active_project,omitempty"`
	Projects      []string `json:"projects"` // list of project paths
}

// registryDir returns the directory for the global registry (~/.luma).
func registryDir() (string, error) {
	dir, err := appconfig.Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create .luma dir: %w", err)
	}
	return dir, nil
}

// loadRegistry loads the global project registry.
func loadRegistry() (*Registry, error) {
	dir, err := registryDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, registryFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Registry{}, nil
		}
		return nil, fmt.Errorf("read registry: %w", err)
	}
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	return &reg, nil
}

// saveRegistry persists the global project registry.
func saveRegistry(reg *Registry) error {
	dir, err := registryDir()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, registryFile)
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal registry: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// Create creates a new project directory structure and metadata file.
// basePath defaults to current working directory if empty.
func Create(name, basePath string) (*Project, error) {
	if name == "" {
		return nil, fmt.Errorf("project name cannot be empty")
	}

	if basePath == "" {
		lumaDir, err := registryDir()
		if err != nil {
			return nil, err
		}
		basePath = filepath.Join(lumaDir, "projects")
		if err := os.MkdirAll(basePath, 0755); err != nil {
			return nil, fmt.Errorf("create projects dir: %w", err)
		}
	}

	projPath := filepath.Join(basePath, name)

	// Check if project already exists
	if _, err := os.Stat(projPath); err == nil {
		return nil, fmt.Errorf("project directory already exists: %s", projPath)
	}

	// Create subdirectories
	for _, sub := range subDirs {
		if err := os.MkdirAll(filepath.Join(projPath, sub), 0755); err != nil {
			return nil, fmt.Errorf("create dir %s: %w", sub, err)
		}
	}

	p := &Project{
		Name:      name,
		Path:      projPath,
		CreatedAt: time.Now(),
	}

	if err := p.Save(); err != nil {
		return nil, fmt.Errorf("save project metadata: %w", err)
	}

	// Register project
	reg, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	reg.Projects = append(reg.Projects, projPath)
	// Auto-activate if this is the first project
	if len(reg.Projects) == 1 {
		reg.ActiveProject = name
	}
	if err := saveRegistry(reg); err != nil {
		return nil, err
	}

	return p, nil
}

// Load reads project metadata from the given project directory.
func Load(projPath string) (*Project, error) {
	metaPath := filepath.Join(projPath, "project.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("read project.json: %w", err)
	}
	var p Project
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse project.json: %w", err)
	}
	return &p, nil
}

// Save writes project metadata to project.json.
func (p *Project) Save() error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal project: %w", err)
	}
	return os.WriteFile(filepath.Join(p.Path, "project.json"), data, 0644)
}

// AddStep appends a processing step to the project history.
func (p *Project) AddStep(step Step) error {
	p.History = append(p.History, step)
	return p.Save()
}

func (p *Project) AddArtifact(artifact Artifact) error {
	if artifact.ID == "" {
		artifact.ID = fmt.Sprintf("%s_%d", artifact.Type, time.Now().Unix())
	}
	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = time.Now()
	}
	p.Artifacts = append(p.Artifacts, artifact)
	return p.Save()
}

// SubDir returns the absolute path of a project subdirectory.
func (p *Project) SubDir(name string) string {
	return filepath.Join(p.Path, name)
}

// CleanTmp removes all files in the project's tmp directory.
func (p *Project) CleanTmp() error {
	tmpDir := p.SubDir(DirTmp)
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		return fmt.Errorf("read tmp dir: %w", err)
	}
	for _, entry := range entries {
		if err := os.RemoveAll(filepath.Join(tmpDir, entry.Name())); err != nil {
			return fmt.Errorf("remove %s: %w", entry.Name(), err)
		}
	}
	return nil
}

// GetActiveProject returns the currently active project, or nil if none.
func GetActiveProject() (*Project, error) {
	reg, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	if reg.ActiveProject == "" {
		return nil, nil
	}

	// Find the active project path
	for _, projPath := range reg.Projects {
		p, err := Load(projPath)
		if err != nil {
			continue
		}
		if p.Name == reg.ActiveProject {
			return p, nil
		}
	}
	return nil, nil
}

// SetActiveProject sets the active project by name.
func SetActiveProject(name string) error {
	reg, err := loadRegistry()
	if err != nil {
		return err
	}

	// Verify project exists
	found := false
	for _, projPath := range reg.Projects {
		p, err := Load(projPath)
		if err != nil {
			continue
		}
		if p.Name == name {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("project not found: %s", name)
	}

	reg.ActiveProject = name
	return saveRegistry(reg)
}

// ListProjects returns all registered projects with their metadata.
func ListProjects() ([]*Project, error) {
	reg, err := loadRegistry()
	if err != nil {
		return nil, err
	}

	var projects []*Project
	for _, projPath := range reg.Projects {
		p, err := Load(projPath)
		if err != nil {
			// Skip projects that can't be loaded (may have been moved/deleted)
			continue
		}
		projects = append(projects, p)
	}
	return projects, nil
}

// ActiveProjectName returns the name of the active project, or empty string.
func ActiveProjectName() (string, error) {
	reg, err := loadRegistry()
	if err != nil {
		return "", err
	}
	return reg.ActiveProject, nil
}

// FindByName finds a project by name from the registry.
func FindByName(name string) (*Project, error) {
	reg, err := loadRegistry()
	if err != nil {
		return nil, err
	}
	for _, projPath := range reg.Projects {
		p, err := Load(projPath)
		if err != nil {
			continue
		}
		if p.Name == name {
			return p, nil
		}
	}
	return nil, fmt.Errorf("project not found: %s", name)
}

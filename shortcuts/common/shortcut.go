package common

import "strings"

// Flag describes a CLI flag for an atomic shortcut.
type Flag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required,omitempty"`
	Default     string `json:"default,omitempty"`
}

// Shortcut describes an agent-callable atomic capability.
type Shortcut struct {
	ID          string   `json:"id"`
	Service     string   `json:"service"`
	CommandLine string   `json:"command"`
	Command     string   `json:"subcommand,omitempty"`
	Description string   `json:"description"`
	Risk        string   `json:"risk"`
	Flags       []Flag   `json:"flags,omitempty"`
	Outputs     []string `json:"outputs,omitempty"`
	Examples    []string `json:"examples,omitempty"`
	Skills      []string `json:"skills,omitempty"`
}

// FullCommand returns the command path as typed by a CLI caller.
func (s Shortcut) FullCommand() string {
	parts := []string{"luma-cli"}
	if s.Service != "" {
		parts = append(parts, s.Service)
	}
	if s.Command != "" {
		parts = append(parts, s.Command)
	}
	return strings.Join(parts, " ")
}

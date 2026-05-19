package cmdutil

import (
	"fmt"
	"strconv"
	"strings"
)

// Args is a small parsed view over CLI arguments.
type Args struct {
	Positionals []string
	Flags       map[string]string
}

// Parse splits command arguments into positional values and --flag values.
func Parse(raw []string) Args {
	parsed := Args{
		Positionals: make([]string, 0, len(raw)),
		Flags:       map[string]string{},
	}
	for i := 0; i < len(raw); i++ {
		arg := raw[i]
		if !strings.HasPrefix(arg, "--") || arg == "--" {
			parsed.Positionals = append(parsed.Positionals, arg)
			continue
		}

		name, value, ok := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		if name == "" {
			continue
		}
		if !ok && i+1 < len(raw) && !strings.HasPrefix(raw[i+1], "--") {
			value = raw[i+1]
			i++
		}
		parsed.Flags[name] = value
	}
	return parsed
}

func (a Args) Pos(index int) string {
	if index < 0 || index >= len(a.Positionals) {
		return ""
	}
	return a.Positionals[index]
}

func (a Args) String(name, fallback string) string {
	if value, ok := a.Flags[name]; ok {
		return value
	}
	return fallback
}

func (a Args) Has(name string) bool {
	_, ok := a.Flags[name]
	return ok
}

func (a Args) Int(name string, fallback int) (int, error) {
	value, ok := a.Flags[name]
	if !ok || value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback, fmt.Errorf("--%s must be an integer", name)
	}
	return n, nil
}

func (a Args) Float(name string, fallback float64) (float64, error) {
	value, ok := a.Flags[name]
	if !ok || value == "" {
		return fallback, nil
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback, fmt.Errorf("--%s must be a number", name)
	}
	return n, nil
}

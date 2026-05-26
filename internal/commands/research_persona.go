package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

type researchPersona struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Summary         string `json:"summary"`
	RoleDescription string `json:"role_description"`
	UpdatedAt       string `json:"updated_at"`
}

type personaStore struct {
	Personas []researchPersona `json:"personas"`
}

func cmdResearchPersona(raw []string) error {
	if len(raw) < 1 {
		printResearchUsage()
		return nil
	}
	switch raw[0] {
	case "list":
		store := loadPersonaStore()
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: store})
			return nil
		}
		if len(store.Personas) == 0 {
			fmt.Println("No personas saved.")
			return nil
		}
		fmt.Printf("%-24s %-18s %s\n", "ID", "NAME", "SUMMARY")
		for _, item := range store.Personas {
			fmt.Printf("%-24s %-18s %s\n", item.ID, item.Name, item.Summary)
		}
	case "get":
		name := strings.TrimSpace("")
		if len(raw) > 1 {
			name = raw[1]
		}
		if name == "" {
			fmt.Println("usage: luma-cli research persona get <name_or_id>")
			return nil
		}
		persona, ok := loadPersonaByName(name)
		if !ok {
			return output.ErrValidation("persona not found: %s", name)
			return nil
		}
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: persona})
			return nil
		}
		data, _ := json.MarshalIndent(persona, "", "  ")
		fmt.Println(string(data))
	case "save":
		args := cmdutil.Parse(raw[1:])
		name := strings.TrimSpace(args.Pos(0))
		if name == "" {
			name = strings.TrimSpace(args.String("name", ""))
		}
		role := strings.TrimSpace(args.String("role", ""))
		if input := strings.TrimSpace(args.String("input", "")); input != "" {
			data, err := os.ReadFile(input)
			if err != nil {
				return output.ErrSystem("read input failed: %v", err)
				return nil
			}
			role = strings.TrimSpace(string(data))
		}
		if name == "" || role == "" {
			fmt.Println("usage: luma-cli research persona save <name> --role <description>")
			return nil
		}
		summary := strings.TrimSpace(args.String("summary", ""))
		persona := savePersona(name, role, summary)
		if runtimeOpts.JSON {
			_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: persona})
			return nil
		}
		fmt.Printf("Saved persona: %s\n", persona.Name)
	default:
		printResearchUsage()
	}
	return nil
}

func personaStorePath() string {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".luma-cli")
	}
	return filepath.Join(configDir, "luma-cli", "personas.json")
}

func loadPersonaStore() personaStore {
	path := personaStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return personaStore{}
	}
	var store personaStore
	if err := json.Unmarshal(data, &store); err != nil {
		return personaStore{}
	}
	return store
}

func savePersonaStore(store personaStore) error {
	return writeJSONFile(personaStorePath(), store)
}

func loadPersonaByName(name string) (researchPersona, bool) {
	name = strings.TrimSpace(name)
	store := loadPersonaStore()
	for _, item := range store.Personas {
		if item.ID == name || item.Name == name {
			return item, true
		}
	}
	return researchPersona{}, false
}

func savePersona(name, role, summary string) researchPersona {
	store := loadPersonaStore()
	now := time.Now().Format(time.RFC3339)
	item := researchPersona{
		ID:              "persona_" + slugForID(name),
		Name:            name,
		Summary:         summary,
		RoleDescription: role,
		UpdatedAt:       now,
	}
	replaced := false
	for i := range store.Personas {
		if store.Personas[i].Name == name || store.Personas[i].ID == item.ID {
			store.Personas[i] = item
			replaced = true
			break
		}
	}
	if !replaced {
		store.Personas = append([]researchPersona{item}, store.Personas...)
	}
	_ = savePersonaStore(store)
	return item
}

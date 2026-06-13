package commands

import "testing"

func TestCommandRegistryIncludesAgentTools(t *testing.T) {
	registry := commandRegistry()
	spec, ok := registry["tools"]
	if !ok {
		t.Fatal("expected tools command to be registered")
	}
	if spec.Handler == nil {
		t.Fatal("expected tools command handler")
	}
}

func TestCommandRegistryIncludesURLRead(t *testing.T) {
	registry := commandRegistry()
	spec, ok := registry["url-read"]
	if !ok {
		t.Fatal("expected url-read command to be registered")
	}
	if spec.Handler == nil {
		t.Fatal("expected url-read command handler")
	}
}

func TestCommandRegistryIncludesAvatarPersona(t *testing.T) {
	registry := commandRegistry()
	spec, ok := registry["avatar-persona"]
	if !ok {
		t.Fatal("expected avatar-persona command to be registered")
	}
	if spec.Handler == nil {
		t.Fatal("expected avatar-persona command handler")
	}
}

func TestCommandNamesSorted(t *testing.T) {
	names := commandNames()
	if len(names) == 0 {
		t.Fatal("expected command names")
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] > names[i] {
			t.Fatalf("command names not sorted: %q before %q", names[i-1], names[i])
		}
	}
}

package registry

import "testing"

func TestFind(t *testing.T) {
	sc, ok := Find("asr.transcribe")
	if !ok {
		t.Fatal("expected asr.transcribe shortcut")
	}
	if sc.Service != "asr" {
		t.Fatalf("expected asr service, got %q", sc.Service)
	}
}

func TestListSorted(t *testing.T) {
	items := List()
	if len(items) == 0 {
		t.Fatal("expected shortcuts")
	}
	for i := 1; i < len(items); i++ {
		if items[i-1].ID > items[i].ID {
			t.Fatalf("shortcuts not sorted: %q before %q", items[i-1].ID, items[i].ID)
		}
	}
}

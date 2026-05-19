package cmdutil

import "testing"

func TestParsePositionalsAndFlags(t *testing.T) {
	args := Parse([]string{"input.mp4", "--language", "zh", "--output=out.mp4", "--dry-run"})

	if got := args.Pos(0); got != "input.mp4" {
		t.Fatalf("expected positional input.mp4, got %q", got)
	}
	if got := args.String("language", "en"); got != "zh" {
		t.Fatalf("expected language zh, got %q", got)
	}
	if got := args.String("output", ""); got != "out.mp4" {
		t.Fatalf("expected output out.mp4, got %q", got)
	}
	if got := args.String("dry-run", "false"); got != "" {
		t.Fatalf("expected empty value for dry-run flag, got %q", got)
	}
	if !args.Has("dry-run") {
		t.Fatal("expected dry-run flag to be present")
	}
	if args.Has("missing") {
		t.Fatal("expected missing flag to be absent")
	}
}

func TestTypedFlags(t *testing.T) {
	args := Parse([]string{"--scale", "4", "--speech-rate", "1.25"})

	scale, err := args.Int("scale", 2)
	if err != nil {
		t.Fatal(err)
	}
	if scale != 4 {
		t.Fatalf("expected scale 4, got %d", scale)
	}

	rate, err := args.Float("speech-rate", 1.1)
	if err != nil {
		t.Fatal(err)
	}
	if rate != 1.25 {
		t.Fatalf("expected rate 1.25, got %f", rate)
	}
}

func TestTypedFlagErrors(t *testing.T) {
	args := Parse([]string{"--scale", "large"})
	if _, err := args.Int("scale", 2); err == nil {
		t.Fatal("expected integer parse error")
	}
}

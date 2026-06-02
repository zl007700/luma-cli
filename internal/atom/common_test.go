package atom

import "testing"

func TestResultURL(t *testing.T) {
	status := map[string]any{
		"output": map[string]any{
			"result": map[string]any{
				"url": "https://example.test/output.mp4",
			},
		},
	}
	if got := ResultURL(status); got != "https://example.test/output.mp4" {
		t.Fatalf("expected nested result url, got %q", got)
	}
}

func TestResultObjectKey(t *testing.T) {
	status := map[string]any{
		"output": map[string]any{
			"result_object_key": "prod/resource/user/file.wav",
		},
	}
	if got := ResultObjectKey(status); got != "prod/resource/user/file.wav" {
		t.Fatalf("expected object key, got %q", got)
	}
}

func TestTaskFailure(t *testing.T) {
	status := map[string]any{
		"status":        "failed",
		"error_message": "missing source object",
	}
	if got := TaskFailure(status); got != "missing source object" {
		t.Fatalf("expected task failure message, got %q", got)
	}

	status["status"] = "completed"
	if got := TaskFailure(status); got != "" {
		t.Fatalf("expected no failure message, got %q", got)
	}
}

func TestStripResourcePrefix(t *testing.T) {
	got := StripResourcePrefix("prod/resource/user-a/audio/out.wav", "user-a")
	if got != "audio/out.wav" {
		t.Fatalf("expected stripped key, got %q", got)
	}
}

func TestNormalizeResourceKey(t *testing.T) {
	got := NormalizeResourceKey("prod/resource/user-a/voice/name_original.wav", "user-a")
	if got != "voice/name_original.wav" {
		t.Fatalf("expected relative key, got %q", got)
	}

	got = NormalizeResourceKey("common/voice/name_original.wav", "user-a")
	if got != "common/voice/name_original.wav" {
		t.Fatalf("expected common key unchanged, got %q", got)
	}
}

func TestResourceKeyFromMapPrefixesCommonAssets(t *testing.T) {
	got := ResourceKeyFromMap(map[string]any{
		"user_id":    "common",
		"object_key": "voice/name_original.wav",
	}, "user-a")
	if got != "common/voice/name_original.wav" {
		t.Fatalf("expected common resource key, got %q", got)
	}
}

func TestAssetFriendlyName(t *testing.T) {
	got := AssetFriendlyName("voice/name_original.wav")
	if got != "name" {
		t.Fatalf("expected friendly name, got %q", got)
	}

	got = AssetFriendlyName("roles/boss_at_home_33b86b65b421.mp4")
	if got != "boss_at_home" {
		t.Fatalf("expected hash suffix stripped, got %q", got)
	}
}

func TestResourceDisplayName(t *testing.T) {
	got := ResourceDisplayName(map[string]any{"meta": map[string]any{"display_name": "老板在家里"}})
	if got != "老板在家里" {
		t.Fatalf("expected display name, got %q", got)
	}
}

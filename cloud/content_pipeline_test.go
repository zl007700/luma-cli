package cloud

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContentStageSendsTopLevelStage(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/agent/content-pipeline/stage" {
			t.Fatalf("path = %s, want content pipeline stage", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ability":    "content.pipeline.detail_expansion_plan",
			"request_id": "ag_test",
			"result": map[string]any{
				"selected": []any{},
			},
			"usage": map[string]any{"total_tokens": 12},
		})
	}))
	defer server.Close()
	t.Setenv("LUMA_API_URL", server.URL)

	resp, err := ContentStage(
		"detail_expansion_plan",
		map[string]any{"payload": map[string]any{"sources": []any{}}},
		map[string]any{"model_tier": "basic_model"},
		"test-card-key",
		0,
	)
	if err != nil {
		t.Fatalf("ContentStage: %v", err)
	}
	if resp.RequestID != "ag_test" {
		t.Fatalf("request_id = %s", resp.RequestID)
	}
	if got["stage"] != "detail_expansion_plan" {
		t.Fatalf("stage was not top-level: %#v", got)
	}
	if _, ok := got["input"].(map[string]any); !ok {
		t.Fatalf("input missing: %#v", got)
	}
}

func TestContentMemorySaveUploadsContentMemoryAsset(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	var signBody map[string]any
	var uploadedBody []byte
	deleted := []string{}
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	mux.HandleFunc("/v1/assets/upload-sign", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&signBody); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"asset_id":       "asset_content_memory_test",
			"bucket":         "bucket",
			"object_key":     "assets/user/test/asset/original/evergreen_topics.current.json",
			"upload_url":     server.URL + "/upload",
			"expire_seconds": 300,
		})
	})
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		uploadedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v1/assets/complete-upload", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"asset_id": "asset_content_memory_test",
			"status":   "processing",
		})
	})
	mux.HandleFunc("/v1/assets/search", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"search_mode": "catalog_v0",
			"items": []map[string]any{
				{"asset_id": "asset_content_memory_test", "kind": ContentMemoryKind, "scope": "user", "group_name": "content_profile_a"},
				{"asset_id": "asset_old_1", "kind": ContentMemoryKind, "scope": "user", "group_name": "content_profile_a"},
				{"asset_id": "asset_old_2", "kind": ContentMemoryKind, "scope": "user", "group_name": "content_profile_a"},
			},
		})
	})
	mux.HandleFunc("/v1/assets/asset_old_1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s", r.Method)
		}
		deleted = append(deleted, "asset_old_1")
		_ = json.NewEncoder(w).Encode(map[string]any{"asset_id": "asset_old_1", "status": "deleted"})
	})
	mux.HandleFunc("/v1/assets/asset_old_2", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s", r.Method)
		}
		deleted = append(deleted, "asset_old_2")
		_ = json.NewEncoder(w).Encode(map[string]any{"asset_id": "asset_old_2", "status": "deleted"})
	})
	t.Setenv("LUMA_API_URL", server.URL)

	item, err := ContentMemorySave("profile_a", "evergreen_topics", map[string]any{"items": []any{}}, "test-card-key")
	if err != nil {
		t.Fatalf("ContentMemorySave: %v", err)
	}
	if item.AssetID != "asset_content_memory_test" {
		t.Fatalf("asset_id = %s", item.AssetID)
	}
	if signBody["kind"] != ContentMemoryKind {
		t.Fatalf("kind = %v", signBody["kind"])
	}
	if signBody["group_name"] != "content_profile_a" {
		t.Fatalf("group_name = %v", signBody["group_name"])
	}
	if signBody["filename"] != "evergreen_topics.current.json" {
		t.Fatalf("filename = %v", signBody["filename"])
	}
	metadata, _ := signBody["metadata"].(map[string]any)
	if metadata["artifact_type"] != "evergreen_topics" || metadata["is_current"] != true {
		t.Fatalf("metadata = %#v", metadata)
	}
	if len(uploadedBody) == 0 {
		t.Fatal("expected uploaded body")
	}
	if len(deleted) != 2 {
		t.Fatalf("deleted = %#v, want two old assets", deleted)
	}
}

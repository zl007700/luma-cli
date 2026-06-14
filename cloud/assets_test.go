package cloud

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFilterAssetItemsKeepsRequestedKindGroupAndScope(t *testing.T) {
	items := []AssetItem{
		{AssetID: "system_voice", Kind: "voice", GroupName: "common", Scope: "system"},
		{AssetID: "user_voice", Kind: "voice", GroupName: "common", Scope: "user"},
		{AssetID: "system_template", Kind: "template", GroupName: "common", Scope: "system"},
		{AssetID: "other_group", Kind: "voice", GroupName: "demo", Scope: "system"},
	}

	got := filterAssetItems(items, "voice", "common", "system")

	if len(got) != 1 {
		t.Fatalf("filtered length = %d, want 1: %#v", len(got), got)
	}
	if got[0].AssetID != "system_voice" {
		t.Fatalf("filtered asset = %q, want system_voice", got[0].AssetID)
	}
}

func TestAssetsSearchWithRequestSendsMetadataFilter(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/assets/search" {
			t.Fatalf("path = %s, want /v1/assets/search", r.URL.Path)
		}
		if r.Header.Get("X-User-Id") != "test-card-key" {
			t.Fatalf("missing card key header")
		}
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"search_mode": "catalog_v0",
			"items": []map[string]any{
				{
					"asset_id":   "asset_1",
					"kind":       "content_memory",
					"scope":      "user",
					"group_name": "content_profile_a",
					"metadata": map[string]any{
						"artifact_type": "evergreen_topics",
						"is_current":    true,
					},
				},
			},
		})
	}))
	defer server.Close()
	t.Setenv("LUMA_API_URL", server.URL)

	resp, err := AssetsSearchWithRequest(AssetsSearchRequest{
		Kind:      "content_memory",
		GroupName: "content_profile_a",
		Scope:     "user",
		Limit:     1,
		Metadata: map[string]any{
			"artifact_type": "evergreen_topics",
			"is_current":    true,
		},
	}, "test-card-key")
	if err != nil {
		t.Fatalf("AssetsSearchWithRequest: %v", err)
	}
	if len(resp.Items) != 1 || resp.Items[0].AssetID != "asset_1" {
		t.Fatalf("unexpected response: %#v", resp.Items)
	}
	metadata, _ := got["metadata"].(map[string]any)
	if metadata["artifact_type"] != "evergreen_topics" || metadata["is_current"] != true {
		t.Fatalf("metadata filter not sent: %#v", got)
	}
}

func TestAssetReadTextSignsAndDownloads(t *testing.T) {
	t.Setenv("LUMA_CONFIG_DIR", t.TempDir())
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	mux.HandleFunc("/v1/assets/sign", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"asset_id":     "asset_text",
			"download_url": server.URL + "/download/asset_text",
		})
	})
	mux.HandleFunc("/download/asset_text", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello content"))
	})
	defer server.Close()
	t.Setenv("LUMA_API_URL", server.URL)

	data, err := AssetReadText("asset_text", "test-card-key")
	if err != nil {
		t.Fatalf("AssetReadText: %v", err)
	}
	if string(data) != "hello content" {
		t.Fatalf("data = %q", string(data))
	}
}

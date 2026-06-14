package cloud

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	ContentMemoryKind   = "content_memory"
	ContentArtifactKind = "content_artifact"
)

type ContentStageRequest struct {
	Stage   string         `json:"stage"`
	Input   map[string]any `json:"input"`
	Options map[string]any `json:"options,omitempty"`
}

func ContentStage(stage string, input map[string]any, options map[string]any, cardKey string, timeout time.Duration) (*AgentAbilityResponse, error) {
	stage = strings.TrimSpace(stage)
	if stage == "" {
		return nil, fmt.Errorf("stage is required")
	}
	payload := map[string]any{
		"stage": stage,
		"input": input,
	}
	if options != nil {
		payload["options"] = options
	}
	if timeout <= 0 {
		timeout = 240 * time.Second
	}
	result, err := apiRequestWithTimeout("POST", "/v1/agent/content-pipeline/stage", payload, cardKey, timeout)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AgentAbilityResponse](result)
	if err != nil {
		return nil, fmt.Errorf("parse content stage response failed: %w", err)
	}
	return &item, nil
}

func ContentGroupName(profileID string) string {
	clean := strings.TrimSpace(profileID)
	if clean == "" {
		clean = "default"
	}
	clean = strings.ReplaceAll(clean, "\\", "_")
	clean = strings.ReplaceAll(clean, "/", "_")
	return "content_" + clean
}

func ContentMemoryLatest(profileID, artifactType, cardKey string) (*AssetItem, error) {
	return contentLatestAsset(ContentMemoryKind, profileID, "", artifactType, true, cardKey)
}

func ContentArtifactLatest(profileID, runID, artifactType, cardKey string) (*AssetItem, error) {
	return contentLatestAsset(ContentArtifactKind, profileID, runID, artifactType, false, cardKey)
}

func ContentMemoryRead(profileID, artifactType, cardKey string) ([]byte, *AssetItem, error) {
	item, err := ContentMemoryLatest(profileID, artifactType, cardKey)
	if err != nil {
		return nil, nil, err
	}
	data, err := AssetReadText(item.AssetID, cardKey)
	if err != nil {
		return nil, item, err
	}
	return data, item, nil
}

func ContentArtifactRead(profileID, runID, artifactType, cardKey string) ([]byte, *AssetItem, error) {
	item, err := ContentArtifactLatest(profileID, runID, artifactType, cardKey)
	if err != nil {
		return nil, nil, err
	}
	data, err := AssetReadText(item.AssetID, cardKey)
	if err != nil {
		return nil, item, err
	}
	return data, item, nil
}

func ContentMemorySave(profileID, artifactType string, value any, cardKey string) (*AssetItem, error) {
	metadata := map[string]any{
		"profile_id":     strings.TrimSpace(profileID),
		"artifact_type":  strings.TrimSpace(artifactType),
		"is_current":     true,
		"schema_version": 1,
		"created_by":     "luma-cli",
	}
	return uploadContentValue(ContentMemoryKind, profileID, "", artifactType, true, value, cardKey, metadata)
}

func ContentArtifactSave(profileID, runID, artifactType string, value any, cardKey string) (*AssetItem, error) {
	metadata := map[string]any{
		"profile_id":     strings.TrimSpace(profileID),
		"run_id":         strings.TrimSpace(runID),
		"artifact_type":  strings.TrimSpace(artifactType),
		"is_current":     false,
		"schema_version": 1,
		"created_by":     "luma-cli",
	}
	return uploadContentValue(ContentArtifactKind, profileID, runID, artifactType, false, value, cardKey, metadata)
}

func contentLatestAsset(kind, profileID, runID, artifactType string, isCurrent bool, cardKey string) (*AssetItem, error) {
	metadata := map[string]any{
		"artifact_type": strings.TrimSpace(artifactType),
	}
	if strings.TrimSpace(profileID) != "" {
		metadata["profile_id"] = strings.TrimSpace(profileID)
	}
	if strings.TrimSpace(runID) != "" {
		metadata["run_id"] = strings.TrimSpace(runID)
	}
	if isCurrent {
		metadata["is_current"] = true
	}
	result, err := AssetsSearchWithRequest(AssetsSearchRequest{
		Kind:      kind,
		GroupName: ContentGroupName(profileID),
		Scope:     "user",
		Limit:     1,
		Metadata:  metadata,
	}, cardKey)
	if err != nil {
		return nil, err
	}
	if len(result.Items) == 0 {
		return nil, fmt.Errorf("content asset not found: kind=%s profile_id=%s run_id=%s artifact_type=%s", kind, profileID, runID, artifactType)
	}
	return &result.Items[0], nil
}

func uploadContentValue(kind, profileID, runID, artifactType string, current bool, value any, cardKey string, metadata map[string]any) (*AssetItem, error) {
	artifactType = strings.TrimSpace(artifactType)
	if artifactType == "" {
		return nil, fmt.Errorf("artifact_type is required")
	}
	filename := artifactType
	if current {
		filename += ".current"
	}
	ext := ".json"
	var data []byte
	var err error
	if text, ok := value.(string); ok {
		ext = ".md"
		data = []byte(text)
	} else {
		data, err = json.MarshalIndent(value, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("marshal content asset failed: %w", err)
		}
	}
	filename += ext
	tmp, err := os.CreateTemp("", "luma-content-*"+ext)
	if err != nil {
		return nil, err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return nil, err
	}
	defer os.Remove(tmpPath)
	displayName := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	return AssetsUploadFileWithName(tmpPath, kind, ContentGroupName(profileID), displayName, filename, cardKey, metadata)
}

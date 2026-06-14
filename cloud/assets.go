package cloud

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type AssetItem struct {
	AssetID              string         `json:"asset_id"`
	Kind                 string         `json:"kind"`
	Scope                string         `json:"scope"`
	DisplayName          string         `json:"display_name"`
	Filename             string         `json:"filename"`
	GroupName            string         `json:"group_name"`
	ContentType          string         `json:"content_type"`
	Status               string         `json:"status"`
	Score                *float64       `json:"score"`
	MatchType            string         `json:"match_type"`
	Duration             *float64       `json:"duration"`
	Width                *int           `json:"width"`
	Height               *int           `json:"height"`
	SHA256               string         `json:"sha256"`
	SizeBytes            *int64         `json:"size_bytes"`
	Tags                 []string       `json:"tags"`
	Metadata             map[string]any `json:"metadata"`
	Bucket               string         `json:"bucket,omitempty"`
	ObjectKey            string         `json:"object_key,omitempty"`
	DownloadURL          string         `json:"download_url,omitempty"`
	ExpireSeconds        int            `json:"expire_seconds,omitempty"`
	PreviewURL           string         `json:"preview_url,omitempty"`
	PreviewExpireSeconds int            `json:"preview_expire_seconds,omitempty"`
}

type AssetsSearchResponse struct {
	SearchMode string      `json:"search_mode"`
	Items      []AssetItem `json:"items"`
}

type AssetsResolveResponse struct {
	Items []AssetItem `json:"items"`
}

type AssetGroupItem struct {
	GroupName   string `json:"group_name"`
	DisplayName string `json:"display_name"`
}

type AssetsGroupsResponse struct {
	Items []AssetGroupItem `json:"items"`
}

type AssetUploadResult struct {
	AssetID       string `json:"asset_id"`
	Bucket        string `json:"bucket"`
	ObjectKey     string `json:"object_key"`
	UploadURL     string `json:"upload_url"`
	ExpireSeconds int    `json:"expire_seconds"`
}

func AssetsSearch(kind, groupName, scope string, limit int, cardKey string) (*AssetsSearchResponse, error) {
	if limit <= 0 {
		limit = 30
	}
	result, err := apiRequest("POST", "/v1/assets/search", map[string]any{
		"kind":       kind,
		"group_name": groupName,
		"scope":      scope,
		"limit":      limit,
	}, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AssetsSearchResponse](result)
	if err != nil {
		return nil, fmt.Errorf("parse assets search response failed: %w", err)
	}
	return &item, nil
}

func AssetsGroups(kind, scope, cardKey string) (*AssetsGroupsResponse, error) {
	path := "/v1/assets/groups"
	params := []string{}
	if kind != "" {
		params = append(params, "kind="+url.QueryEscape(kind))
	}
	if scope != "" {
		params = append(params, "scope="+url.QueryEscape(scope))
	}
	if len(params) > 0 {
		path += "?" + strings.Join(params, "&")
	}
	result, err := apiRequest("GET", path, nil, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AssetsGroupsResponse](result)
	if err != nil {
		return nil, fmt.Errorf("parse assets groups response failed: %w", err)
	}
	return &item, nil
}

func AssetsResolve(assetIDs []string, cardKey string) (*AssetsResolveResponse, error) {
	result, err := apiRequest("POST", "/v1/assets/resolve", map[string]any{"asset_ids": assetIDs}, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AssetsResolveResponse](result)
	if err != nil {
		return nil, fmt.Errorf("parse assets resolve response failed: %w", err)
	}
	return &item, nil
}

func AssetsSign(assetID, cardKey string) (*AssetItem, error) {
	result, err := apiRequest("POST", "/v1/assets/sign", map[string]any{"asset_id": assetID}, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AssetItem](result)
	if err != nil {
		return nil, fmt.Errorf("parse asset sign response failed: %w", err)
	}
	return &item, nil
}

func AssetsDelete(assetID, cardKey string) (map[string]any, error) {
	return apiRequest("DELETE", "/v1/assets/"+assetID, nil, cardKey)
}

func AssetsUploadFile(filePath, kind, groupName, displayName, cardKey string, metadata map[string]any) (*AssetItem, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("asset upload path is a directory: %s", filePath)
	}
	filename := filepath.Base(filePath)
	contentType := guessMimeType(filePath)
	signResult, err := apiRequest("POST", "/v1/assets/upload-sign", map[string]any{
		"kind":         kind,
		"filename":     filename,
		"content_type": contentType,
		"display_name": displayName,
		"group_name":   groupName,
		"metadata":     metadata,
	}, cardKey)
	if err != nil {
		return nil, fmt.Errorf("asset upload sign failed: %w", err)
	}
	upload, err := mapToStruct[AssetUploadResult](signResult)
	if err != nil {
		return nil, fmt.Errorf("parse asset upload sign response failed: %w", err)
	}
	if upload.AssetID == "" || upload.UploadURL == "" {
		return nil, fmt.Errorf("invalid asset upload sign response")
	}
	if err := putFile(upload.UploadURL, filePath, contentType); err != nil {
		return nil, err
	}
	complete, err := apiRequestWithTimeout("POST", "/v1/assets/complete-upload", map[string]any{
		"asset_id": upload.AssetID,
	}, cardKey, 30*time.Second)
	if err != nil {
		return nil, fmt.Errorf("asset complete upload failed: %w", err)
	}
	item, err := mapToStruct[AssetItem](complete)
	if err != nil {
		return nil, fmt.Errorf("parse asset complete response failed: %w", err)
	}
	if item.AssetID == "" {
		item.AssetID = upload.AssetID
	}
	item.Bucket = upload.Bucket
	item.ObjectKey = upload.ObjectKey
	return &item, nil
}

func putFile(uploadURL, filePath, contentType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return err
	}
	req, err := http.NewRequest("PUT", uploadURL, file)
	if err != nil {
		return fmt.Errorf("create upload request failed: %w", err)
	}
	req.ContentLength = stat.Size()
	req.Header.Set("Content-Type", contentType)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("file upload failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload returned HTTP %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

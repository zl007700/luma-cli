package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	appconfig "github.com/luma-cli/lumer-cli/internal/config"
)

// DefaultBaseURL is the default cloud API endpoint.
// Can be overridden via LUMA_API_URL environment variable.
var DefaultBaseURL = "https://api.pikgeo.com"

// BaseURL returns the active API base URL, checking env var first.
func BaseURL() string {
	return appconfig.APIBaseURL(DefaultBaseURL)
}

const multipartThreshold = 100 * 1024 * 1024 // 100MB

type RuntimeResolve struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	Platform       string `json:"platform"`
	Filename       string `json:"filename"`
	Size           int64  `json:"size"`
	SHA256         string `json:"sha256"`
	ArchiveType    string `json:"archive_type"`
	ExecutablePath string `json:"executable_path"`
	FFProbePath    string `json:"ffprobe_path"`
	DownloadURL    string `json:"download_url"`
	ExpireSeconds  int    `json:"expire_seconds"`
}

type ClientResource struct {
	ID            string   `json:"id"`
	Type          string   `json:"type"`
	Name          string   `json:"name"`
	Version       string   `json:"version"`
	Filename      string   `json:"filename"`
	Size          int64    `json:"size"`
	SHA256        string   `json:"sha256"`
	Tags          []string `json:"tags"`
	Description   string   `json:"description"`
	DownloadURL   string   `json:"download_url,omitempty"`
	ExpireSeconds int      `json:"expire_seconds,omitempty"`
}

type SubtitleDefaults struct {
	Font             string `json:"font"`
	FontSize         int    `json:"font_size"`
	SideMargin       int    `json:"side_margin"`
	BottomMargin     int    `json:"bottom_margin"`
	Color            string `json:"color"`
	StrokeColor      string `json:"stroke_color"`
	HighlightColor   string `json:"highlight_color"`
	MaxChars         int    `json:"max_chars"`
	EffectsEnabled   bool   `json:"effects_enabled"`
	HighlightEnabled bool   `json:"highlight_enabled"`
}

type CoverDefaults struct {
	Template     string `json:"template"`
	Font         string `json:"font"`
	TitleSize    int    `json:"title_size"`
	SubtitleSize int    `json:"subtitle_size"`
}

type BGMDefaults struct {
	Default     string  `json:"default"`
	VoiceVolume float64 `json:"voice_volume"`
	BGMVolume   float64 `json:"bgm_volume"`
}

type ClientDefaults struct {
	Subtitle SubtitleDefaults `json:"subtitle"`
	Cover    CoverDefaults    `json:"cover"`
	BGM      BGMDefaults      `json:"bgm"`
}

type ScriptRewriteResponse struct {
	Ability   string         `json:"ability"`
	RequestID string         `json:"request_id"`
	Result    map[string]any `json:"result"`
	Usage     map[string]any `json:"usage"`
}

type AgentAbilityResponse struct {
	Ability   string         `json:"ability"`
	RequestID string         `json:"request_id"`
	Result    map[string]any `json:"result"`
	Usage     map[string]any `json:"usage"`
}

type ResearchResponse = AgentAbilityResponse

func apiRequest(method string, path string, body any, cardKey string) (map[string]any, error) {
	url := BaseURL() + path

	var bodyBytes []byte
	var err error
	if body != nil {
		if bb, ok := body.([]byte); ok {
			bodyBytes = bb
		} else {
			bodyBytes, err = json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("marshal body failed: %w", err)
			}
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", cardKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Client-Version", "v1.0.0")
	req.Header.Set("X-Client-Mode", "cloud")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("server error: HTTP %d, body: %s", resp.StatusCode, string(respBody))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response failed: %w, body: %s", err, string(respBody))
	}
	return result, nil
}

func mapToStruct[T any](payload map[string]any) (T, error) {
	var out T
	data, err := json.Marshal(payload)
	if err != nil {
		return out, err
	}
	err = json.Unmarshal(data, &out)
	return out, err
}

// SubmitTask submits a cloud task (asr, tts, lipsync, enhance, etc.)
func SubmitTask(taskType, taskName string, inputPayload map[string]any, cardKey string) (map[string]any, error) {
	payload := map[string]any{
		"task_type":  taskType,
		"task_name":  taskName,
		"group_name": taskType + "_output",
		"input":      inputPayload,
	}
	body, _ := json.Marshal(payload)
	return apiRequest("POST", "/v1/tasks", body, cardKey)
}

// PollTask fetches the current status of a task.
func PollTask(taskID, cardKey string) (map[string]any, error) {
	return apiRequest("GET", "/v1/tasks/"+taskID, nil, cardKey)
}

// ResolveAsset gets a downloadable URL for a named asset.
func ResolveAsset(name, groupName, cardKey string) (string, error) {
	body, _ := json.Marshal(map[string]any{"name": name, "group_name": groupName})
	result, err := apiRequest("POST", "/v1/resources/resolve", body, cardKey)
	if err != nil {
		return "", err
	}
	url, _ := result["url"].(string)
	return url, nil
}

// AssetList lists assets in a group.
func AssetList(groupName, cardKey string) ([]any, error) {
	result, err := apiRequest("GET", "/v1/resources?group_name="+groupName, nil, cardKey)
	if err != nil {
		return nil, err
	}
	items, _ := result["items"].([]any)
	return items, nil
}

func UnderstandResource(groupName, objectName, cardKey string) (map[string]any, error) {
	return apiRequest("POST", "/v1/resources/understand", map[string]any{
		"group_name":  groupName,
		"object_name": objectName,
	}, cardKey)
}

func DeleteResource(groupName, stem, cardKey string) (map[string]any, error) {
	path := "/v1/resources?group_name=" + url.QueryEscape(groupName) + "&stem=" + url.QueryEscape(stem)
	return apiRequest("DELETE", path, nil, cardKey)
}

func ResolveRuntime(name, osName, arch, version, cardKey string) (*RuntimeResolve, error) {
	payload := map[string]any{
		"name":    name,
		"os":      osName,
		"arch":    arch,
		"version": version,
	}
	result, err := apiRequest("POST", "/v1/client-runtime/resolve", payload, cardKey)
	if err != nil {
		return nil, err
	}
	resolved, err := mapToStruct[RuntimeResolve](result)
	if err != nil {
		return nil, fmt.Errorf("parse runtime response failed: %w", err)
	}
	return &resolved, nil
}

func ListClientResources(resourceType, tag, cardKey string) ([]ClientResource, error) {
	path := "/v1/client-resources"
	sep := "?"
	if resourceType != "" {
		path += sep + "type=" + url.QueryEscape(resourceType)
		sep = "&"
	}
	if tag != "" {
		path += sep + "tag=" + url.QueryEscape(tag)
	}
	result, err := apiRequest("GET", path, nil, cardKey)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(result["items"])
	if err != nil {
		return nil, err
	}
	var items []ClientResource
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse resources response failed: %w", err)
	}
	return items, nil
}

func SignClientResource(resourceID, cardKey string) (*ClientResource, error) {
	result, err := apiRequest("POST", "/v1/client-resources/sign", map[string]any{
		"resource_id": resourceID,
	}, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[ClientResource](result)
	if err != nil {
		return nil, fmt.Errorf("parse resource sign response failed: %w", err)
	}
	return &item, nil
}

func ClientResourceDefaults(cardKey string) (*ClientDefaults, error) {
	result, err := apiRequest("GET", "/v1/client-resources/defaults", nil, cardKey)
	if err != nil {
		return nil, err
	}
	defaults, ok := result["defaults"].(map[string]any)
	if !ok {
		defaults = result
	}
	item, err := mapToStruct[ClientDefaults](defaults)
	if err != nil {
		return nil, fmt.Errorf("parse client defaults response failed: %w", err)
	}
	return &item, nil
}

func RewriteScript(text, length, model, cardKey string) (*ScriptRewriteResponse, error) {
	payload := map[string]any{
		"input": map[string]any{
			"text": text,
		},
		"options": map[string]any{
			"length": length,
		},
	}
	if model != "" {
		payload["options"].(map[string]any)["model"] = model
	}
	result, err := apiRequest("POST", "/v1/agent/script/rewrite", payload, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[ScriptRewriteResponse](result)
	if err != nil {
		return nil, fmt.Errorf("parse script rewrite response failed: %w", err)
	}
	return &item, nil
}

func RunAgentAbility(path string, input map[string]any, options map[string]any, cardKey string) (*AgentAbilityResponse, error) {
	payload := map[string]any{
		"input": input,
	}
	if options != nil {
		payload["options"] = options
	}
	result, err := apiRequest("POST", path, payload, cardKey)
	if err != nil {
		return nil, err
	}
	item, err := mapToStruct[AgentAbilityResponse](result)
	if err != nil {
		return nil, fmt.Errorf("parse agent ability response failed: %w", err)
	}
	return &item, nil
}

func RunResearch(roleDescription, mode, dateRange, cardKey string) (*ResearchResponse, error) {
	mode = strings.TrimSpace(strings.ToLower(mode))
	path := "/v1/content-research/precise-search"
	ability := "research.precise_search"
	if mode == "expanded" {
		path = "/v1/content-research/expanded-search"
		ability = "research.expanded_search"
	}
	result, err := apiRequest("POST", path, map[string]any{
		"role_description": roleDescription,
		"date_range":       dateRange,
	}, cardKey)
	if err != nil {
		return nil, err
	}
	return &ResearchResponse{
		Ability: ability,
		Result:  result,
		Usage:   map[string]any{},
	}, nil
}

func DouyinDownloadCookie(cardKey string) (string, error) {
	result, err := apiRequest("GET", "/api/douyin/download-cookie", nil, cardKey)
	if err != nil {
		return "", err
	}
	cookie, _ := result["cookie"].(string)
	if cookie == "" {
		return "", fmt.Errorf("empty douyin download cookie")
	}
	return cookie, nil
}

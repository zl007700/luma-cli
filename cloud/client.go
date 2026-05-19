package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// DefaultBaseURL is the default cloud API endpoint.
// Can be overridden via LUMA_API_URL environment variable.
var DefaultBaseURL = "https://api.pikgeo.com"

// BaseURL returns the active API base URL, checking env var first.
func BaseURL() string {
	if url := os.Getenv("LUMA_API_URL"); url != "" {
		return url
	}
	return DefaultBaseURL
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

type ScriptRewriteResponse struct {
	Ability   string         `json:"ability"`
	RequestID string         `json:"request_id"`
	Result    map[string]any `json:"result"`
	Usage     map[string]any `json:"usage"`
}

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

func guessMimeType(filePath string) string {
	switch ext := filepath.Ext(filePath); ext {
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "video/mp4"
	case ".wav":
		return "audio/wav"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}

func guessResourceType(mimeType string) string {
	if mimeType == "video/mp4" || mimeType == "video/quicktime" || mimeType == "video/x-msvideo" {
		return "video"
	}
	if mimeType == "audio/mpeg" || mimeType == "audio/wav" || mimeType == "audio/mp3" {
		return "audio"
	}
	if mimeType == "image/jpeg" || mimeType == "image/png" {
		return "image"
	}
	return "other"
}

// UploadFile uploads a local file to the cloud and returns the object key.
func UploadFile(filePath, cardKey, groupName string) (string, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}
	fileSize := stat.Size()
	mimeType := guessMimeType(filePath)
	resourceType := guessResourceType(mimeType)

	filename := filepath.Base(filePath)

	if fileSize >= multipartThreshold {
		return uploadMultipart(filePath, filename, groupName, resourceType, mimeType, cardKey)
	}

	asciiFilename := filename
	for _, c := range filename {
		if c >= 128 {
			asciiFilename = fmt.Sprintf("upload_%d%s", time.Now().UnixNano()%100000, filepath.Ext(filePath))
			break
		}
	}

	return uploadSinglePart(filePath, asciiFilename, groupName, resourceType, mimeType, cardKey)
}

func uploadSinglePart(filePath, filename, groupName, resourceType, mimeType, cardKey string) (string, error) {
	signBody, _ := json.Marshal(map[string]any{
		"group_name":    groupName,
		"filename":      filename,
		"resource_type": resourceType,
		"mime_type":     mimeType,
		"variant":       "original",
	})
	signResult, err := apiRequest("POST", "/v1/resources/upload-sign", signBody, cardKey)
	if err != nil {
		return "", fmt.Errorf("upload sign failed: %w", err)
	}
	uploadURL, _ := signResult["upload_url"].(string)
	objectKey, _ := signResult["object_key"].(string)
	if uploadURL == "" || objectKey == "" {
		return "", fmt.Errorf("invalid upload sign response")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	req, err := http.NewRequest("PUT", uploadURL, file)
	if err != nil {
		return "", fmt.Errorf("create upload request failed: %w", err)
	}
	req.Header.Set("Content-Type", mimeType)
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("file upload failed: %w", err)
	}
	httpResp.Body.Close()
	if httpResp.StatusCode >= 400 {
		return "", fmt.Errorf("upload returned status %d", httpResp.StatusCode)
	}

	saveBody, _ := json.Marshal(map[string]any{
		"resource_id":   signResult["resource_id"],
		"group_name":    groupName,
		"resource_type": resourceType,
		"mime_type":     mimeType,
		"object_key":    objectKey,
		"filename":      filename,
	})
	_, err = apiRequest("POST", "/v1/resources", saveBody, cardKey)
	if err != nil {
		return "", fmt.Errorf("save resource record failed: %w", err)
	}

	return objectKey, nil
}

func uploadMultipart(filePath, filename, groupName, resourceType, mimeType, cardKey string) (string, error) {
	asciiFilename := fmt.Sprintf("upload_%d%s", time.Now().UnixNano()%100000, filepath.Ext(filePath))

	initBody, _ := json.Marshal(map[string]any{
		"group_name":    groupName,
		"filename":      asciiFilename,
		"resource_type": resourceType,
		"mime_type":     mimeType,
		"variant":       "original",
	})
	initResult, err := apiRequest("POST", "/v1/resources/multipart/init", initBody, cardKey)
	if err != nil {
		return "", fmt.Errorf("multipart init failed: %w", err)
	}

	resourceID, _ := initResult["resource_id"].(string)
	uploadID, _ := initResult["upload_id"].(string)
	objectKey, _ := initResult["object_key"].(string)
	if resourceID == "" || uploadID == "" || objectKey == "" {
		return "", fmt.Errorf("invalid multipart init response")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	const partSize = 5 * 1024 * 1024 // 5MB parts
	var parts []map[string]any
	partNumber := 1
	offset := int64(0)

	for {
		stat, _ := file.Stat()
		remaining := stat.Size() - offset
		if remaining <= 0 {
			break
		}
		currentSize := int64(partSize)
		if remaining < currentSize {
			currentSize = remaining
		}

		signBody, _ := json.Marshal(map[string]any{
			"object_key":  objectKey,
			"upload_id":   uploadID,
			"part_number": partNumber,
		})
		signResult, err := apiRequest("POST", "/v1/resources/multipart/sign-part", signBody, cardKey)
		if err != nil {
			return "", fmt.Errorf("multipart sign part failed: %w", err)
		}
		partURL, _ := signResult["upload_url"].(string)
		if partURL == "" {
			return "", fmt.Errorf("invalid part sign response")
		}

		chunk := make([]byte, currentSize)
		if _, err := file.ReadAt(chunk, offset); err != nil {
			return "", fmt.Errorf("read chunk failed: %w", err)
		}

		req, err := http.NewRequest("PUT", partURL, bytes.NewReader(chunk))
		if err != nil {
			return "", fmt.Errorf("create part request failed: %w", err)
		}
		req.Header.Set("Content-Type", mimeType)
		httpResp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("multipart part upload failed: %w", err)
		}
		httpResp.Body.Close()
		if httpResp.StatusCode >= 400 {
			return "", fmt.Errorf("part upload returned status %d", httpResp.StatusCode)
		}

		etag := httpResp.Header.Get("ETag")
		if etag == "" {
			etag = httpResp.Header.Get("Etag")
		}
		parts = append(parts, map[string]any{
			"part_number": partNumber,
			"etag":        etag,
		})

		offset += currentSize
		partNumber++
	}

	completeBody, _ := json.Marshal(map[string]any{
		"resource_id": resourceID,
		"group_name":  groupName,
		"object_key":  objectKey,
		"upload_id":   uploadID,
		"parts":       parts,
	})
	_, err = apiRequest("POST", "/v1/resources/multipart/complete", completeBody, cardKey)
	if err != nil {
		return "", fmt.Errorf("multipart complete failed: %w", err)
	}

	return objectKey, nil
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

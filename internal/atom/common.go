package atom

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ResultURL extracts a downloadable result URL from a completed task status.
func ResultURL(status map[string]any) string {
	if output, ok := status["output"].(map[string]any); ok {
		if url, ok := output["result_url"].(string); ok && url != "" {
			return url
		}
		if result, ok := output["result"].(map[string]any); ok {
			if url, ok := result["url"].(string); ok && url != "" {
				return url
			}
		}
	}
	if url, ok := status["result_url"].(string); ok && url != "" {
		return url
	}
	return ""
}

// ResultObjectKey extracts the result object key from a completed task status.
func ResultObjectKey(status map[string]any) string {
	if key, ok := status["result_object_key"].(string); ok && key != "" {
		return key
	}
	if output, ok := status["output"].(map[string]any); ok {
		if key, ok := output["result_object_key"].(string); ok && key != "" {
			return key
		}
	}
	return ""
}

// StripResourcePrefix removes the API-added resource prefix from object keys.
func StripResourcePrefix(objectKey, userID string) string {
	prefix := "prod/resource/" + userID + "/"
	return strings.TrimPrefix(objectKey, prefix)
}

// DownloadFile downloads a URL to a local path.
func DownloadFile(url, outputPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("write file: %w", err)
	}
	return nil
}

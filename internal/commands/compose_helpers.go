package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/internal/clientruntime"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func installedFFmpegPath() (string, error) {
	runtimeInfo, err := clientruntime.CurrentRuntime("ffmpeg")
	if err == nil && runtimeInfo.ExecutablePath != "" {
		return runtimeInfo.ExecutablePath, nil
	}
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("ffmpeg not found. Run: luma-cli runtime install ffmpeg")
}

func resolveLocalCachedOrCloudResource(value string, cfg *config) (string, error) {
	if value == "" {
		return "", fmt.Errorf("empty resource")
	}
	if _, err := os.Stat(value); err == nil {
		return filepath.Abs(value)
	}
	if cached, err := clientruntime.CurrentResource(value); err == nil {
		return cached.Path, nil
	}
	if cfg == nil {
		return "", fmt.Errorf("not a local file or cached resource: %s. Run 'luma-cli auth login <phone_or_account>' to cache cloud resources", value)
	}
	cached, err := clientruntime.CacheResource(cfg.CardKey, value)
	if err != nil {
		return "", fmt.Errorf("not a local file or cloud resource: %s", value)
	}
	return cached.Path, nil
}

func isObjectKeyRef(value string) bool {
	value = strings.TrimSpace(value)
	return strings.Contains(value, "/") || strings.HasPrefix(value, "prod/") || strings.HasPrefix(value, "resource/")
}

func writeSimpleResult(data map[string]any) {
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: data})
		return
	}
	if p, ok := data["output_path"].(string); ok {
		fmt.Printf("Output: %s\n", p)
	}
	for _, key := range []string{"count", "scene_count", "matched_count", "mode", "task_id", "csv_path", "group_path"} {
		if value, ok := data[key]; ok && strAny(value) != "" {
			fmt.Printf("%s: %v\n", key, value)
		}
	}
}

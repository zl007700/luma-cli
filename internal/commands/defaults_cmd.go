package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/clientruntime"
	"github.com/luma-cli/lumer-cli/internal/output"
)

const fallbackCoverFontResourceID = "font_22b2e39414"

func cmdDefaults(args []string) error {
	if len(args) < 1 || args[0] != "show" {
		fmt.Println("usage: luma-cli defaults show")
		return nil
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	defaults := loadClientDefaults(cfg)
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: defaults})
		return nil
	}
	data, _ := json.MarshalIndent(defaults, "", "  ")
	fmt.Println(string(data))
	return nil
}

func loadClientDefaults(cfg *config) *cloud.ClientDefaults {
	defaults := builtinClientDefaults()
	if cfg == nil {
		return defaults
	}
	remote, err := cloud.ClientResourceDefaults(cfg.CardKey)
	if err != nil || remote == nil {
		return defaults
	}
	mergeClientDefaults(defaults, remote)
	return defaults
}

func builtinClientDefaults() *cloud.ClientDefaults {
	return &cloud.ClientDefaults{
		Subtitle: cloud.SubtitleDefaults{
			Font:             fallbackCoverFontResourceID,
			FontSize:         80,
			SideMargin:       60,
			BottomMargin:     500,
			Color:            "#FDFDFF",
			StrokeColor:      "#1F0101",
			HighlightColor:   "#FFD95A",
			MaxChars:         15,
			EffectsEnabled:   true,
			HighlightEnabled: true,
		},
		Cover: cloud.CoverDefaults{
			Font:         fallbackCoverFontResourceID,
			TitleSize:    72,
			SubtitleSize: 38,
		},
		BGM: cloud.BGMDefaults{
			VoiceVolume: 1.2,
			BGMVolume:   0.15,
		},
	}
}

func mergeClientDefaults(dst, src *cloud.ClientDefaults) {
	if src.Subtitle.Font != "" {
		dst.Subtitle.Font = src.Subtitle.Font
	}
	if src.Subtitle.FontSize > 0 {
		dst.Subtitle.FontSize = src.Subtitle.FontSize
	}
	if src.Subtitle.SideMargin > 0 {
		dst.Subtitle.SideMargin = src.Subtitle.SideMargin
	}
	if src.Subtitle.BottomMargin > 0 {
		dst.Subtitle.BottomMargin = src.Subtitle.BottomMargin
	}
	if src.Subtitle.Color != "" {
		dst.Subtitle.Color = src.Subtitle.Color
	}
	if src.Subtitle.StrokeColor != "" {
		dst.Subtitle.StrokeColor = src.Subtitle.StrokeColor
	}
	if src.Subtitle.HighlightColor != "" {
		dst.Subtitle.HighlightColor = src.Subtitle.HighlightColor
	}
	if src.Subtitle.MaxChars > 0 {
		dst.Subtitle.MaxChars = src.Subtitle.MaxChars
	}
	dst.Subtitle.EffectsEnabled = src.Subtitle.EffectsEnabled
	dst.Subtitle.HighlightEnabled = src.Subtitle.HighlightEnabled

	if src.Cover.Template != "" {
		dst.Cover.Template = src.Cover.Template
	}
	if src.Cover.Font != "" {
		dst.Cover.Font = src.Cover.Font
	}
	if src.Cover.TitleSize > 0 {
		dst.Cover.TitleSize = src.Cover.TitleSize
	}
	if src.Cover.SubtitleSize > 0 {
		dst.Cover.SubtitleSize = src.Cover.SubtitleSize
	}

	if src.BGM.Default != "" {
		dst.BGM.Default = src.BGM.Default
	}
	if src.BGM.VoiceVolume > 0 {
		dst.BGM.VoiceVolume = src.BGM.VoiceVolume
	}
	if src.BGM.BGMVolume > 0 {
		dst.BGM.BGMVolume = src.BGM.BGMVolume
	}
}

func cacheDefaultResource(resourceID string, cfg *config) (string, error) {
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return "", fmt.Errorf("default resource is empty")
	}
	return resolveLocalCachedOrCloudResource(resourceID, cfg)
}

func copyFontToDir(fontPath, targetDir string) (string, error) {
	if strings.TrimSpace(fontPath) == "" {
		return "", nil
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}
	target := filepath.Join(targetDir, filepath.Base(fontPath))
	if filepath.Clean(fontPath) == filepath.Clean(target) {
		return target, nil
	}
	data, err := os.ReadFile(fontPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(target, data, 0644); err != nil {
		return "", err
	}
	return target, nil
}

func fontNameFromPath(fontPath string) string {
	base := filepath.Base(fontPath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if name == "" {
		return "Microsoft YaHei"
	}
	return name
}

func formatVolume(value float64, fallback string) string {
	if value <= 0 {
		return fallback
	}
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func currentCachedResourcePath(resourceID string) string {
	cached, err := clientruntime.CurrentResource(resourceID)
	if err != nil {
		return ""
	}
	return cached.Path
}

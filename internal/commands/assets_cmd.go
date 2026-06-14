package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	appconfig "github.com/luma-cli/lumer-cli/internal/config"
	"github.com/luma-cli/lumer-cli/internal/output"
)

type cachedAsset struct {
	AssetID     string `json:"asset_id"`
	Kind        string `json:"kind"`
	DisplayName string `json:"display_name"`
	Filename    string `json:"filename"`
	Path        string `json:"path"`
	SHA256      string `json:"sha256"`
}

func cmdAssets(args []string) error {
	if len(args) < 1 {
		printAssetsUsage()
		return nil
	}
	switch args[0] {
	case "upload":
		return cmdAssetsUpload(args[1:])
	case "search", "list":
		return cmdAssetsSearch(args[1:])
	case "groups", "group-list":
		return cmdAssetsGroups(args[1:])
	case "resolve":
		return cmdAssetsResolve(args[1:])
	case "sign":
		return cmdAssetsSign(args[1:])
	case "cache":
		return cmdAssetsCache(args[1:])
	case "delete", "remove", "rm":
		return cmdAssetsDelete(args[1:])
	default:
		fmt.Printf("unknown assets subcommand: %s\n\n", args[0])
		printAssetsUsage()
		return nil
	}
}

func cmdAssetsUpload(raw []string) error {
	parsed := cmdutil.Parse(raw)
	filePath := parsed.Pos(0)
	if filePath == "" {
		fmt.Println("usage: luma-cli assets upload <file> --kind <kind> [--group <name>] [--name <display_name>]")
		return nil
	}
	kind := strings.TrimSpace(parsed.String("kind", ""))
	if kind == "" {
		return output.ErrValidation("--kind is required")
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	group := normalizeAssetGroupName(parsed.String("group", parsed.String("group-name", "")))
	displayName := strings.TrimSpace(parsed.String("name", parsed.String("display-name", "")))
	metadata := map[string]any{}
	if metaJSON := strings.TrimSpace(parsed.String("metadata-json", "")); metaJSON != "" {
		if err := json.Unmarshal([]byte(metaJSON), &metadata); err != nil {
			return output.ErrValidation("--metadata-json must be a JSON object: %v", err)
		}
	}
	item, err := cloud.AssetsUploadFile(filePath, kind, group, displayName, cfg.CardKey, metadata)
	if err != nil {
		return output.ErrNetwork("asset upload failed: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: item})
		return nil
	}
	fmt.Printf("Uploaded asset: %s\n", item.AssetID)
	fmt.Printf("Status: %s\n", item.Status)
	if item.ObjectKey != "" {
		fmt.Printf("Object key: %s\n", item.ObjectKey)
	}
	if item.Status != "active" {
		fmt.Println("Asset is waiting for backend intake; run `luma-cli assets search --kind ...` after asset-worker marks it active.")
	}
	return nil
}

func cmdAssetsSearch(raw []string) error {
	parsed := cmdutil.Parse(raw)
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	limit, err := parsed.Int("limit", 30)
	if err != nil {
		return output.ErrValidation("%v", err)
	}
	kind := strings.TrimSpace(parsed.String("kind", parsed.Pos(0)))
	group := normalizeAssetGroupName(parsed.String("group", parsed.String("group-name", "")))
	scope := strings.TrimSpace(parsed.String("scope", ""))
	result, err := cloud.AssetsSearch(kind, group, scope, limit, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("assets search failed: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: result})
		return nil
	}
	if len(result.Items) == 0 {
		fmt.Println("No assets found.")
		return nil
	}
	fmt.Printf("%-28s %-16s %-16s %-20s %-32s %s\n", "ID", "KIND", "GROUP", "NAME", "PROBE", "CAPTION")
	for _, item := range result.Items {
		fmt.Printf("%-28s %-16s %-16s %-20s %-32s %s\n", item.AssetID, item.Kind, item.GroupName, registryAssetDisplayName(item), oneLineAssetProbe(item), oneLineAssetCaption(item))
	}
	return nil
}

func cmdAssetsGroups(raw []string) error {
	parsed := cmdutil.Parse(raw)
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	kind := strings.TrimSpace(parsed.String("kind", parsed.Pos(0)))
	scope := strings.TrimSpace(parsed.String("scope", ""))
	result, err := cloud.AssetsGroups(kind, scope, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("assets groups failed: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: result})
		return nil
	}
	if len(result.Items) == 0 {
		fmt.Println("No asset groups found.")
		return nil
	}
	fmt.Printf("%-24s %s\n", "GROUP", "DISPLAY")
	for _, item := range result.Items {
		fmt.Printf("%-24s %s\n", item.GroupName, item.DisplayName)
	}
	return nil
}

func cmdAssetsResolve(raw []string) error {
	if len(raw) < 1 {
		fmt.Println("usage: luma-cli assets resolve <asset_id> [asset_id...]")
		return nil
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	result, err := cloud.AssetsResolve(raw, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("assets resolve failed: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: result})
		return nil
	}
	for _, item := range result.Items {
		fmt.Printf("%s\t%s\t%s\t%s\t%s\t%s\n", item.AssetID, item.Kind, item.ObjectKey, registryAssetDisplayName(item), oneLineAssetProbe(item), oneLineAssetCaption(item))
	}
	return nil
}

func cmdAssetsSign(raw []string) error {
	if len(raw) < 1 {
		fmt.Println("usage: luma-cli assets sign <asset_id>")
		return nil
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	item, err := cloud.AssetsSign(raw[0], cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("assets sign failed: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: item})
		return nil
	}
	fmt.Println(item.DownloadURL)
	return nil
}

func cmdAssetsCache(raw []string) error {
	if len(raw) < 1 {
		fmt.Println("usage: luma-cli assets cache <asset_id> [--output <path>]")
		return nil
	}
	parsed := cmdutil.Parse(raw)
	assetID := parsed.Pos(0)
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	item, err := cloud.AssetsSign(assetID, cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("assets sign failed: %v", err)
	}
	if item.DownloadURL == "" {
		return output.ErrNetwork("asset sign response missing download_url")
	}
	target := strings.TrimSpace(parsed.String("output", ""))
	if target == "" {
		root, err := assetCacheRoot()
		if err != nil {
			return output.ErrSystem("%v", err)
		}
		filename := item.Filename
		if filename == "" {
			filename = item.AssetID
		}
		target = filepath.Join(root, item.AssetID, filepath.Base(filename))
	}
	abs, err := absoluteOutputPath(target)
	if err != nil {
		return output.ErrValidation("invalid output path: %v", err)
	}
	if err := atom.DownloadFile(item.DownloadURL, abs); err != nil {
		return output.ErrNetwork("asset download failed: %v", err)
	}
	cached := cachedAsset{
		AssetID:     item.AssetID,
		Kind:        item.Kind,
		DisplayName: item.DisplayName,
		Filename:    item.Filename,
		Path:        abs,
		SHA256:      item.SHA256,
	}
	if err := writeAssetCacheCurrent(cached); err != nil {
		return output.ErrSystem("write cache metadata failed: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: cached})
		return nil
	}
	fmt.Printf("Cached %s\n", cached.AssetID)
	fmt.Printf("Path: %s\n", cached.Path)
	return nil
}

func cmdAssetsDelete(raw []string) error {
	if len(raw) < 1 {
		fmt.Println("usage: luma-cli assets delete <asset_id>")
		return nil
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	result, err := cloud.AssetsDelete(raw[0], cfg.CardKey)
	if err != nil {
		return output.ErrNetwork("assets delete failed: %v", err)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: result})
		return nil
	}
	fmt.Printf("Deleted asset: %s\n", raw[0])
	return nil
}

func assetCacheRoot() (string, error) {
	dir, err := appconfig.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "assets"), nil
}

func writeAssetCacheCurrent(cached cachedAsset) error {
	root, err := assetCacheRoot()
	if err != nil {
		return err
	}
	target := filepath.Join(root, cached.AssetID, "current.json")
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, 0644)
}

func registryAssetDisplayName(item cloud.AssetItem) string {
	if strings.TrimSpace(item.DisplayName) != "" {
		return strings.TrimSpace(item.DisplayName)
	}
	semantic, _ := item.Metadata["semantic"].(map[string]any)
	if semantic == nil {
		return ""
	}
	for _, key := range []string{"name", "title"} {
		if text := strings.TrimSpace(fmt.Sprint(semantic[key])); text != "" && text != "<nil>" {
			return text
		}
	}
	return ""
}

func oneLineAssetCaption(item cloud.AssetItem) string {
	semantic, _ := item.Metadata["semantic"].(map[string]any)
	if semantic == nil {
		return ""
	}
	text := ""
	for _, key := range []string{"caption", "summary", "description", "short_des"} {
		text = strings.TrimSpace(fmt.Sprint(semantic[key]))
		if text != "" && text != "<nil>" {
			break
		}
	}
	text = strings.Join(strings.Fields(text), " ")
	if len([]rune(text)) > 240 {
		runes := []rune(text)
		text = string(runes[:240]) + "..."
	}
	return text
}

func oneLineAssetProbe(item cloud.AssetItem) string {
	probe := registryAssetMap(item.Metadata, "probe")
	if len(probe) == 0 {
		probe = registryAssetMap(item.Metadata, "technical")
	}
	parts := []string{}
	resolution := strings.TrimSpace(fmt.Sprint(probe["resolution"]))
	if resolution == "" || resolution == "<nil>" {
		width := registryAssetNumber(probe["width"], item.Width)
		height := registryAssetNumber(probe["height"], item.Height)
		if width != "" && height != "" {
			resolution = width + "x" + height
		}
	}
	if resolution != "" && resolution != "<nil>" {
		parts = append(parts, resolution)
	}
	if fps := registryAssetNumber(probe["fps"], nil); fps != "" && fps != "0" {
		parts = append(parts, fps+"fps")
	}
	if duration := registryAssetNumber(probe["duration_sec"], item.Duration); duration != "" && duration != "0" {
		parts = append(parts, duration+"s")
	}
	for _, key := range []string{"video_codec", "audio_codec", "format", "format_name"} {
		value := strings.TrimSpace(fmt.Sprint(probe[key]))
		if value != "" && value != "<nil>" {
			parts = append(parts, value)
			break
		}
	}
	return strings.Join(parts, " ")
}

func registryAssetMap(parent map[string]any, key string) map[string]any {
	if parent == nil {
		return nil
	}
	value, _ := parent[key].(map[string]any)
	return value
}

func registryAssetNumber(value any, fallback any) string {
	if value == nil {
		value = fallback
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" {
		return ""
	}
	if strings.HasSuffix(text, ".000") {
		text = strings.TrimSuffix(text, ".000")
	}
	return text
}

func printAssetsUsage() {
	fmt.Println("luma-cli assets <subcommand>")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  upload <file> --kind <kind> [--group <name>] [--name <display_name>]")
	fmt.Println("  search [--kind <kind>] [--group <name>] [--scope system|user|job] [--limit <n>]")
	fmt.Println("  groups [--kind <kind>] [--scope system|user|job]")
	fmt.Println("  resolve <asset_id> [asset_id...]")
	fmt.Println("  sign <asset_id>")
	fmt.Println("  cache <asset_id> [--output <path>]")
	fmt.Println("  delete <asset_id>")
}

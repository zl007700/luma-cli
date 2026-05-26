package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

type voiceView struct {
	Name      string `json:"name"`
	ObjectKey string `json:"object_key,omitempty"`
	Remark    string `json:"remark,omitempty"`
}

func cmdVoice(args []string) error {
	if len(args) < 1 {
		printVoiceUsage()
		return nil
	}
	cfg, err := requireConfig()
	if err != nil {
		return err
	}
	switch args[0] {
	case "clone":
		cmdVoiceClone(args[1:], cfg)
	case "list":
		cmdVoiceList(args[1:], cfg)
	default:
		printVoiceUsage()
	}
	return nil
}

func cmdVoiceClone(raw []string, cfg *config) error {
	parsed := cmdutil.Parse(raw)
	audioPath := strings.TrimSpace(parsed.Pos(0))
	if audioPath == "" {
		audioPath = strings.TrimSpace(parsed.String("audio", ""))
	}
	if audioPath == "" {
		fmt.Println("usage: luma-cli voice clone <sample_audio> [--name <voice_name>] [--verbose]")
		return nil
	}
	if _, err := os.Stat(audioPath); err != nil {
		return output.ErrValidation("audio file not found: %s\n", audioPath)
	}
	name := strings.TrimSpace(parsed.String("name", ""))
	objectKey, err := cloud.UploadFileWithName(audioPath, cfg.CardKey, "voice", name)
	if err != nil {
		return output.ErrNetwork("voice upload failed: %v\n", err)
	}
	if name == "" {
		name = atom.AssetFriendlyName(objectKey)
	}
	view := voiceView{Name: name, ObjectKey: atom.NormalizeResourceKey(objectKey, cfg.CardKey)}
	recordProjectArtifact("voice", view.ObjectKey, "voice.clone")
	if runtimeOpts.JSON {
		if !parsed.Has("verbose") {
			view.ObjectKey = ""
		}
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: view})
		return nil
	}
	fmt.Printf("Voice cloned: %s\n", view.Name)
	fmt.Printf("Use with: luma-cli tts \"text\" --voice %s\n", view.Name)
	if parsed.Has("verbose") {
		fmt.Printf("Object key: %s\n", view.ObjectKey)
	}
	return nil
}

func cmdVoiceList(raw []string, cfg *config) error {
	parsed := cmdutil.Parse(raw)
	verbose := parsed.Has("verbose")
	items, err := cloud.AssetList("voice", cfg.CardKey)
	if err != nil {
		return output.ErrSystem("%v\n", err)
	}
	views := make([]voiceView, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		key := atom.ResourceKeyFromMap(m, cfg.CardKey)
		name := atom.AssetFriendlyName(key)
		if filename, _ := m["filename"].(string); filename != "" {
			name = atom.AssetFriendlyName(filename)
		}
		view := voiceView{Name: name, ObjectKey: key}
		if remark, _ := m["remark"].(string); remark != "" {
			view.Remark = remark
		}
		if !verbose {
			view.ObjectKey = ""
		}
		views = append(views, view)
	}
	if runtimeOpts.JSON {
		_ = output.WriteJSON(os.Stdout, output.Envelope{OK: true, Data: map[string]any{"items": views}})
		return nil
	}
	if len(views) == 0 {
		fmt.Println("No voices found.")
		return nil
	}
	if verbose {
		fmt.Printf("%-24s %s\n", "NAME", "OBJECT_KEY")
		for _, item := range views {
			fmt.Printf("%-24s %s\n", item.Name, item.ObjectKey)
		}
		return nil
	}
	fmt.Println("NAME")
	for _, item := range views {
		fmt.Println(item.Name)
	}
	return nil
}

func printVoiceUsage() {
	fmt.Println("luma-cli voice <subcommand>")
	fmt.Println("  clone <sample_audio> [--name <voice_name>] [--verbose]")
	fmt.Println("  list [--verbose]")
}

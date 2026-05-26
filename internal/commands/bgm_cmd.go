package commands

import (
	"fmt"
	"os/exec"

	"github.com/luma-cli/lumer-cli/internal/cmdutil"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func cmdBGM(args []string) error {
	if len(args) < 1 || args[0] != "mix" {
		fmt.Println("usage: luma-cli bgm mix <video> [--bgm <file_or_resource_id>] [--output <mp4>] [--voice-volume 1.0] [--bgm-volume 0.25]")
		return nil
	}
	parsed := cmdutil.Parse(args[1:])
	videoPath := parsed.Pos(0)
	bgmValue := parsed.String("bgm", "")
	if videoPath == "" {
		fmt.Println("usage: luma-cli bgm mix <video> [--bgm <file_or_resource_id>] [--output <mp4>]")
		return nil
	}
	cfg := loadConfig()
	defaults := loadClientDefaults(cfg)
	if bgmValue == "" {
		bgmValue = defaults.BGM.Default
	}
	if bgmValue == "" {
		return output.ErrValidation("no BGM specified and no default BGM configured")
	}
	outputPath := parsed.String("output", "step6_bgm.mp4")
	voiceVolume := parsed.String("voice-volume", formatVolume(defaults.BGM.VoiceVolume, "1.0"))
	bgmVolume := parsed.String("bgm-volume", formatVolume(defaults.BGM.BGMVolume, "0.25"))
	bgmPath, err := resolveLocalCachedOrCloudResource(bgmValue, cfg)
	if err != nil {
		return output.ErrNetwork("resolve bgm failed: %v", err)
	}
	absOut, err := ensureOutputDir(outputPath)
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	ffmpeg, err := installedFFmpegPath()
	if err != nil {
		return output.ErrSystem("%v", err)
	}
	filter := fmt.Sprintf("[0:a]volume=%s[a0];[1:a]volume=%s,aloop=loop=-1:size=2e+09[a1];[a0][a1]amix=inputs=2:duration=first:dropout_transition=2[aout]", voiceVolume, bgmVolume)
	cmd := exec.Command(ffmpeg, "-y", "-i", videoPath, "-i", bgmPath, "-filter_complex", filter, "-map", "0:v", "-map", "[aout]", "-c:v", "copy", "-c:a", "aac", "-shortest", absOut)
	if data, err := cmd.CombinedOutput(); err != nil {
		return output.ErrSystem("ffmpeg bgm mix failed: %v\n%s", err, string(data))
	}
	// Rename output to include content hash for traceability.
	if hashed, err := hashSuffixFile(absOut); err == nil {
		absOut = hashed
	}

	recordProjectArtifact("bgm", absOut, "bgm.mix")
	writeSimpleResult(map[string]any{"output_path": absOut, "bgm_path": bgmPath})
	return nil
}

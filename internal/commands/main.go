package commands

import (
	"fmt"
	"os"

	"github.com/luma-cli/lumer-cli/internal/output"
	"github.com/luma-cli/lumer-cli/internal/skillsync"
)

var version = "dev"

func printUsage() {
	fmt.Println("luma-cli - Luma CLI tool")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  luma-cli version                 Show version")
	fmt.Println("  luma-cli help                    Show help")
	fmt.Println("  luma-cli update                  Update CLI and sync skills")
	fmt.Println("  luma-cli skills sync             Install or update agent skills")
	fmt.Println("  luma-cli skills status           Show skills sync status")
	fmt.Println("  luma-cli auth login <key>        Save card key")
	fmt.Println("  luma-cli auth status             Show login status")
	fmt.Println("  luma-cli agent run <ability>     Run backend-owned agent ability")
	fmt.Println("  luma-cli asr <file>              Transcribe audio or video")
	fmt.Println("  luma-cli asset upload <file>     Upload an asset")
	fmt.Println("  luma-cli asset list [group]      List assets")
	fmt.Println("  luma-cli asset understand <name> Understand uploaded asset")
	fmt.Println("  luma-cli bgm mix <video> --bgm <file_or_resource_id>")
	fmt.Println("  luma-cli cover frame <video>     Extract a cover frame")
	fmt.Println("  luma-cli defaults show           Show product default settings")
	fmt.Println("  luma-cli pip scan|plan|render    Plan and render picture-in-picture")
	fmt.Println("  luma-cli tts <text>              Synthesize speech")
	fmt.Println("  luma-cli voice clone <audio>     Clone a voice from reference audio")
	fmt.Println("  luma-cli lipsync --avatar <name> Create a lip-sync video")
	fmt.Println("  luma-cli material describe <dir> Build local material metadata")
	fmt.Println("  luma-cli material understand <file> Upload and understand one material")
	fmt.Println("  luma-cli material merge          Merge material metadata")
	fmt.Println("  luma-cli enhance <video>         Enhance a video")
	fmt.Println("  luma-cli task status <task_id>   Show task status")
	fmt.Println("  luma-cli download <url> [file]   Download a file")
	fmt.Println("  luma-cli social download <link>  Download Douyin video via share link")
	fmt.Println("  luma-cli douyin <share_link>     Douyin helpers")
	fmt.Println("  luma-cli viral text <text> [--persona <name>] [--length short|medium|long]")
	fmt.Println("  luma-cli viral video <file> [--persona <name>] [--length short|medium|long]")
	fmt.Println("  luma-cli project create <name> [--dir <path>]")
	fmt.Println("  luma-cli project list            List projects")
	fmt.Println("  luma-cli project use <name>      Switch active project")
	fmt.Println("  luma-cli project info            Show active project")
	fmt.Println("  luma-cli project clean           Clean project temp files")
	fmt.Println("  luma-cli runtime install ffmpeg  Install local video runtime")
	fmt.Println("  luma-cli resource list [type]    List cloud-managed client resources")
	fmt.Println("  luma-cli research run --role <description>")
	fmt.Println("  luma-cli script rewrite <text>   Rewrite short-video script copy")
	fmt.Println("  luma-cli subtitle <video> <srt_or_txt> [options]")
	fmt.Println("")
	fmt.Println("Agent tools:")
	fmt.Println("  luma-cli tools list")
	fmt.Println("  luma-cli tools describe asr.transcribe")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  luma-cli auth login <CARD_KEY>")
	fmt.Println("  luma-cli asr video.mp4")
	fmt.Println("  luma-cli lipsync --avatar 数字人男 --audio tts_output.wav")
	fmt.Println("  luma-cli enhance lipsync_output.mp4")
}

func Run(args []string) int {
	if len(args) < 2 {
		printUsage()
		return 1
	}

	runtimeOpts = runtimeOptions{}
	commandArgs := args[1:]
	if commandArgs[0] == "--json" {
		runtimeOpts.JSON = true
		commandArgs = commandArgs[1:]
		if len(commandArgs) == 0 {
			printUsage()
			return 1
		}
	}

	setupNotices()

	switch commandArgs[0] {
	case "version":
		fmt.Printf("luma-cli version %s\n", version)
	case "help":
		printUsage()
	default:
		if spec, ok := commandRegistry()[commandArgs[0]]; ok {
			spec.Handler(commandArgs[1:])
			return 0
		}
		fmt.Printf("unknown command: %s\n\n", commandArgs[0])
		printUsage()
		return 1
	}
	return 0
}

// setupNotices wires the skills drift check into output.PendingNotice.
// On mismatch, every JSON envelope gains a _notice.skills block;
// in text mode a hint is printed to stderr.
func setupNotices() {
	if version == "" || version == "dev" || version == "local" {
		return
	}

	stamp, err := skillsync.ReadStamp()
	if err != nil {
		return
	}

	var current string
	if stamp != nil {
		current = stamp.Version
	}

	if current == version {
		return
	}

	msg := fmt.Sprintf("skills not installed, run: luma-cli update")
	if current != "" {
		msg = fmt.Sprintf("skills %s out of sync with binary %s, run: luma-cli update", current, version)
	}

	notice := map[string]any{
		"skills": map[string]any{
			"current": current,
			"target":  version,
			"message": msg,
			"command": "luma-cli update",
		},
	}

	output.PendingNotice = func() any {
		return notice
	}

	if !runtimeOpts.JSON {
		fmt.Fprintf(os.Stderr, "Notice: %s\n", msg)
	}
}

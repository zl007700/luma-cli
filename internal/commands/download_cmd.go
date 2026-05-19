package commands

import (
	"fmt"

	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/cmdutil"
)

func cmdDownload(args []string) {
	parsed := cmdutil.Parse(args)
	if len(parsed.Positionals) < 1 {
		fmt.Println("usage: luma-cli download <url> [output_file]")
		return
	}
	url := parsed.Pos(0)
	output := "download.mp4"
	if parsed.Pos(1) != "" {
		output = parsed.Pos(1)
	}
	absOutput, err := absoluteOutputPath(output)
	if err != nil {
		fmt.Printf("Error: invalid output path: %v\n", err)
		return
	}
	if err := atom.DownloadFile(url, absOutput); err != nil {
		fmt.Printf("Error: download failed: %v\n", err)
		return
	}
	fmt.Printf("Downloaded: %s\n", absOutput)
}

package commands

import (
	"fmt"

	"github.com/luma-cli/lumer-cli/internal/atom"
	"github.com/luma-cli/lumer-cli/internal/output"
)

func cmdDownload(args []string) error {
	if len(args) < 1 {
		fmt.Println("usage: luma-cli download <url> [output_file]")
		return nil
	}
	return runDownload(args[0], args[1])
}

func runDownload(url, outputFileName string) error {
	if url == "" {
		return output.ErrValidation("url is required")
	}
	if outputFileName == "" {
		outputFileName = "download.mp4"
	}
	absOut, err := absoluteOutputPath(outputFileName)
	if err != nil {
		return output.ErrValidation("invalid output path: %v", err)
	}
	if err := atom.DownloadFile(url, absOut); err != nil {
		return output.ErrNetwork("download failed: %v", err)
	}
	fmt.Printf("Downloaded: %s\n", absOut)
	return nil
}

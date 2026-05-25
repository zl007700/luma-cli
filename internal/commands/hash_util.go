package commands

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
)

// hashSuffixFile renames filePath to include a short content hash before the extension.
// "step2_tts.wav" becomes "step2_tts_a1b2c3d4e5f6.wav". Returns the new path.
func hashSuffixFile(filePath string) (string, error) {
	h, err := cloud.FileContentHash(filePath)
	if err != nil {
		return filePath, err
	}
	ext := filepath.Ext(filePath)
	base := strings.TrimSuffix(filePath, ext)
	hashed := base + "_" + h[:12] + ext
	if err := os.Rename(filePath, hashed); err != nil {
		return filePath, err
	}
	return hashed, nil
}

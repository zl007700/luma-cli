package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/luma-cli/lumer-cli/project"
	"github.com/luma-cli/lumer-cli/subtitle"
)

// --- Project & path helpers ---

type projectDirs struct {
	audio     string
	subtitles string
	effects   string
	output    string
}

func getProjectDirs(proj *project.Project) projectDirs {
	if proj == nil {
		return projectDirs{}
	}
	return projectDirs{
		audio:     proj.SubDir(project.DirAudio),
		subtitles: proj.SubDir(project.DirSubtitles),
		effects:   proj.SubDir(project.DirEffects),
		output:    proj.SubDir(project.DirOutput),
	}
}

func resolveProjectByName(name string) *project.Project {
	if name != "" {
		p, err := project.FindByName(name)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return nil
		}
		return p
	}
	p, _ := project.GetActiveProject()
	return p
}

func resolveOutputPath(opts *subtitleOptions, proj *project.Project, dirs projectDirs) string {
	if opts.outputPath != "" {
		return resolveProjectOutput(proj, opts.outputPath)
	}
	if proj != nil {
		if opts.isTextMode {
			return filepath.Join(dirs.output, "step5_subtitle.mp4")
		}
		return filepath.Join(dirs.output, "step5_subtitle.mp4")
	}
	if opts.isTextMode {
		return "step5_subtitle.mp4"
	}
	return "step5_subtitle.mp4"
}

func resolveVideoSize(isTextMode bool, videoPath string) (int, int) {
	width, height := 1080, 1920
	if !isTextMode {
		w, h, err := subtitle.GetVideoSize(videoPath)
		if err == nil && w > 0 && h > 0 {
			width, height = w, h
		}
	}
	return width, height
}

func resolveFontSize(requestedFontSize, requestedBottomMargin, height int, fontSizeSet, bottomMarginSet bool) (int, int) {
	autoFontSize, _, _, marginV := subtitle.AutoSizeParams(0, height)
	if requestedFontSize > 0 {
		if !fontSizeSet {
			requestedFontSize = scaleSubtitleDimension(requestedFontSize, height, 1920, 32)
		}
		if requestedBottomMargin > 0 {
			if !bottomMarginSet {
				requestedBottomMargin = scaleSubtitleDimension(requestedBottomMargin, height, 1920, 90)
			}
			return requestedFontSize, requestedBottomMargin
		}
		return requestedFontSize, marginV
	}
	if requestedBottomMargin > 0 {
		if !bottomMarginSet {
			requestedBottomMargin = scaleSubtitleDimension(requestedBottomMargin, height, 1920, 90)
		}
		return autoFontSize, requestedBottomMargin
	}
	return autoFontSize, marginV
}

func resolveSideMargin(requestedSideMargin, width int, sideMarginSet bool) int {
	if requestedSideMargin <= 0 {
		requestedSideMargin = 60
	}
	if sideMarginSet {
		return requestedSideMargin
	}
	return scaleSubtitleDimension(requestedSideMargin, width, 1080, 24)
}

func scaleSubtitleDimension(value, actual, base, minValue int) int {
	if value <= 0 || actual <= 0 || base <= 0 {
		return value
	}
	scaled := int(float64(value)*float64(actual)/float64(base) + 0.5)
	if scaled < minValue {
		return minValue
	}
	return scaled
}

func resolveAssPath(outputPath string, proj *project.Project, dirs projectDirs) string {
	if proj == nil {
		return outputPath + ".ass"
	}
	assName := filepath.Base(outputPath)
	ext := filepath.Ext(assName)
	return filepath.Join(dirs.subtitles, strings.TrimSuffix(assName, ext)+".ass")
}

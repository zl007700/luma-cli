package commands

import (
	"crypto/sha1"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func describeMaterials(inputPath string) ([]materialDescriptor, error) {
	root, err := filepath.Abs(inputPath)
	if err != nil {
		return nil, err
	}
	stat, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	if stat.IsDir() {
		err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if isMaterialFile(path) {
				paths = append(paths, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else if isMaterialFile(root) {
		paths = append(paths, root)
	}
	ffprobe := bestEffortFFprobe()
	materials := make([]materialDescriptor, 0, len(paths))
	for _, path := range paths {
		abs, _ := filepath.Abs(path)
		mediaType := inferPIPMediaType(abs)
		info := pipMediaInfo{}
		if ffprobe != "" {
			if probed, err := probeMedia(ffprobe, abs); err == nil {
				info = probed
			}
		}
		resolution := ""
		if info.Width > 0 && info.Height > 0 {
			resolution = fmt.Sprintf("%dx%d", info.Width, info.Height)
		}
		title := strings.TrimSuffix(filepath.Base(abs), filepath.Ext(abs))
		materials = append(materials, materialDescriptor{
			MaterialID:  materialID(abs),
			MediaType:   mediaType,
			Path:        abs,
			Title:       title,
			Summary:     title,
			Tags:        tagsFromName(title),
			Duration:    info.Duration,
			Resolution:  resolution,
			VisualFocus: title,
		})
	}
	return materials, nil
}

func listMaterialGroups(rootPath string) ([]materialGroupView, error) {
	root, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	groups := make([]materialGroupView, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		groupPath := filepath.Join(root, entry.Name())
		group, err := describeMaterialGroupShallow(groupPath)
		if err != nil {
			continue
		}
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].ID < groups[j].ID
	})
	return groups, nil
}

func describeMaterialGroup(groupPath string) (materialGroupView, error) {
	group, err := describeMaterialGroupShallow(groupPath)
	if err != nil {
		return group, err
	}
	materialsRoot := materialGroupMaterialsRoot(group.Path)
	materials, err := describeMaterials(materialsRoot)
	if err != nil {
		return group, err
	}
	if metas, err := readMaterialMetas(filepath.Join(group.Path, "meta")); err == nil {
		materials = mergeMetaIntoMaterials(materials, metas)
		group.MetaCount = len(metas)
	}
	group.Materials = materials
	group.MaterialCount = len(materials)
	return group, nil
}

func describeMaterialGroupShallow(groupPath string) (materialGroupView, error) {
	abs, err := filepath.Abs(groupPath)
	if err != nil {
		return materialGroupView{}, err
	}
	stat, err := os.Stat(abs)
	if err != nil {
		return materialGroupView{}, err
	}
	if !stat.IsDir() {
		return materialGroupView{}, fmt.Errorf("not a directory: %s", abs)
	}
	group := materialGroupView{
		ID:   filepath.Base(abs),
		Path: abs,
	}
	manifestPath := filepath.Join(abs, "group.json")
	if manifest, err := readJSONObject(manifestPath); err == nil {
		group.Manifest = manifest
		group.Name = firstString(manifest, "name", "title", "display_name")
	}
	materialsRoot := materialGroupMaterialsRoot(abs)
	if materials, err := describeMaterials(materialsRoot); err == nil {
		group.MaterialCount = len(materials)
	}
	if metas, err := readMaterialMetas(filepath.Join(abs, "meta")); err == nil {
		group.MetaCount = len(metas)
	}
	return group, nil
}

func materialGroupMaterialsRoot(groupPath string) string {
	for _, name := range []string{"items", "materials", "source"} {
		candidate := filepath.Join(groupPath, name)
		if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
			return candidate
		}
	}
	return groupPath
}

func bestEffortFFprobe() string {
	if ffmpeg, err := installedFFmpegPath(); err == nil {
		if ffprobe, err := installedFFprobePath(ffmpeg); err == nil {
			return ffprobe
		}
	}
	if ffprobe, err := exec.LookPath("ffprobe"); err == nil {
		return ffprobe
	}
	return ""
}

func isMaterialFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp4", ".mov", ".m4v", ".avi", ".mkv", ".webm", ".jpg", ".jpeg", ".png", ".webp", ".bmp":
		return true
	default:
		return false
	}
}

func inferPIPMediaType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".webp", ".bmp":
		return "image"
	default:
		return "video"
	}
}

func materialID(path string) string {
	sum := sha1.Sum([]byte(filepath.Clean(path)))
	return "mat_" + fmt.Sprintf("%x", sum[:8])
}

func tagsFromName(name string) []string {
	parts := strings.FieldsFunc(name, func(r rune) bool {
		return r == '_' || r == '-' || r == ' ' || r == '.' || r == ',' || r == '，'
	})
	tags := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			tags = append(tags, part)
		}
	}
	return tags
}

package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	lconfig "github.com/luma-cli/lumer-cli/internal/config"
)

func defaultMaterialLibraryRoot() string {
	dir, err := lconfig.Dir()
	if err != nil {
		return filepath.Join(".", ".luma", "material-library")
	}
	return filepath.Join(dir, "material-library")
}

func defaultMaterialGroupsRoot() string {
	return filepath.Join(defaultMaterialLibraryRoot(), "groups")
}

func resolveMaterialGroupPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return value
	}
	if filepath.IsAbs(value) || strings.ContainsAny(value, `/\`) {
		return value
	}
	candidate := filepath.Join(defaultMaterialGroupsRoot(), value)
	if stat, err := os.Stat(candidate); err == nil && stat.IsDir() {
		return candidate
	}
	return value
}

func importMaterialGroup(sourcePath, name string, replace bool) (string, error) {
	sourceAbs, err := filepath.Abs(sourcePath)
	if err != nil {
		return "", err
	}
	stat, err := os.Stat(sourceAbs)
	if err != nil {
		return "", err
	}
	if !stat.IsDir() {
		return "", fmt.Errorf("source is not a directory: %s", sourceAbs)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = filepath.Base(sourceAbs)
	}
	if err := validateMaterialGroupName(name); err != nil {
		return "", err
	}
	root := defaultMaterialGroupsRoot()
	target := filepath.Join(root, name)
	if err := ensureChildPath(root, target); err != nil {
		return "", err
	}
	if _, err := os.Stat(target); err == nil {
		if !replace {
			return "", fmt.Errorf("target already exists: %s; pass --replace to overwrite", target)
		}
		if err := os.RemoveAll(target); err != nil {
			return "", fmt.Errorf("replace existing group: %w", err)
		}
	} else if !os.IsNotExist(err) {
		return "", err
	}
	if err := copyDir(sourceAbs, target); err != nil {
		return "", err
	}
	return target, nil
}

func validateMaterialGroupName(name string) error {
	if name == "" {
		return fmt.Errorf("group name cannot be empty")
	}
	if strings.ContainsAny(name, `/\`) || name == "." || name == ".." || strings.Contains(name, "..") {
		return fmt.Errorf("invalid group name: %s", name)
	}
	return nil
}

func ensureChildPath(root, target string) error {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	if targetAbs != rootAbs && !strings.HasPrefix(targetAbs, rootAbs+string(os.PathSeparator)) {
		return fmt.Errorf("target escapes material library: %s", target)
	}
	return nil
}

func copyDir(source, target string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(target, rel)
		if entry.IsDir() {
			return os.MkdirAll(dst, 0755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		return copyMaterialFile(path, dst, info.Mode())
	})
}

func copyMaterialFile(source, target string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

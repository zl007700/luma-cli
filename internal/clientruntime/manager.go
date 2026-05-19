package clientruntime

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/luma-cli/lumer-cli/cloud"
	"github.com/luma-cli/lumer-cli/internal/config"
)

type InstalledRuntime struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	Platform       string `json:"platform"`
	ExecutablePath string `json:"executable_path"`
	FFProbePath    string `json:"ffprobe_path,omitempty"`
	SHA256         string `json:"sha256"`
}

type CachedResource struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
}

func RuntimeRoot() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "runtime"), nil
}

func ResourceRoot() (string, error) {
	dir, err := config.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "client-resources"), nil
}

func CurrentRuntime(name string) (*InstalledRuntime, error) {
	root, err := RuntimeRoot()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(root, name, "current.json"))
	if err != nil {
		return nil, err
	}
	var installed InstalledRuntime
	if err := json.Unmarshal(data, &installed); err != nil {
		return nil, err
	}
	return &installed, nil
}

func CurrentResource(resourceID string) (*CachedResource, error) {
	root, err := ResourceRoot()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(root, resourceID, "current.json"))
	if err != nil {
		return nil, err
	}
	var cached CachedResource
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}
	return &cached, nil
}

func InstallRuntime(cardKey, name, version string) (*InstalledRuntime, error) {
	if version == "" {
		version = "auto"
	}
	resolved, err := cloud.ResolveRuntime(name, runtime.GOOS, runtime.GOARCH, version, cardKey)
	if err != nil {
		return nil, err
	}
	if resolved.DownloadURL == "" {
		return nil, fmt.Errorf("runtime response did not include download_url")
	}

	root, err := RuntimeRoot()
	if err != nil {
		return nil, err
	}
	platform := resolved.Platform
	if platform == "" {
		platform = runtime.GOOS + "-" + runtime.GOARCH
	}
	installDir := filepath.Join(root, resolved.Name, resolved.Version, platform)
	if err := os.RemoveAll(installDir); err != nil {
		return nil, fmt.Errorf("clean install dir: %w", err)
	}
	if err := os.MkdirAll(installDir, 0755); err != nil {
		return nil, fmt.Errorf("create install dir: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "luma-runtime-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	filename := resolved.Filename
	if filename == "" {
		filename = resolved.Name + ".archive"
	}
	archivePath := filepath.Join(tmpDir, filename)
	if err := download(resolved.DownloadURL, archivePath); err != nil {
		return nil, err
	}
	if resolved.SHA256 != "" {
		if err := verifySHA256(archivePath, resolved.SHA256); err != nil {
			return nil, err
		}
	}

	if err := extractArchive(archivePath, installDir, resolved.ArchiveType); err != nil {
		return nil, err
	}
	execPath := resolved.ExecutablePath
	if execPath != "" {
		execPath = filepath.Join(installDir, filepath.FromSlash(execPath))
	} else {
		execPath, err = findExecutable(installDir, resolved.Name)
		if err != nil {
			return nil, err
		}
	}
	if err := os.Chmod(execPath, 0755); err != nil && runtime.GOOS != "windows" {
		return nil, err
	}
	ffprobePath := ""
	if resolved.FFProbePath != "" {
		ffprobePath = filepath.Join(installDir, filepath.FromSlash(resolved.FFProbePath))
		if err := os.Chmod(ffprobePath, 0755); err != nil && runtime.GOOS != "windows" {
			return nil, err
		}
	}

	installed := &InstalledRuntime{
		Name:           resolved.Name,
		Version:        resolved.Version,
		Platform:       platform,
		ExecutablePath: execPath,
		FFProbePath:    ffprobePath,
		SHA256:         resolved.SHA256,
	}
	currentPath := filepath.Join(root, resolved.Name, "current.json")
	if err := writeJSON(currentPath, installed); err != nil {
		return nil, err
	}
	return installed, nil
}

func CacheResource(cardKey, resourceID string) (*CachedResource, error) {
	signed, err := cloud.SignClientResource(resourceID, cardKey)
	if err != nil {
		return nil, err
	}
	if signed.DownloadURL == "" {
		return nil, fmt.Errorf("resource response did not include download_url")
	}
	root, err := ResourceRoot()
	if err != nil {
		return nil, err
	}
	version := signed.Version
	if version == "" {
		version = "default"
	}
	filename := signed.Filename
	if filename == "" {
		filename = resourceID
	}
	targetDir := filepath.Join(root, signed.ID, version)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return nil, err
	}
	targetPath := filepath.Join(targetDir, filepath.Base(filename))
	if err := download(signed.DownloadURL, targetPath); err != nil {
		return nil, err
	}
	if signed.SHA256 != "" {
		if err := verifySHA256(targetPath, signed.SHA256); err != nil {
			return nil, err
		}
	}
	cached := &CachedResource{
		ID:      signed.ID,
		Type:    signed.Type,
		Name:    signed.Name,
		Version: version,
		Path:    targetPath,
		SHA256:  signed.SHA256,
	}
	if err := writeJSON(filepath.Join(root, signed.ID, "current.json"), cached); err != nil {
		return nil, err
	}
	return cached, nil
}

func download(url, outputPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}
	out, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func verifySHA256(path, expected string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}
	got := hex.EncodeToString(hash.Sum(nil))
	if !strings.EqualFold(got, strings.TrimSpace(expected)) {
		return fmt.Errorf("sha256 mismatch: got %s, expected %s", got, expected)
	}
	return nil
}

func extractArchive(archivePath, destDir, archiveType string) error {
	archiveType = strings.ToLower(strings.TrimSpace(archiveType))
	if archiveType == "" {
		name := strings.ToLower(archivePath)
		switch {
		case strings.HasSuffix(name, ".zip"):
			archiveType = "zip"
		case strings.HasSuffix(name, ".tar.gz"), strings.HasSuffix(name, ".tgz"):
			archiveType = "tar.gz"
		default:
			return fmt.Errorf("unknown archive type for %s", archivePath)
		}
	}
	switch archiveType {
	case "zip":
		return extractZip(archivePath, destDir)
	case "tar.gz", "tgz":
		return extractTarGz(archivePath, destDir)
	default:
		return fmt.Errorf("unsupported archive type: %s", archiveType)
	}
}

func extractZip(archivePath, destDir string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		target, err := safeJoin(destDir, file.Name)
		if err != nil {
			return err
		}
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		in, err := file.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.Mode())
		if err != nil {
			in.Close()
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		in.Close()
		if copyErr != nil {
			return copyErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(destDir, header.Name)
		if err != nil {
			return err
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(out, tr)
			closeErr := out.Close()
			if copyErr != nil {
				return copyErr
			}
			if closeErr != nil {
				return closeErr
			}
		}
	}
}

func safeJoin(root, name string) (string, error) {
	target := filepath.Join(root, filepath.Clean(filepath.FromSlash(name)))
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return "", err
	}
	if targetAbs != rootAbs && !strings.HasPrefix(targetAbs, rootAbs+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry escapes target directory: %s", name)
	}
	return targetAbs, nil
}

func findExecutable(root, name string) (string, error) {
	targets := []string{name}
	if runtime.GOOS == "windows" {
		targets = append(targets, name+".exe")
	} else {
		targets = append(targets, name)
	}
	var found string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || found != "" {
			return err
		}
		base := strings.ToLower(filepath.Base(path))
		for _, target := range targets {
			if base == strings.ToLower(target) {
				found = path
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("could not find executable %s under extracted runtime", name)
	}
	return found, nil
}

func writeJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

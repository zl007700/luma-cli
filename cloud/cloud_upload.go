package cloud

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// FileContentHash returns the SHA-256 hex digest of a file.
func FileContentHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func guessMimeType(filePath string) string {
	switch ext := filepath.Ext(filePath); ext {
	case ".mp3":
		return "audio/mpeg"
	case ".mp4":
		return "video/mp4"
	case ".wav":
		return "audio/wav"
	case ".avi":
		return "video/x-msvideo"
	case ".mov":
		return "video/quicktime"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	default:
		return "application/octet-stream"
	}
}

func guessResourceType(mimeType string) string {
	if mimeType == "video/mp4" || mimeType == "video/quicktime" || mimeType == "video/x-msvideo" {
		return "video"
	}
	if mimeType == "audio/mpeg" || mimeType == "audio/wav" || mimeType == "audio/mp3" {
		return "audio"
	}
	if mimeType == "image/jpeg" || mimeType == "image/png" {
		return "image"
	}
	return "other"
}

// UploadFile uploads a local file to the cloud and returns the object key.
func UploadFile(filePath, cardKey, groupName string) (string, error) {
	return UploadFileWithName(filePath, cardKey, groupName, "")
}

// UploadFileWithName uploads a local file with an optional resource filename.
func UploadFileWithName(filePath, cardKey, groupName, resourceName string) (string, error) {
	stat, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}
	fileSize := stat.Size()
	mimeType := guessMimeType(filePath)
	resourceType := guessResourceType(mimeType)

	filename := filepath.Base(filePath)
	if strings.TrimSpace(resourceName) != "" {
		filename = strings.TrimSpace(resourceName)
		if filepath.Ext(filename) == "" {
			filename += filepath.Ext(filePath)
		}
	}

	// Append content hash to the filename so re-uploading a different file
	// with the same basename never hits a stale server-side cache.
	if h, err := FileContentHash(filePath); err == nil {
		ext := filepath.Ext(filename)
		base := strings.TrimSuffix(filename, ext)
		filename = base + "_" + h[:12] + ext
	}

	if fileSize >= multipartThreshold {
		return uploadMultipart(filePath, filename, groupName, resourceType, mimeType, cardKey)
	}

	asciiFilename := filename
	for _, c := range filename {
		if c >= 128 {
			asciiFilename = fmt.Sprintf("upload_%d%s", time.Now().UnixNano()%100000, filepath.Ext(filePath))
			break
		}
	}

	return uploadSinglePart(filePath, asciiFilename, groupName, resourceType, mimeType, cardKey)
}

func uploadSinglePart(filePath, filename, groupName, resourceType, mimeType, cardKey string) (string, error) {
	signBody, _ := json.Marshal(map[string]any{
		"group_name":    groupName,
		"filename":      filename,
		"resource_type": resourceType,
		"mime_type":     mimeType,
		"variant":       "original",
	})
	signResult, err := apiRequest("POST", "/v1/resources/upload-sign", signBody, cardKey)
	if err != nil {
		return "", fmt.Errorf("upload sign failed: %w", err)
	}
	uploadURL, _ := signResult["upload_url"].(string)
	objectKey, _ := signResult["object_key"].(string)
	if uploadURL == "" || objectKey == "" {
		return "", fmt.Errorf("invalid upload sign response")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	req, err := http.NewRequest("PUT", uploadURL, file)
	if err != nil {
		return "", fmt.Errorf("create upload request failed: %w", err)
	}
	req.Header.Set("Content-Type", mimeType)
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("file upload failed: %w", err)
	}
	httpResp.Body.Close()
	if httpResp.StatusCode >= 400 {
		return "", fmt.Errorf("upload returned status %d", httpResp.StatusCode)
	}

	saveBody, _ := json.Marshal(map[string]any{
		"resource_id":   signResult["resource_id"],
		"group_name":    groupName,
		"resource_type": resourceType,
		"mime_type":     mimeType,
		"object_key":    objectKey,
		"filename":      filename,
	})
	_, err = apiRequest("POST", "/v1/resources", saveBody, cardKey)
	if err != nil {
		return "", fmt.Errorf("save resource record failed: %w", err)
	}

	return objectKey, nil
}

func uploadMultipart(filePath, filename, groupName, resourceType, mimeType, cardKey string) (string, error) {
	asciiFilename := fmt.Sprintf("upload_%d%s", time.Now().UnixNano()%100000, filepath.Ext(filePath))

	initBody, _ := json.Marshal(map[string]any{
		"group_name":    groupName,
		"filename":      asciiFilename,
		"resource_type": resourceType,
		"mime_type":     mimeType,
		"variant":       "original",
	})
	initResult, err := apiRequest("POST", "/v1/resources/multipart/init", initBody, cardKey)
	if err != nil {
		return "", fmt.Errorf("multipart init failed: %w", err)
	}

	resourceID, _ := initResult["resource_id"].(string)
	uploadID, _ := initResult["upload_id"].(string)
	objectKey, _ := initResult["object_key"].(string)
	if resourceID == "" || uploadID == "" || objectKey == "" {
		return "", fmt.Errorf("invalid multipart init response")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	const partSize = 5 * 1024 * 1024 // 5MB parts
	var parts []map[string]any
	partNumber := 1
	offset := int64(0)

	for {
		stat, _ := file.Stat()
		remaining := stat.Size() - offset
		if remaining <= 0 {
			break
		}
		currentSize := int64(partSize)
		if remaining < currentSize {
			currentSize = remaining
		}

		signBody, _ := json.Marshal(map[string]any{
			"object_key":  objectKey,
			"upload_id":   uploadID,
			"part_number": partNumber,
		})
		signResult, err := apiRequest("POST", "/v1/resources/multipart/sign-part", signBody, cardKey)
		if err != nil {
			return "", fmt.Errorf("multipart sign part failed: %w", err)
		}
		partURL, _ := signResult["upload_url"].(string)
		if partURL == "" {
			return "", fmt.Errorf("invalid part sign response")
		}

		chunk := make([]byte, currentSize)
		if _, err := file.ReadAt(chunk, offset); err != nil {
			return "", fmt.Errorf("read chunk failed: %w", err)
		}

		req, err := http.NewRequest("PUT", partURL, bytes.NewReader(chunk))
		if err != nil {
			return "", fmt.Errorf("create part request failed: %w", err)
		}
		req.Header.Set("Content-Type", mimeType)
		httpResp, err := http.DefaultClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("multipart part upload failed: %w", err)
		}
		httpResp.Body.Close()
		if httpResp.StatusCode >= 400 {
			return "", fmt.Errorf("part upload returned status %d", httpResp.StatusCode)
		}

		etag := httpResp.Header.Get("ETag")
		if etag == "" {
			etag = httpResp.Header.Get("Etag")
		}
		parts = append(parts, map[string]any{
			"part_number": partNumber,
			"etag":        etag,
		})

		offset += currentSize
		partNumber++
	}

	completeBody, _ := json.Marshal(map[string]any{
		"resource_id": resourceID,
		"group_name":  groupName,
		"object_key":  objectKey,
		"upload_id":   uploadID,
		"parts":       parts,
	})
	_, err = apiRequest("POST", "/v1/resources/multipart/complete", completeBody, cardKey)
	if err != nil {
		return "", fmt.Errorf("multipart complete failed: %w", err)
	}

	return objectKey, nil
}

package atom

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/luma-cli/lumer-cli/cloud"
)

// ASROptions describes an ASR atomic capability invocation.
type ASROptions struct {
	FilePath   string
	Language   string
	CardKey    string
	TimeoutSec int
}

// ASRResult is the structured ASR atomic capability result.
type ASRResult struct {
	TaskID    string
	ObjectKey string
	Text      string
	Segments  []any
	Status    map[string]any
}

// RunASR uploads media, submits ASR, waits for completion, and extracts text.
func RunASR(opts ASROptions) (*ASRResult, error) {
	if opts.Language == "" {
		opts.Language = "zh"
	}
	if opts.TimeoutSec <= 0 {
		opts.TimeoutSec = 300
	}

	objectKey, err := cloud.UploadFile(opts.FilePath, opts.CardKey, "asr_input")
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	taskResult, err := cloud.SubmitTask("asr", filepath.Base(opts.FilePath), map[string]any{
		"audio_object_key": objectKey,
		"language":         opts.Language,
	}, opts.CardKey)
	if err != nil {
		return nil, fmt.Errorf("submit failed: %w", err)
	}

	taskID, _ := taskResult["task_id"].(string)
	if taskID == "" {
		return nil, fmt.Errorf("no task_id returned")
	}

	status, stillRunning := cloud.WaitTaskComplete(taskID, opts.CardKey, opts.TimeoutSec)
	if stillRunning {
		return nil, fmt.Errorf("ASR task timed out")
	}
	if msg := TaskFailure(status); msg != "" {
		return nil, fmt.Errorf("ASR task failed: %s", msg)
	}

	var text string
	var segments []any
	if output, ok := status["output"].(map[string]any); ok {
		if result, ok := output["result"].(map[string]any); ok {
			text, _ = result["text"].(string)
			segments, _ = result["segments"].([]any)
		}
	}
	if text == "" {
		return nil, fmt.Errorf("ASR returned no text")
	}

	return &ASRResult{
		TaskID:    taskID,
		ObjectKey: objectKey,
		Text:      text,
		Segments:  segments,
		Status:    status,
	}, nil
}

// TTSOptions describes a TTS atomic capability invocation.
type TTSOptions struct {
	Text       string
	VoiceKey   string
	SpeechRate float64
	CardKey    string
	OutputPath string
	TimeoutSec int
}

// TTSResult is the structured TTS atomic capability result.
type TTSResult struct {
	TaskID          string
	ResultObjectKey string
	AudioObjectKey  string
	OutputURL       string
	OutputPath      string
	Status          map[string]any
}

// RunTTS submits TTS, waits for completion, and downloads the result when a URL is returned.
func RunTTS(opts TTSOptions) (*TTSResult, error) {
	if opts.SpeechRate == 0 {
		opts.SpeechRate = 1.1
	}
	if opts.TimeoutSec <= 0 {
		opts.TimeoutSec = 300
	}

	taskResult, err := cloud.SubmitTask("tts", "tts_output", map[string]any{
		"text":             opts.Text,
		"voice_object_key": opts.VoiceKey,
		"speech_rate":      opts.SpeechRate,
	}, opts.CardKey)
	if err != nil {
		return nil, fmt.Errorf("submit failed: %w", err)
	}

	taskID, _ := taskResult["task_id"].(string)
	if taskID == "" {
		return nil, fmt.Errorf("no task_id returned")
	}

	status, stillRunning := cloud.WaitTaskComplete(taskID, opts.CardKey, opts.TimeoutSec)
	if stillRunning {
		return nil, fmt.Errorf("TTS task timed out")
	}
	if msg := TaskFailure(status); msg != "" {
		return nil, fmt.Errorf("TTS task failed: %s", msg)
	}

	resultObjKey := ResultObjectKey(status)
	outputURL := ResultURL(status)
	if outputURL == "" && resultObjKey == "" {
		return nil, fmt.Errorf("TTS returned no output")
	}

	if outputURL != "" && opts.OutputPath != "" {
		if err := DownloadFile(outputURL, opts.OutputPath); err != nil {
			return nil, fmt.Errorf("download failed: %w", err)
		}
	}

	return &TTSResult{
		TaskID:          taskID,
		ResultObjectKey: resultObjKey,
		AudioObjectKey:  StripResourcePrefix(resultObjKey, opts.CardKey),
		OutputURL:       outputURL,
		OutputPath:      opts.OutputPath,
		Status:          status,
	}, nil
}

// EnhanceOptions describes a video enhancement atomic capability invocation.
type EnhanceOptions struct {
	VideoPath  string
	Scale      int
	CardKey    string
	OutputPath string
	TimeoutSec int
}

// EnhanceResult is the structured video enhancement result.
type EnhanceResult struct {
	TaskID     string
	ObjectKey  string
	OutputURL  string
	OutputPath string
	Status     map[string]any
}

// RunEnhance uploads a video, submits enhancement, waits for completion, and downloads output.
func RunEnhance(opts EnhanceOptions) (*EnhanceResult, error) {
	if opts.Scale <= 0 {
		opts.Scale = 2
	}
	if opts.TimeoutSec <= 0 {
		opts.TimeoutSec = 600
	}

	objectKey, err := cloud.UploadFile(opts.VideoPath, opts.CardKey, "enhance_input")
	if err != nil {
		return nil, fmt.Errorf("upload failed: %w", err)
	}

	taskResult, err := cloud.SubmitTask("enhance", filepath.Base(opts.VideoPath), map[string]any{
		"video_object_key": objectKey,
		"scale":            opts.Scale,
	}, opts.CardKey)
	if err != nil {
		return nil, fmt.Errorf("submit failed: %w", err)
	}

	taskID, _ := taskResult["task_id"].(string)
	if taskID == "" {
		return nil, fmt.Errorf("no task_id returned")
	}

	status, stillRunning := cloud.WaitTaskComplete(taskID, opts.CardKey, opts.TimeoutSec)
	if stillRunning {
		return nil, fmt.Errorf("Enhance task timed out")
	}
	if msg := TaskFailure(status); msg != "" {
		return nil, fmt.Errorf("Enhance task failed: %s", msg)
	}

	outputURL := ResultURL(status)
	if outputURL == "" {
		data, _ := json.Marshal(status)
		return nil, fmt.Errorf("Enhance returned no output: %s", string(data))
	}

	if opts.OutputPath != "" {
		if err := DownloadFile(outputURL, opts.OutputPath); err != nil {
			return nil, fmt.Errorf("download failed: %w", err)
		}
	}

	return &EnhanceResult{
		TaskID:     taskID,
		ObjectKey:  objectKey,
		OutputURL:  outputURL,
		OutputPath: opts.OutputPath,
		Status:     status,
	}, nil
}

// LipSyncOptions describes a lip-sync atomic capability invocation.
type LipSyncOptions struct {
	VideoKey   string
	AudioKey   string
	CardKey    string
	OutputPath string
	TimeoutSec int
}

// LipSyncResult is the structured lip-sync result.
type LipSyncResult struct {
	TaskID     string
	OutputURL  string
	OutputPath string
	Status     map[string]any
}

// RunLipSync submits lip-sync, waits for completion, and downloads output.
func RunLipSync(opts LipSyncOptions) (*LipSyncResult, error) {
	if opts.TimeoutSec <= 0 {
		opts.TimeoutSec = 600
	}

	taskResult, err := cloud.SubmitTask("lipsync", "lipsync_output", map[string]any{
		"video_object_key":    opts.VideoKey,
		"audio_object_key":    opts.AudioKey,
		"guidance_scale":      1.0,
		"num_inference_steps": 15,
		"enable_superres":     true,
		"superres_scale":      2,
		"start_mode":          "from_start",
		"multi_shot":          map[string]any{"resource_list": []any{}},
	}, opts.CardKey)
	if err != nil {
		return nil, fmt.Errorf("submit failed: %w", err)
	}

	taskID, _ := taskResult["task_id"].(string)
	if taskID == "" {
		return nil, fmt.Errorf("no task_id returned")
	}

	status, stillRunning := cloud.WaitTaskComplete(taskID, opts.CardKey, opts.TimeoutSec)
	if stillRunning {
		return nil, fmt.Errorf("LipSync task timed out")
	}
	if msg := TaskFailure(status); msg != "" {
		return nil, fmt.Errorf("LipSync task failed: %s", msg)
	}

	outputURL := ResultURL(status)
	if outputURL == "" {
		return nil, fmt.Errorf("LipSync returned no output")
	}

	if opts.OutputPath != "" {
		if err := DownloadFile(outputURL, opts.OutputPath); err != nil {
			return nil, fmt.Errorf("download failed: %w", err)
		}
	}

	return &LipSyncResult{
		TaskID:     taskID,
		OutputURL:  outputURL,
		OutputPath: opts.OutputPath,
		Status:     status,
	}, nil
}

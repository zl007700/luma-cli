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

// AlignmentOptions describes an audio/text alignment task.
type AlignmentOptions struct {
	AudioPath  string
	TextList   []string
	Language   string
	CardKey    string
	TimeoutSec int
}

// AlignmentSegment is one cloud-aligned subtitle segment.
type AlignmentSegment struct {
	SegID int
	Start float64
	End   float64
	Text  string
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

// RunAlignment uploads audio, submits alignment, waits for completion, and extracts segment timing.
func RunAlignment(opts AlignmentOptions) ([]AlignmentSegment, error) {
	if opts.Language == "" {
		opts.Language = "zh"
	}
	if opts.TimeoutSec <= 0 {
		opts.TimeoutSec = 300
	}
	if len(opts.TextList) == 0 {
		return nil, fmt.Errorf("alignment text list is empty")
	}

	objectKey, err := cloud.UploadFile(opts.AudioPath, opts.CardKey, "alignment_input")
	if err != nil {
		return nil, fmt.Errorf("upload alignment audio failed: %w", err)
	}

	taskResult, err := cloud.SubmitTask("alignment", filepath.Base(opts.AudioPath), map[string]any{
		"audio_object_key": objectKey,
		"language":         opts.Language,
		"text_list":        opts.TextList,
	}, opts.CardKey)
	if err != nil {
		return nil, fmt.Errorf("submit alignment failed: %w", err)
	}

	taskID, _ := taskResult["task_id"].(string)
	if taskID == "" {
		return nil, fmt.Errorf("no alignment task_id returned")
	}

	status, stillRunning := cloud.WaitTaskComplete(taskID, opts.CardKey, opts.TimeoutSec)
	if stillRunning {
		return nil, fmt.Errorf("alignment task timed out")
	}
	if msg := TaskFailure(status); msg != "" {
		return nil, fmt.Errorf("alignment task failed: %s", msg)
	}

	result := alignmentResultPayload(status)
	if result == nil {
		return nil, fmt.Errorf("alignment returned no result")
	}
	rawSegments := listFromAny(result["segments"])
	if len(rawSegments) == 0 {
		rawSegments = listFromAny(result["aligned_segments"])
	}
	if len(rawSegments) == 0 {
		return nil, fmt.Errorf("alignment returned no segments")
	}

	segments := make([]AlignmentSegment, 0, len(rawSegments))
	for index, raw := range rawSegments {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		start := floatFromAny(firstPresent(item, "start", "start_time"))
		end := floatFromAny(firstPresent(item, "end", "end_time"))
		if end <= start {
			continue
		}
		segments = append(segments, AlignmentSegment{
			SegID: intFromAny(firstPresent(item, "seg_id", "id", "index"), index),
			Start: start,
			End:   end,
			Text:  stringFromAny(item["text"]),
		})
	}
	if len(segments) == 0 {
		return nil, fmt.Errorf("alignment returned no valid timed segments")
	}
	return segments, nil
}

func alignmentResultPayload(status map[string]any) map[string]any {
	if output, ok := status["output"].(map[string]any); ok {
		if result, ok := output["result"].(map[string]any); ok {
			return result
		}
	}
	if result, ok := status["result"].(map[string]any); ok {
		return result
	}
	return nil
}

func listFromAny(value any) []any {
	if items, ok := value.([]any); ok {
		return items
	}
	return nil
}

func firstPresent(item map[string]any, keys ...string) any {
	for _, key := range keys {
		if value, ok := item[key]; ok {
			return value
		}
	}
	return nil
}

func floatFromAny(value any) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		out, _ := v.Float64()
		return out
	default:
		var out float64
		fmt.Sscanf(fmt.Sprint(value), "%f", &out)
		return out
	}
}

func intFromAny(value any, fallback int) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		out, _ := v.Int64()
		return int(out)
	default:
		var out int
		if _, err := fmt.Sscanf(fmt.Sprint(value), "%d", &out); err == nil {
			return out
		}
		return fallback
	}
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

// TTSOptions describes a TTS atomic capability invocation.
type TTSOptions struct {
	Text            string
	VoiceKey        string
	SpeechRate      float64
	TrimLongSilence bool
	CardKey         string
	OutputPath      string
	TimeoutSec      int
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
		"text":              opts.Text,
		"voice_object_key":  opts.VoiceKey,
		"speech_rate":       opts.SpeechRate,
		"trim_long_silence": opts.TrimLongSilence,
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
	VideoKey          string
	AudioKey          string
	GuidanceScale     float64
	NumInferenceSteps int
	EnableSuperres    bool
	DisableSuperres   bool
	SuperresScale     int
	RandomStart       bool
	MultiShot         map[string]any
	CardKey           string
	OutputPath        string
	TimeoutSec        int
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
	if opts.GuidanceScale == 0 {
		opts.GuidanceScale = 1.0
	}
	if opts.NumInferenceSteps <= 0 {
		opts.NumInferenceSteps = 15
	}
	if opts.SuperresScale <= 0 {
		opts.SuperresScale = 2
	}
	enableSuperres := true
	if opts.DisableSuperres {
		enableSuperres = false
	} else if opts.EnableSuperres {
		enableSuperres = true
	}
	multiShot := opts.MultiShot
	if multiShot == nil {
		multiShot = map[string]any{"resource_list": []any{}}
	}

	taskResult, err := cloud.SubmitTask("lipsync", "lipsync_output", map[string]any{
		"video_object_key":    opts.VideoKey,
		"audio_object_key":    opts.AudioKey,
		"guidance_scale":      opts.GuidanceScale,
		"num_inference_steps": opts.NumInferenceSteps,
		"enable_superres":     enableSuperres,
		"superres_scale":      opts.SuperresScale,
		"random_start":        opts.RandomStart,
		"start_mode":          "from_start",
		"multi_shot":          multiShot,
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

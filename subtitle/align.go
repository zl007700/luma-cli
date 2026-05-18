package subtitle

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/luma-cli/lumer-cli/cloud"
)

// ExtractAudio extracts audio from video to a WAV file (16kHz mono).
func ExtractAudio(videoPath, outputPath, ffmpegPath string) error {
	if ffmpegPath == "" {
		ffmpegPath = findFFmpeg()
	}
	cmd := exec.Command(ffmpegPath,
		"-y", "-i", videoPath,
		"-vn", "-acodec", "pcm_s16le",
		"-ar", "16000", "-ac", "1",
		outputPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// AlignSegments aligns text segments to audio timestamps using cloud ASR API.
func AlignSegments(audioPath string, segments []Segment, apiKey string) ([]Segment, error) {
	// Build text list from segments
	var texts []string
	for _, seg := range segments {
		texts = append(texts, seg.Text)
	}

	// Call cloud ASR alignment API
	inputPayload := map[string]any{
		"audio_object_key": "", // empty means upload local file
		"local_audio_path": audioPath,
		"segments":         texts,
		"language":         "zh",
	}

	body, _ := json.Marshal(inputPayload)
	req, err := newHTTPRequest("POST", cloud.BaseURL()+"/v1/asr/align", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-User-Id", apiKey)

	resp, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Segments []struct {
			SegID int     `json:"seg_id"`
			Start float64 `json:"start"`
			End   float64 `json:"end"`
			Text  string  `json:"text"`
		} `json:"segments"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parse alignment response failed: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("alignment error: %s", result.Error.Message)
	}

	// Build result map
	alignMap := make(map[int]struct{ start, end float64 })
	for _, s := range result.Segments {
		alignMap[s.SegID] = struct{ start, end float64 }{s.Start, s.End}
	}

	// Apply to segments
	for i := range segments {
		if timing, ok := alignMap[segments[i].SegID]; ok {
			segments[i].Start = timing.start
			segments[i].End = timing.end
		}
	}

	return segments, nil
}

// GetVideoDuration returns duration of video in seconds.
func GetVideoDuration(videoPath, ffprobePath string) (float64, error) {
	if ffprobePath == "" {
		ffprobePath = resolveFFprobe()
	}
	if ffprobePath == "" {
		ffprobePath = findFFmpeg()
	}

	cmd := exec.Command(ffprobePath,
		"-v", "error",
		"-show_entries", "format=duration",
		"-of", "json",
		videoPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var data struct {
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if err := json.Unmarshal(output, &data); err != nil {
		return 0, err
	}
	var dur float64
	fmt.Sscanf(data.Format.Duration, "%f", &dur)
	return dur, nil
}

// FallbackEvenAlign distributes segments evenly across total duration.
func FallbackEvenAlign(segments []Segment, totalDuration float64) []Segment {
	if totalDuration <= 0 {
		return segments
	}
	var cursor float64
	totalChars := 0
	for _, seg := range segments {
		totalChars += len([]rune(seg.Text))
	}
	for i := range segments {
		segLen := len([]rune(segments[i].Text))
		ratio := float64(segLen) / float64(max(totalChars, 1))

		nextCursor := cursor + totalDuration*ratio
		segments[i].Start = cursor
		segments[i].End = nextCursor
		cursor = nextCursor
	}
	return segments
}

// resolveFFprobe finds ffprobe path, with ffmpeg-sibling fallback.
func resolveFFprobe() string {
	if p, err := exec.LookPath("ffprobe"); err == nil {
		return p
	}
	ffmpegPath := findFFmpeg()
	if ffmpegPath == "" {
		return ""
	}
	probePath := filepath.Join(filepath.Dir(ffmpegPath), "ffprobe")
	if _, err := os.Stat(probePath); err == nil {
		return probePath
	}
	// Windows fallback
	probePathExe := probePath + ".exe"
	if _, err := os.Stat(probePathExe); err == nil {
		return probePathExe
	}
	return ""
}

// GetVideoSize returns width and height of video.
func GetVideoSize(videoPath string) (int, int, error) {
	ffprobePath := resolveFFprobe()
	if ffprobePath == "" {
		ffprobePath = findFFmpeg()
	}

	cmd := exec.Command(ffprobePath,
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "json",
		videoPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	jsonStr := string(output)
	wRe := regexp.MustCompile(`"width"\s*:\s*(\d+)`)
	hRe := regexp.MustCompile(`"height"\s*:\s*(\d+)`)
	wm := wRe.FindStringSubmatch(jsonStr)
	hm := hRe.FindStringSubmatch(jsonStr)

	var w, h int
	if len(wm) >= 2 {
		fmt.Sscanf(wm[1], "%d", &w)
	}
	if len(hm) >= 2 {
		fmt.Sscanf(hm[1], "%d", &h)
	}
	return w, h, nil
}

// AutoSizeParams calculates font size, margins, etc. based on video dimensions.
func AutoSizeParams(width, height int) (fontSize, marginL, marginR, marginV int) {
	if height == 0 {
		height = 1920
	}
	if width == 0 {
		width = 1080
	}
	fontSize = int(float64(height) * 0.037)
	if fontSize < 32 {
		fontSize = 32
	}
	marginL = 60
	marginR = 60
	marginV = int(float64(height) * 0.155)
	if marginV < 90 {
		marginV = 90
	}
	return fontSize, marginL, marginR, marginV
}

func findFFmpeg() string {
	if p, err := exec.LookPath("ffmpeg"); err == nil {
		return p
	}
	for _, loc := range []string{
		"C:\\ffmpeg\\bin\\ffmpeg.exe",
		"C:\\Program Files\\ffmpeg\\bin\\ffmpeg.exe",
	} {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}
	return ""
}


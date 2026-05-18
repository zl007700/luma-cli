package subtitle

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// EffectConfig holds rendering configuration for subtitle effects
type EffectConfig struct {
	Width       int
	Height      int
	FPS         int
	FontPath    string
	FontSize    int
	AnchorX     float64
	AnchorY     float64
	AnchorMode  string // "center" or "bottom"
	TextColor   color.RGBA
	StrokeColor color.RGBA
	StrokeWidth float64
	OutputDir   string // directory for temporary/output files (falls back to os.TempDir())
}

// Ease provides easing functions matching Z agent's Python implementation
type Ease struct{}

// BackOut implements the back-out easing function
func (e *Ease) BackOut(t float64) float64 {
	if t >= 1.0 {
		return 1.0
	}
	if t <= 0.0 {
		return 0.0
	}
	p := 1.0 - t
	return 1.0 - p*p*p - 0.1*math.Sin(p*math.Pi)
}

// CubicOut implements cubic-out easing
func (e *Ease) CubicOut(t float64) float64 {
	if t >= 1.0 {
		return 1.0
	}
	if t <= 0.0 {
		return 0.0
	}
	p := 1.0 - t
	return 1.0 - p*p*p
}

// Bounce implements bounce easing
func (e *Ease) Bounce(t float64) float64 {
	if t >= 1.0 {
		return 1.0
	}
	if t <= 0.0 {
		return 0.0
	}
	if t < 0.5 {
		return 0.5 - 0.5*math.Cos(t*math.Pi*2.0)*math.Cos(t*math.Pi*0.5)
	}
	return 1.0
}

// RenderEffect renders a subtitle segment with effect using Python PIL and returns output MOV path
// Falls back to Go placeholder if Python is unavailable
func RenderEffect(text string, effectType string, start, end float64, cfg EffectConfig) (string, error) {
	if effectType == "" || effectType == "none" || text == "" {
		return "", nil
	}

	duration := end - start
	if duration <= 0 {
		return "", nil
	}

	// Try Python rendering first
	movPath, err := renderEffectPython(text, effectType, start, end, cfg)
	if err == nil && movPath != "" {
		return movPath, nil
	}

	// Fallback to Go placeholder rendering
	return renderEffectGoFallback(text, effectType, start, end, cfg)
}

func renderEffectPython(text, effectType string, start, end float64, cfg EffectConfig) (string, error) {
	// Build JSON config for Python
	fontPath := ""
	if cfg.FontPath != "" {
		fontPath = cfg.FontPath
	}

	bottomMargin := int(float64(cfg.Height) * 0.155)
	if bottomMargin < 90 {
		bottomMargin = 90
	}
	anchorY := float64(cfg.Height - bottomMargin)

	pythonCfg := map[string]any{
		"text":         text,
		"effect_type":  effectType,
		"start":        start,
		"end":          end,
		"width":        cfg.Width,
		"height":       cfg.Height,
		"fps":          cfg.FPS,
		"font_path":    fontPath,
		"font_size":    cfg.FontSize,
		"anchor_y":     anchorY,
		"text_color":   fmt.Sprintf("#%02X%02X%02X", cfg.TextColor.R, cfg.TextColor.G, cfg.TextColor.B),
		"stroke_color": fmt.Sprintf("#%02X%02X%02X", cfg.StrokeColor.R, cfg.StrokeColor.G, cfg.StrokeColor.B),
		"stroke_width":  cfg.StrokeWidth,
	}

	cfgJSON, err := json.Marshal(pythonCfg)
	if err != nil {
		return "", fmt.Errorf("marshal config failed: %w", err)
	}

	// Try python3 first, then python
	pythonExes := []string{"python3", "python"}
	var lastErr error

	for _, pythonExe := range pythonExes {
		cmd := exec.Command(pythonExe, "-c", fmt.Sprintf(`
import sys
sys.path.insert(0, r'%s')
from render_effect import main
main()
`, filepath.ToSlash(filepath.Dir(os.Args[0]))), string(cfgJSON))
		cmd.Env = append(os.Environ(), "PYTHONUTF8=1")
		output, err := cmd.Output()
		if err != nil {
			lastErr = err
			continue
		}
		movPath := strings.TrimSpace(string(output))
		if movPath != "" && strings.HasSuffix(movPath, ".mov") {
			return movPath, nil
		}
	}

	if lastErr != nil {
		return "", fmt.Errorf("python render failed: %w", lastErr)
	}
	return "", nil
}

// renderEffectGoFallback is the old Go-based placeholder rendering
func renderEffectGoFallback(text string, effectType string, start, end float64, cfg EffectConfig) (string, error) {
	duration := end - start
	if duration <= 0 {
		return "", nil
	}

	maxAnimDuration := 1.0
	if effectType == "blur_in" || effectType == "fade_in" {
		if duration > maxAnimDuration {
			duration = maxAnimDuration
		}
	}

	fps := cfg.FPS
	if fps <= 0 {
		fps = 30
	}

	totalFrames := int(duration * float64(fps))
	if totalFrames < 1 {
		totalFrames = 1
	}

	ease := &Ease{}

	frameDir := cfg.OutputDir
	if frameDir == "" {
		frameDir = os.TempDir()
	}
	tmpDir, err := os.MkdirTemp(frameDir, "effect_frames_")
	if err != nil {
		return "", fmt.Errorf("create temp dir failed: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	for i := 0; i < totalFrames; i++ {
		frame := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
		clearRGBA(frame)

		progress := float64(i) / float64(max(1, totalFrames-1))
		spriteImg := renderTextImage(text, cfg)

		var frameImg *image.RGBA

		switch effectType {
		case "bounce_in":
			frameImg = bounceInFrame(spriteImg, progress, ease, cfg)
		case "scale_pop":
			frameImg = scalePopFrame(spriteImg, progress, ease, cfg)
		case "wave_bounce":
			frameImg = waveBounceFrame(spriteImg, progress, ease, cfg)
		case "rotate_pop":
			frameImg = rotatePopFrame(spriteImg, progress, ease, cfg)
		case "blur_in":
			frameImg = blurInFrame(spriteImg, progress, ease, cfg)
		default:
			frameImg = spriteImg
		}

		sw := frameImg.Bounds().Dx()
		sh := frameImg.Bounds().Dy()
		cx := cfg.Width/2 - sw/2
		cy := cfg.Height/2 - sh/2
		if cfg.AnchorMode == "bottom" && cfg.AnchorY > 0 {
			cy = int(cfg.AnchorY) - sh
			if cy < 0 {
				cy = cfg.Height/2 - sh/2
			}
		}

		draw.Draw(frame, frameImg.Bounds().Add(image.Pt(cx, cy)), frameImg, image.ZP, draw.Over)

		fPath := filepath.Join(tmpDir, fmt.Sprintf("frame_%04d.png", i))
		f, err := os.Create(fPath)
		if err != nil {
			return "", fmt.Errorf("create frame file failed: %w", err)
		}
		png.Encode(f, frame)
		f.Close()
	}

	outDir := cfg.OutputDir
	if outDir == "" {
		outDir = os.TempDir()
	}
	outputPath := filepath.Join(outDir, fmt.Sprintf("effect_%d.mov", time.Now().UnixNano()))
	cmd := exec.Command("ffmpeg", "-y",
		"-framerate", fmt.Sprintf("%d", fps),
		"-i", filepath.Join(tmpDir, "frame_%04d.png"),
		"-c:v", "qtrle",
		"-pix_fmt", "argb",
		outputPath,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ffmpeg encode failed: %w", err)
	}

	return outputPath, nil
}

// renderTextImage renders text as a simple image (placeholder - will be enhanced with proper font)
func renderTextImage(text string, cfg EffectConfig) *image.RGBA {
	fontSize := cfg.FontSize
	if fontSize <= 0 {
		fontSize = 48
	}

	// Estimate text width: Chinese chars ~1.2em, ASCII ~0.6em
	width := 0
	for _, r := range text {
		if r >= 0x4E00 && r <= 0x9FFF {
			width += fontSize * 12 / 10 // Chinese: 1.2x
		} else {
			width += fontSize * 6 / 10 // ASCII: 0.6x
		}
	}

	height := fontSize * 2
	padding := fontSize / 4

	imgWidth := width + padding*2
	imgHeight := height + padding*2

	if imgWidth < 1 {
		imgWidth = 1
	}
	if imgHeight < 1 {
		imgHeight = 1
	}

	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	// Draw text area background (for visibility)
	// fill with transparent, draw text placeholder rect
	textColor := cfg.TextColor
	if textColor.A == 0 {
		textColor = color.RGBA{253, 253, 255, 255}
	}

	// Draw a text-colored rectangle representing the text
	// (Simplified - proper font rendering needs external lib)
	rectH := fontSize * 8 / 10
	rectY := padding + (height - rectH) / 2

	for py := rectY; py < rectY+rectH && py < imgHeight; py++ {
		for px := padding; px < padding+width && px < imgWidth; px++ {
			// Simple gradient for visual interest
			ratio := float64(px-padding) / float64(max(1, width))
			img.SetRGBA(px, py, color.RGBA{
				R: textColor.R,
				G: textColor.G,
				B: textColor.B,
				A: 200 + uint8(55*math.Sin(ratio*math.Pi)),
			})
		}
	}

	return img
}

// clearRGBA sets all pixels to transparent
func clearRGBA(img *image.RGBA) {
	for i := range img.Pix {
		img.Pix[i] = 0
	}
}

func bounceInFrame(sprite *image.RGBA, progress float64, ease *Ease, cfg EffectConfig) *image.RGBA {
	bounce := ease.Bounce(min(progress*1.3, 1.0))
	offsetY := int(100 * (1 - bounce))

	result := image.NewRGBA(sprite.Bounds())
	draw.Draw(result, sprite.Bounds(), sprite, image.ZP, draw.Src)

	// Shift by offsetY
	if offsetY == 0 {
		return result
	}

	out := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
	clearRGBA(out)

	sw := sprite.Bounds().Dx()
	sh := sprite.Bounds().Dy()
	cx := cfg.Width/2 - sw/2
	cy := cfg.Height/2 - sh/2 + offsetY

	draw.Draw(out, result.Bounds().Add(image.Pt(cx, cy)), result, image.ZP, draw.Over)
	return out
}

func scalePopFrame(sprite *image.RGBA, progress float64, ease *Ease, cfg EffectConfig) *image.RGBA {
	scale := 0.1 + 0.9*ease.BackOut(progress)
	return scaleRGBA(sprite, scale)
}

func waveBounceFrame(sprite *image.RGBA, progress float64, ease *Ease, cfg EffectConfig) *image.RGBA {
	scaleX := 0.9 + 0.15*math.Sin(progress*math.Pi)
	scaleY := 1.0 + 0.1*progress*ease.CubicOut(progress)
	return scaleXY(sprite, scaleX, scaleY)
}

func rotatePopFrame(sprite *image.RGBA, progress float64, ease *Ease, cfg EffectConfig) *image.RGBA {
	angle := -20.0 + 25.0*ease.BackOut(progress) // -20 to 5 degrees
	return rotateRGBA(sprite, angle)
}

func blurInFrame(sprite *image.RGBA, progress float64, ease *Ease, cfg EffectConfig) *image.RGBA {
	radius := (1.0 - progress) * 10.0
	alpha := int(255 * (0.3 + 0.7*progress))

	blurred := boxBlurApprox(sprite, radius)

	// Apply alpha modulation
	b := blurred.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			p := blurred.RGBAAt(x, y)
			p.A = uint8(int(p.A) * alpha / 255)
			blurred.SetRGBA(x, y, p)
		}
	}

	return blurred
}

// scaleRGBA scales sprite by uniform scale factor using simple pixel replication
func scaleRGBA(sprite *image.RGBA, scale float64) *image.RGBA {
	if scale <= 0 {
		scale = 0.01
	}
	sw := sprite.Bounds().Dx()
	sh := sprite.Bounds().Dy()
	w := int(float64(sw) * scale)
	h := int(float64(sh) * scale)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			sx := int(float64(dx) / scale)
			sy := int(float64(dy) / scale)
			if sx >= sw {
				sx = sw - 1
			}
			if sy >= sh {
				sy = sh - 1
			}
			dst.SetRGBA(dx, dy, sprite.RGBAAt(sx, sy))
		}
	}
	return dst
}

// scaleXY scales sprite with different X and Y scale factors
func scaleXY(sprite *image.RGBA, scaleX, scaleY float64) *image.RGBA {
	sw := sprite.Bounds().Dx()
	sh := sprite.Bounds().Dy()
	w := int(float64(sw) * scaleX)
	h := int(float64(sh) * scaleY)
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			sx := int(float64(dx) / scaleX)
			sy := int(float64(dy) / scaleY)
			if sx >= sw {
				sx = sw - 1
			}
			if sy >= sh {
				sy = sh - 1
			}
			dst.SetRGBA(dx, dy, sprite.RGBAAt(sx, sy))
		}
	}
	return dst
}

// rotateRGBA rotates image by angle degrees around center
func rotateRGBA(src *image.RGBA, angle float64) *image.RGBA {
	theta := angle * math.Pi / 180.0
	cos := math.Cos(theta)
	sin := math.Sin(theta)

	w := src.Bounds().Dx()
	h := src.Bounds().Dy()

	// Calculate output bounds
	newW := int(math.Abs(float64(w)*cos) + math.Abs(float64(h)*sin))
	newH := int(math.Abs(float64(w)*sin) + math.Abs(float64(h)*cos))
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	clearRGBA(dst)

	cx, cy := w/2, h/2
	dcx, dcy := newW/2, newH/2

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx := x - cx
			dy := y - cy
			px := int(float64(dx)*cos - float64(dy)*sin)
			py := int(float64(dx)*sin + float64(dy)*cos)
			rx := px + dcx
			ry := py + dcy
			if rx >= 0 && rx < newW && ry >= 0 && ry < newH {
				dst.SetRGBA(rx, ry, src.RGBAAt(x, y))
			}
		}
	}
	return dst
}

// boxBlurApprox is a simplified box blur
func boxBlurApprox(src *image.RGBA, radius float64) *image.RGBA {
	if radius < 0.5 {
		return src
	}
	r := int(radius)
	if r > 10 {
		r = 10
	}

	w := src.Bounds().Dx()
	h := src.Bounds().Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	// Horizontal pass
	temp := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var rSum, gSum, bSum, aSum, count int64
			for k := -r; k <= r; k++ {
				nx := x + k
				if nx >= 0 && nx < w {
					p := src.RGBAAt(nx, y)
					rSum += int64(p.R)
					gSum += int64(p.G)
					bSum += int64(p.B)
					aSum += int64(p.A)
					count++
				}
			}
			if count > 0 {
				temp.SetRGBA(x, y, color.RGBA{
					R: uint8(rSum / count),
					G: uint8(gSum / count),
					B: uint8(bSum / count),
					A: uint8(aSum / count),
				})
			}
		}
	}

	// Vertical pass
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var rSum, gSum, bSum, aSum, count int64
			for k := -r; k <= r; k++ {
				ny := y + k
				if ny >= 0 && ny < h {
					p := temp.RGBAAt(x, ny)
					rSum += int64(p.R)
					gSum += int64(p.G)
					bSum += int64(p.B)
					aSum += int64(p.A)
					count++
				}
			}
			if count > 0 {
				dst.SetRGBA(x, y, color.RGBA{
					R: uint8(rSum / count),
					G: uint8(gSum / count),
					B: uint8(bSum / count),
					A: uint8(aSum / count),
				})
			}
		}
	}

	return dst
}

// ApplyEffectOverlays renders effect MOV clips and overlays them onto the video.
// effectsDir is used for intermediate files; falls back to os.TempDir() if empty.
func ApplyEffectOverlays(videoPath string, segments []Segment, width, height, fontSize int, colorHex, strokeColorHex, highlightColorHex string, effectsDir string) (string, error) {
	if len(segments) == 0 {
		return videoPath, nil
	}

	// Calculate bottom margin for subtitle positioning
	bottomMargin := int(float64(height) * 0.155)
	if bottomMargin < 90 {
		bottomMargin = 90
	}
	anchorY := float64(height - bottomMargin)

	cfg := EffectConfig{
		Width:       width,
		Height:      height,
		FPS:         30,
		FontSize:    fontSize,
		AnchorX:     float64(width / 2),
		AnchorY:     anchorY,
		AnchorMode:  "bottom",
		TextColor:   RGBAFromHex(colorHex),
		StrokeColor: RGBAFromHex(strokeColorHex),
		StrokeWidth: float64(fontSize) * 0.07,
		OutputDir:   effectsDir,
	}

	// Render each effect clip
	type overlayEntry struct {
		movPath string
		start   float64
		end     float64
	}

	var overlays []overlayEntry
	for _, seg := range segments {
		if seg.EffectType == "" || seg.EffectType == "none" || seg.Text == "" {
			continue
		}

		movPath, err := RenderEffect(seg.Text, seg.EffectType, seg.Start, seg.End, cfg)
		if err != nil {
			fmt.Printf("RenderEffect error for seg %d: %v\n", seg.SegID, err)
			continue
		}
		if movPath != "" {
			overlays = append(overlays, overlayEntry{movPath: movPath, start: seg.Start, end: seg.End})
		}
	}

	if len(overlays) == 0 {
		return videoPath, nil
	}

	// Build ffmpeg overlay command
	// Use overlay filter to stack all effect clips onto video
	overlayBaseDir := effectsDir
	if overlayBaseDir == "" {
		overlayBaseDir = os.TempDir()
	}
	tmpDir, err := os.MkdirTemp(overlayBaseDir, "effect_overlay_")
	if err != nil {
		return videoPath, fmt.Errorf("create temp dir failed: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	intermediatePath := filepath.Join(tmpDir, "with_effects.mp4")

	// Build complex filter for multiple overlays
	filterParts := []string{"[0:v]setsar=1[v0]"}
	inputLabels := []string{"0:v"}
	currentLabel := "v0"

	for i, ov := range overlays {
		inputIdx := i + 1
		filterParts = append(filterParts,
			fmt.Sprintf("[%d:v]setpts=PTS-STARTPTS+%.6f/TB[fxv%d]", inputIdx, ov.start, i+1))
		filterParts = append(filterParts,
			fmt.Sprintf("[%s][fxv%d]overlay=0:0:eof_action=pass[%s]", currentLabel, i+1, fmt.Sprintf("v%d", i+1)))
		currentLabel = fmt.Sprintf("v%d", i+1)
		inputLabels = append(inputLabels, fmt.Sprintf("%d:v", inputIdx))
	}

	// Build command: ffmpeg -i video -i mov1 -i mov2 ... -filter_complex "..." -map [final] -c:v libx264 output
	args := []string{
		"-y", "-i", videoPath,
	}
	for _, ov := range overlays {
		args = append(args, "-i", ov.movPath)
	}
	args = append(args,
		"-filter_complex", strings.Join(filterParts, ";"),
		"-map", fmt.Sprintf("[%s]", currentLabel),
		"-c:v", "libx264", "-preset", "fast", "-crf", "18",
		intermediatePath,
	)

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return videoPath, fmt.Errorf("ffmpeg overlay failed: %w", err)
	}

	// Replace original with result
	if err := os.Rename(intermediatePath, videoPath); err != nil {
		return videoPath, fmt.Errorf("rename output failed: %w", err)
	}

	// Cleanup MOV files
	for _, ov := range overlays {
		os.Remove(ov.movPath)
	}

	return videoPath, nil
}
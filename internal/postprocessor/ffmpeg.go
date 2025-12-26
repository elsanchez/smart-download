package postprocessor

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/elsanchez/smart-download/internal/domain"
)

// FFmpegProcessor implementa procesamiento con FFmpeg
type FFmpegProcessor struct {
	tempDir string
}

// NewFFmpegProcessor crea un nuevo procesador FFmpeg
func NewFFmpegProcessor(tempDir string) *FFmpegProcessor {
	return &FFmpegProcessor{
		tempDir: tempDir,
	}
}

// VideoInfo contiene información del video
type VideoInfo struct {
	Width         int
	Height        int
	VideoCodec    string
	AudioCodec    string
	Duration      float64
	Bitrate       int64
	FrameRate     float64
	HasVideo      bool
	HasAudio      bool
}

// GetVideoInfo obtiene información del video usando ffprobe
func (f *FFmpegProcessor) GetVideoInfo(ctx context.Context, inputPath string) (*VideoInfo, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		inputPath,
	}

	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe failed: %w", err)
	}

	var result struct {
		Streams []struct {
			CodecType     string  `json:"codec_type"`
			CodecName     string  `json:"codec_name"`
			Width         int     `json:"width"`
			Height        int     `json:"height"`
			RFrameRate    string  `json:"r_frame_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			BitRate  string `json:"bit_rate"`
		} `json:"format"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("parse ffprobe output: %w", err)
	}

	info := &VideoInfo{}

	for _, stream := range result.Streams {
		switch stream.CodecType {
		case "video":
			info.HasVideo = true
			info.VideoCodec = stream.CodecName
			info.Width = stream.Width
			info.Height = stream.Height

			// Parse frame rate (formato: "30/1" o "30000/1001")
			if stream.RFrameRate != "" {
				parts := strings.Split(stream.RFrameRate, "/")
				if len(parts) == 2 {
					num, _ := strconv.ParseFloat(parts[0], 64)
					den, _ := strconv.ParseFloat(parts[1], 64)
					if den > 0 {
						info.FrameRate = num / den
					}
				}
			}
		case "audio":
			info.HasAudio = true
			info.AudioCodec = stream.CodecName
		}
	}

	// Parse duration
	if result.Format.Duration != "" {
		info.Duration, _ = strconv.ParseFloat(result.Format.Duration, 64)
	}

	// Parse bitrate
	if result.Format.BitRate != "" {
		info.Bitrate, _ = strconv.ParseInt(result.Format.BitRate, 10, 64)
	}

	return info, nil
}

// IsWhatsAppCompatible verifica si el video es compatible con WhatsApp
func (f *FFmpegProcessor) IsWhatsAppCompatible(ctx context.Context, inputPath string) (bool, string, error) {
	info, err := f.GetVideoInfo(ctx, inputPath)
	if err != nil {
		return false, "", err
	}

	reasons := []string{}

	// Verificar codec de video (H.264)
	if info.VideoCodec != "h264" {
		reasons = append(reasons, fmt.Sprintf("video codec is %s (needs h264)", info.VideoCodec))
	}

	// Verificar codec de audio (AAC)
	if info.HasAudio && info.AudioCodec != "aac" {
		reasons = append(reasons, fmt.Sprintf("audio codec is %s (needs aac)", info.AudioCodec))
	}

	// Verificar resolución (máximo 1080p)
	if info.Height > 1080 {
		reasons = append(reasons, fmt.Sprintf("resolution is %dx%d (max 1920x1080)", info.Width, info.Height))
	}

	if len(reasons) > 0 {
		return false, strings.Join(reasons, "; "), nil
	}

	return true, "", nil
}

// ConvertToWhatsAppMP4 convierte el video a formato compatible con WhatsApp
func (f *FFmpegProcessor) ConvertToWhatsAppMP4(ctx context.Context, inputPath string) (string, error) {
	// Generar path de salida
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	outputPath := base + "_whatsapp.mp4"

	// Obtener info del video
	info, err := f.GetVideoInfo(ctx, inputPath)
	if err != nil {
		return "", fmt.Errorf("get video info: %w", err)
	}

	// Construir argumentos FFmpeg
	args := []string{
		"-i", inputPath,
		"-hide_banner",
		"-loglevel", "error",
	}

	// Video: H.264 con escala si es necesario
	if info.Height > 1080 {
		// Escalar manteniendo aspect ratio
		args = append(args,
			"-vf", "scale=-2:1080", // -2 asegura width divisible por 2
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
		)
	} else if info.VideoCodec != "h264" {
		// Solo re-encodear video
		args = append(args,
			"-c:v", "libx264",
			"-preset", "medium",
			"-crf", "23",
		)
	} else {
		// Copiar video sin re-encodear
		args = append(args, "-c:v", "copy")
	}

	// Audio: AAC
	if info.HasAudio {
		if info.AudioCodec != "aac" {
			args = append(args,
				"-c:a", "aac",
				"-b:a", "128k",
			)
		} else {
			// Copiar audio sin re-encodear
			args = append(args, "-c:a", "copy")
		}
	}

	// Formato MP4
	args = append(args,
		"-f", "mp4",
		"-movflags", "+faststart", // Optimizar para streaming
		"-y", // Sobrescribir
		outputPath,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, output)
	}

	return outputPath, nil
}

// ConvertToGIF convierte el video a GIF optimizado
func (f *FFmpegProcessor) ConvertToGIF(ctx context.Context, inputPath string, width int, startTime, duration string) (string, error) {
	// Generar path de salida
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)
	outputPath := base + ".gif"

	// Palette temporal para mejor calidad
	palettePath := filepath.Join(f.tempDir, "palette.png")
	defer os.Remove(palettePath)

	// Paso 1: Generar paleta de colores
	paletteArgs := []string{
		"-i", inputPath,
		"-hide_banner",
		"-loglevel", "error",
	}

	if startTime != "" {
		paletteArgs = append(paletteArgs, "-ss", startTime)
	}
	if duration != "" {
		paletteArgs = append(paletteArgs, "-t", duration)
	}

	paletteArgs = append(paletteArgs,
		"-vf", fmt.Sprintf("fps=15,scale=%d:-1:flags=lanczos,palettegen=stats_mode=diff", width),
		"-y",
		palettePath,
	)

	cmd := exec.CommandContext(ctx, "ffmpeg", paletteArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("generate palette: %w\nOutput: %s", err, output)
	}

	// Paso 2: Generar GIF usando paleta
	gifArgs := []string{
		"-i", inputPath,
		"-i", palettePath,
		"-hide_banner",
		"-loglevel", "error",
	}

	if startTime != "" {
		gifArgs = append(gifArgs, "-ss", startTime)
	}
	if duration != "" {
		gifArgs = append(gifArgs, "-t", duration)
	}

	gifArgs = append(gifArgs,
		"-lavfi", fmt.Sprintf("fps=15,scale=%d:-1:flags=lanczos[x];[x][1:v]paletteuse=dither=bayer:bayer_scale=5", width),
		"-y",
		outputPath,
	)

	cmd = exec.CommandContext(ctx, "ffmpeg", gifArgs...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("generate gif: %w\nOutput: %s", err, output)
	}

	return outputPath, nil
}

// OptimizeGIF optimiza un GIF existente para WhatsApp (<8MB, 498px)
func (f *FFmpegProcessor) OptimizeGIF(ctx context.Context, inputPath string) (string, error) {
	const (
		maxWidth = 498
		maxSize  = 8 * 1024 * 1024 // 8MB
	)

	// Verificar tamaño actual
	stat, err := os.Stat(inputPath)
	if err != nil {
		return "", fmt.Errorf("stat file: %w", err)
	}

	// Si ya es pequeño y correcto ancho, no procesar
	info, err := f.GetVideoInfo(ctx, inputPath)
	if err != nil {
		return "", err
	}

	if stat.Size() <= maxSize && info.Width <= maxWidth {
		return inputPath, nil
	}

	// Re-generar con ancho correcto
	return f.ConvertToGIF(ctx, inputPath, maxWidth, "", "")
}

// parseTimeToSeconds convierte varios formatos de tiempo a segundos
// Soporta:
// - Duraciones Go: "1m30s", "90s", "1h4m10s"
// - Segundos: "90"
// - HH:MM:SS: "00:01:30"
func parseTimeToSeconds(timeStr string) (string, error) {
	// Intentar parsear como duración Go (1m30s, 90s, 1h4m10s)
	if duration, err := time.ParseDuration(timeStr); err == nil {
		return fmt.Sprintf("%.3f", duration.Seconds()), nil
	}

	// Verificar si es un número simple (segundos)
	if _, err := strconv.ParseFloat(timeStr, 64); err == nil {
		return timeStr, nil
	}

	// Verificar si es formato HH:MM:SS
	parts := strings.Split(timeStr, ":")
	if len(parts) == 3 {
		h, err1 := strconv.Atoi(parts[0])
		m, err2 := strconv.Atoi(parts[1])
		s, err3 := strconv.ParseFloat(parts[2], 64)
		if err1 == nil && err2 == nil && err3 == nil {
			totalSeconds := float64(h*3600 + m*60) + s
			return fmt.Sprintf("%.3f", totalSeconds), nil
		}
	}

	return "", fmt.Errorf("invalid time format: %s (expected: 1m30s, 90, or 00:01:30)", timeStr)
}

// ClipVideo extrae un segmento del video
func (f *FFmpegProcessor) ClipVideo(ctx context.Context, inputPath, startTime, endTime string) (string, error) {
	// Normalizar tiempos a segundos
	startSeconds, err := parseTimeToSeconds(startTime)
	if err != nil {
		return "", fmt.Errorf("invalid start time: %w", err)
	}
	endSeconds, err := parseTimeToSeconds(endTime)
	if err != nil {
		return "", fmt.Errorf("invalid end time: %w", err)
	}

	// Generar path de salida
	ext := filepath.Ext(inputPath)
	base := strings.TrimSuffix(inputPath, ext)

	// Limpiar timestamps para nombre de archivo (usar formato original)
	startClean := strings.ReplaceAll(startTime, ":", "-")
	endClean := strings.ReplaceAll(endTime, ":", "-")
	outputPath := fmt.Sprintf("%s_clip_%s_%s%s", base, startClean, endClean, ext)

	args := []string{
		"-i", inputPath,
		"-ss", startSeconds,
		"-to", endSeconds,
		"-hide_banner",
		"-loglevel", "error",
		"-c", "copy", // Stream copy (sin re-encodear)
		"-avoid_negative_ts", "make_zero",
		"-y",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg clip: %w\nOutput: %s", err, output)
	}

	return outputPath, nil
}

// Process implementa PostProcessor.Process
func (f *FFmpegProcessor) Process(ctx context.Context, inputPath string, options *domain.DownloadOptions) (string, error) {
	currentPath := inputPath
	var err error

	// 1. Clipping si está especificado
	if options.ClipStart != "" && options.ClipEnd != "" {
		currentPath, err = f.ClipVideo(ctx, currentPath, options.ClipStart, options.ClipEnd)
		if err != nil {
			return "", fmt.Errorf("clip video: %w", err)
		}
		// Si se creó clip, eliminar original
		if currentPath != inputPath {
			os.Remove(inputPath)
		}
	}

	// 2. Conversión a GIF si está especificado
	if options.ConvertToGIF {
		width := 480
		if options.GIFWidth > 0 {
			width = options.GIFWidth
		}

		currentPath, err = f.ConvertToGIF(ctx, currentPath, width, "", "")
		if err != nil {
			return "", fmt.Errorf("convert to gif: %w", err)
		}
		// Eliminar video original después de conversión exitosa
		if !strings.HasSuffix(inputPath, ".gif") {
			os.Remove(inputPath)
		}
		return currentPath, nil
	}

	// 3. Conversión a WhatsApp MP4 (siempre, a menos que ya sea compatible)
	compatible, reason, err := f.IsWhatsAppCompatible(ctx, currentPath)
	if err != nil {
		return "", fmt.Errorf("check whatsapp compatibility: %w", err)
	}

	if !compatible {
		whatsappPath, err := f.ConvertToWhatsAppMP4(ctx, currentPath)
		if err != nil {
			return "", fmt.Errorf("convert to whatsapp mp4: %w (reason: %s)", err, reason)
		}
		// Eliminar original después de conversión exitosa
		if whatsappPath != currentPath {
			os.Remove(currentPath)
		}
		currentPath = whatsappPath
	}

	return currentPath, nil
}

// NeedsProcessing implementa PostProcessor.NeedsProcessing
func (f *FFmpegProcessor) NeedsProcessing(inputPath string, options *domain.DownloadOptions) (bool, error) {
	// Siempre procesar si hay clipping o conversión a GIF
	if options.ClipStart != "" || options.ClipEnd != "" || options.ConvertToGIF {
		return true, nil
	}

	// Para videos, verificar compatibilidad WhatsApp
	ctx := context.Background()
	compatible, _, err := f.IsWhatsAppCompatible(ctx, inputPath)
	if err != nil {
		// Si no podemos verificar, asumir que necesita procesamiento
		return true, nil
	}

	return !compatible, nil
}

// CheckFFmpegInstalled verifica que FFmpeg esté instalado
func CheckFFmpegInstalled() error {
	if err := exec.Command("ffmpeg", "-version").Run(); err != nil {
		return fmt.Errorf("ffmpeg not found: %w (install: sudo apt install ffmpeg)", err)
	}
	if err := exec.Command("ffprobe", "-version").Run(); err != nil {
		return fmt.Errorf("ffprobe not found: %w (install: sudo apt install ffmpeg)", err)
	}
	return nil
}

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

**smart-download** is a professional video/media download manager for Desktop Linux with background queue system, automatic WhatsApp-compatible MP4 conversion, cookie management, and FFmpeg post-processing.

## Project Structure

```
smart-download/
├── cmd/
│   ├── smart-downloadd/     # Daemon entry point
│   │   └── main.go          # 99 lines - initialization, dependency checks
│   └── smd/                 # CLI client tool
│       └── main.go          # 466 lines - flag parsing, commands (add, convert, status, list, stats)
├── internal/
│   ├── daemon/              # Daemon core
│   │   ├── queue.go         # Queue manager with worker pool (206 lines)
│   │   ├── server.go        # Unix socket server (108 lines)
│   │   └── handlers.go      # Request handlers (145 lines)
│   ├── domain/              # Domain entities
│   │   ├── download.go      # Download entity + DownloadOptions
│   │   └── account.go       # Account entity
│   ├── downloader/          # Download layer
│   │   ├── detector.go      # Platform detection (143 lines)
│   │   ├── ytdlp.go         # yt-dlp wrapper (188 lines)
│   │   ├── gallerydl.go     # gallery-dl wrapper (148 lines)
│   │   └── manager.go       # Auto-selector (62 lines)
│   ├── postprocessor/       # Post-processing layer
│   │   ├── ffmpeg.go        # FFmpeg wrapper (456 lines)
│   │   └── postprocessor.go # Interface definition
│   └── repository/          # Data layer
│       ├── download.go      # DownloadRepository interface
│       ├── account.go       # AccountRepository interface
│       └── sqlite/          # SQLite implementation
│           ├── db.go        # Database factory + migrations
│           ├── download.go  # Download repo (277 lines)
│           ├── account.go   # Account repo (233 lines)
│           └── migrations/  # SQL migrations
├── pkg/
│   └── client/              # Client library
│       └── client.go        # Unix socket client (163 lines)
├── Makefile                 # Build automation
├── README.md                # User documentation
└── go.mod                   # Go dependencies
```

## Architecture

### Clean Architecture Layers

```
Presentation Layer (cmd/)
    ↓
Application Layer (internal/daemon/)
    ↓
Domain Layer (internal/domain/)
    ↓
Infrastructure Layer (internal/downloader, internal/postprocessor, internal/repository)
```

### Key Components

**1. Daemon (smart-downloadd)**
- Unix socket server listening on `$XDG_RUNTIME_DIR/smart-download.sock`
- Queue manager with configurable worker pool (default: 3)
- Automatic dependency checking (yt-dlp, gallery-dl, ffmpeg)
- Graceful shutdown on SIGINT/SIGTERM

**2. Queue Manager** (`internal/daemon/queue.go`)
- Worker pool pattern with channel-based semaphore
- Processing pipeline: `pending → downloading → processing → completed`
- Polls database every 5 seconds for pending downloads
- Desktop notifications (notify-send)
- Clipboard integration (xsel/xclip)

**3. Downloader Layer** (`internal/downloader/`)
- Platform detection via URL patterns (12+ platforms)
- Username extraction with regex
- yt-dlp wrapper: 1800+ sites support
- gallery-dl wrapper: 100+ sites (pixiv, fanbox, etc.)
- Auto-selection based on URL

**4. Post-Processor Layer** (`internal/postprocessor/`)
- FFmpeg integration for video processing
- WhatsApp MP4 conversion (H.264 + AAC, ≤1080p)
- GIF generation with palette optimization
- Video clipping with stream copy
- Smart processing: only converts when needed

**5. Repository Pattern** (`internal/repository/`)
- Interface-based abstraction
- SQLite implementation with sqlx
- Database migrations with golang-migrate
- Proper NULL handling with sql.NullString/NullInt64

### Data Flow

**Queue-based downloads (smd add):**
```
CLI (smd) → Unix Socket → Daemon Handlers → Queue Manager
                                                  ↓
                                            Download Repo
                                                  ↓
                                            Downloader Manager
                                                  ↓
                                            Post-Processor (if needed)
                                                  ↓
                                            Update DB + Notify
```

**Local file conversion (smd convert):**
```
CLI (smd convert) → Direct FFmpeg Processor
                           ↓
                    WhatsApp MP4 Conversion
                           ↓
                    Output File (same directory)
```

Note: `smd convert` bypasses the daemon and processes files locally for immediate conversion without queue overhead.

## Development Environment

### Dependencies

**Runtime:**
```bash
# System packages
sudo apt install ffmpeg

# Python tools
pip install yt-dlp gallery-dl

# Go 1.21+
go version
```

**Development:**
```bash
# Install
make install

# Build
make build

# Test
go test ./...
```

### Go Modules

```go
require (
    github.com/golang-migrate/migrate/v4 v4.17.0
    github.com/jmoiron/sqlx v1.3.5
    github.com/mattn/go-sqlite3 v1.14.19
)
```

## Core Patterns

### 1. Repository Pattern

**Interface Definition** (`internal/repository/download.go`):
```go
type DownloadRepository interface {
    Create(ctx context.Context, dl *domain.Download) (int64, error)
    GetByID(ctx context.Context, id int64) (*domain.Download, error)
    UpdateStatus(ctx context.Context, id int64, status domain.DownloadStatus, errorMsg string) error
    // ... more methods
}
```

**SQLite Implementation** (`internal/repository/sqlite/download.go`):
- Uses sqlx for struct mapping
- Proper NULL handling with sql.NullString
- JSON serialization for options field

**Benefits:**
- Easy to swap database (PostgreSQL, MySQL)
- Testable with mocks
- Clear separation of concerns

### 2. Worker Pool Pattern

**Queue Manager** (`internal/daemon/queue.go:42-53`):
```go
type QueueManager struct {
    workerPool   chan struct{}  // Semaphore
    workers      int             // Max concurrent
    wg           sync.WaitGroup  // Wait for completion
}

// Acquire worker slot (blocks if full)
q.workerPool <- struct{}{}
go q.processDownload(dl)

// Release slot when done
defer func() { <-q.workerPool }()
```

**Benefits:**
- Limits concurrent downloads
- Prevents resource exhaustion
- Easy to configure (change `workers` variable)

### 3. Platform Detection

**Regex-based URL parsing** (`internal/downloader/detector.go`):
```go
func DetectPlatform(urlStr string) string {
    switch {
    case strings.Contains(urlStr, "youtube.com"):
        return "youtube"
    case strings.Contains(urlStr, "twitter.com"):
        return "twitter"
    // ... 10+ more platforms
    }
}
```

**Username Extraction**:
```go
patterns := map[string]*regexp.Regexp{
    "twitter": regexp.MustCompile(`(?:twitter\.com|x\.com)/([^/]+)`),
    "instagram": regexp.MustCompile(`instagram\.com/(?:stories/)?([^/]+)`),
}
```

### 4. Post-Processing Pipeline

**Three-stage processing** (`internal/postprocessor/ffmpeg.go:344-397`):

1. **Clipping** (optional): Extract segment with stream copy
2. **GIF Conversion** (optional): Generate with palette
3. **WhatsApp Conversion** (default): H.264 + AAC

**Smart processing:**
- Checks compatibility before converting
- Stream copy when already compatible (no re-encoding)
- Deletes intermediates after successful processing

## Database Schema

### Migrations

**Location:** `internal/repository/sqlite/migrations/`

**Pattern:** `XXX_description.up.sql` / `XXX_description.down.sql`

**Embedding:**
```go
//go:embed migrations/*.sql
var migrationsFS embed.FS
```

### Tables

**downloads:**
- Tracks download status and metadata
- `options` field: JSON-encoded DownloadOptions
- Foreign key to accounts table

**accounts:**
- Cookie management for platforms
- `is_active` for account rotation
- UNIQUE constraint on (platform, name)

## FFmpeg Integration

### Video Info Extraction

**Uses ffprobe with JSON output:**
```bash
ffprobe -v quiet -print_format json -show_format -show_streams video.mp4
```

**Parsed fields:**
- Video codec, resolution, frame rate
- Audio codec
- Duration, bitrate

### WhatsApp Conversion

**Criteria:**
- Video: H.264 (libx264)
- Audio: AAC
- Max resolution: 1920x1080
- Movflags: +faststart (for streaming)

**Smart conversion:**
```go
if info.VideoCodec == "h264" && info.AudioCodec == "aac" && info.Height <= 1080 {
    // Stream copy (no re-encoding)
    args = append(args, "-c:v", "copy", "-c:a", "copy")
} else {
    // Re-encode
    args = append(args, "-c:v", "libx264", "-preset", "medium", "-crf", "23")
}
```

### GIF Generation

**Two-pass palette method:**
1. Generate palette: `palettegen=stats_mode=diff`
2. Apply palette: `paletteuse=dither=bayer:bayer_scale=5`

**Settings:**
- 15 FPS (smooth + reasonable size)
- Configurable width (default: 480px)
- Lanczos scaling for quality

### Video Clipping

**Stream copy for speed:**
```bash
ffmpeg -i input.mp4 -ss 00:10 -to 00:30 -c copy output.mp4
```

**avoid_negative_ts:** Ensures proper timestamp handling

## Local File Conversion (smd convert)

The `convert` command provides local file processing without daemon involvement.

### Implementation (`cmd/smd/main.go:282-465`)

**Architecture:**
- Bypasses daemon and Unix socket entirely
- Direct FFmpeg processor instantiation
- Synchronous processing (blocks until complete)
- No database persistence (files only)

**Key functions:**
```go
func handleConvert(args []string) {
    // Parse flags: --recursive, --output, --check-only
    // Collect video files from paths/directories
    // Process each file with progress display
    // Show conversion summary
}

func collectVideoFiles(paths []string, recursive bool) []string {
    // Detect directories vs files
    // Filter by video extensions (11 formats)
    // Recursive traversal with filepath.Walk
    // Deduplication with seen map
}
```

**Supported formats:**
- `.mp4`, `.mkv`, `.avi`, `.mov`, `.webm`, `.flv`, `.wmv`, `.m4v`, `.mpg`, `.mpeg`, `.3gp`

**Processing pipeline:**
1. Collect input files (paths, globs, directories)
2. Filter by video extension
3. For each file:
   - Check WhatsApp compatibility with `IsWhatsAppCompatible()`
   - Skip if already compatible
   - Convert with `ConvertToWhatsAppMP4()` if needed
   - Track statistics (converted, compatible, failed)
4. Display summary

**Output naming:**
- Same directory as input: `{name}_whatsapp.mp4`
- Custom output directory: `--output /path/` → `/path/{name}_whatsapp.mp4`

**Flags:**
- `--recursive`: Process subdirectories
- `--output <dir>`: Custom output directory
- `--check-only`: Check compatibility without converting
- `--clip-start <time>`: Start time for clipping (HH:MM:SS or seconds)
- `--clip-end <time>`: End time for clipping (HH:MM:SS or seconds)

**Usage examples:**
```bash
# Single file
smd convert video.mp4

# Multiple files (shell expansion)
smd convert *.mp4

# Directory (non-recursive)
smd convert /path/to/videos/

# Recursive processing
smd convert /path/to/videos/ --recursive

# Check-only mode
smd convert /path/ --check-only

# Custom output
smd convert video.mp4 --output /tmp/

# Clipping (extract segment)
smd convert video.mp4 --clip-start 10 --clip-end 30
smd convert video.mp4 --clip-start 00:01:00 --clip-end 00:02:00
```

**Clipping feature:**
- Uses FFmpeg's `ClipVideo()` before WhatsApp conversion
- Supports both seconds (e.g., `10`) and HH:MM:SS format (e.g., `00:01:00`)
- Output filename includes clip range: `{name}_clip_{start}_{end}_whatsapp.mp4`
- Example: `video_clip_10_30_whatsapp.mp4` or `video_clip_000100_000200_whatsapp.mp4`
- Temporary clipped file is automatically cleaned up after conversion

**Design rationale:**
- **Why bypass daemon?** Local file conversion is synchronous, instant feedback desired
- **Why no database?** No queue tracking needed, files already exist locally
- **Why same directory default?** User expects output next to source (common FFmpeg pattern)
- **Why clip before convert?** Reduces processing time by converting only the needed segment

## Unix Socket Protocol

### Request Format

```json
{
  "action": "add|status|list|stats",
  "payload": <action-specific JSON>
}
```

### Response Format

```json
{
  "success": true|false,
  "data": <response data>,
  "error": "error message if success=false"
}
```

### Actions

**add:**
```json
{
  "url": "https://...",
  "options": {
    "clip_start": "10",
    "clip_end": "30",
    "convert_to_gif": false,
    "gif_width": 480,
    "no_convert": false,
    "resolution": "1080p",
    "audio_only": false
  },
  "account_id": 1
}
```

**status:**
```json
{"id": 123}
```

**list:**
```json
{"limit": 50}
```

**stats:** No payload

## Common Development Tasks

### Adding New Platform Support

1. Add pattern to `internal/downloader/detector.go`:
   ```go
   case strings.Contains(urlStr, "newplatform.com"):
       return "newplatform"
   ```

2. Add username regex:
   ```go
   "newplatform": regexp.MustCompile(`newplatform\.com/([^/]+)`),
   ```

3. Update `NeedsGalleryDL()` if needed

### Adding New Post-Processing Feature

1. Add option to `internal/domain/download.go`:
   ```go
   type DownloadOptions struct {
       // ...
       NewFeature bool `json:"new_feature,omitempty"`
   }
   ```

2. Implement in `internal/postprocessor/ffmpeg.go`

3. Add CLI flag in `cmd/smd/main.go`:
   ```go
   newFeature := addFlags.Bool("new-feature", false, "Description")
   ```

4. Update `Process()` method to apply feature

### Database Migration

1. Create migration files:
   ```bash
   cd internal/repository/sqlite/migrations/
   touch 003_add_feature.up.sql
   touch 003_add_feature.down.sql
   ```

2. Write SQL in `.up.sql`

3. Write rollback SQL in `.down.sql`

4. Restart daemon (auto-applies migrations)

## Testing

### Unit Tests

**Run all tests:**
```bash
go test ./...
```

**Run with coverage:**
```bash
go test -cover ./...
```

**Test specific package:**
```bash
go test ./internal/downloader/
```

### Integration Tests

**Database tests** (`internal/repository/sqlite/db_test.go`):
- Uses in-memory SQLite (`:memory:`)
- Tests migrations
- Tests CRUD operations

**Platform detection tests** (`internal/downloader/detector_test.go`):
- 6 test suites covering 30+ test cases
- Validates URL patterns
- Validates username extraction

### Manual Testing

```bash
# Start daemon
smart-downloadd

# Test basic download
smd add https://youtube.com/watch?v=xxx

# Test clipping
smd add https://youtube.com/watch?v=xxx --clip-start 10 --clip-end 30

# Test GIF
smd add https://youtube.com/watch?v=xxx --gif

# Test local file conversion
smd convert /path/to/video.mp4

# Test directory conversion
smd convert /path/to/videos/ --recursive

# Test convert with clipping
smd convert /path/to/video.mp4 --clip-start 10 --clip-end 30

# Check logs
tail -f ~/.local/share/smart-download/daemon.log  # if logging enabled
```

## Configuration

### Worker Count

**Edit:** `cmd/smart-downloadd/main.go:75`
```go
workers := 3  // Change to desired number
```

### Output Directory

**Edit:** `cmd/smart-downloadd/main.go:42`
```go
outputDir := filepath.Join(homeDir, "Downloads", "download_video")
```

### Polling Interval

**Edit:** `internal/daemon/queue.go:48`
```go
pollInterval: 5 * time.Second,  // Change as needed
```

### FFmpeg Settings

**WhatsApp conversion** (`internal/postprocessor/ffmpeg.go:167-172`):
```go
"-preset", "medium",  // Encoding speed (ultrafast, fast, medium, slow)
"-crf", "23",         // Quality (0=lossless, 51=worst, 23=default)
```

**GIF settings** (`internal/postprocessor/ffmpeg.go:238, 260`):
```go
fps=15  // Frame rate (lower = smaller file)
```

## Error Handling

### Pattern

**All errors propagate up:**
```go
if err != nil {
    return "", fmt.Errorf("operation failed: %w", err)
}
```

**Logging at appropriate levels:**
```go
log.Printf("Warning: %v", err)      // Non-fatal
log.Fatalf("Fatal error: %v", err)  // Fatal
```

### Database Errors

**NULL handling:**
```go
type downloadRow struct {
    OutputPath sql.NullString `db:"output_path"`
}

// Extract value
outputPath := row.OutputPath.String  // Empty string if NULL
```

### FFmpeg Errors

**Capture output:**
```go
cmd := exec.CommandContext(ctx, "ffmpeg", args...)
output, err := cmd.CombinedOutput()
if err != nil {
    return "", fmt.Errorf("ffmpeg failed: %w\nOutput: %s", err, output)
}
```

## Performance Considerations

### Download Performance

- **Parallel workers:** Default 3, adjust based on bandwidth
- **Stream copy:** Used when possible (no re-encoding)
- **Temp file cleanup:** Automatic after successful processing

### Database Performance

- **Indexes:** Created on `status`, `platform`, `created_at`
- **SQLite WAL mode:** Better concurrency
- **Connection pooling:** Single connection per daemon

### Memory Usage

- **Worker pool:** Limits concurrent operations
- **Temp files:** Cleaned up immediately after use
- **No file buffering:** Streams directly to disk

## Deployment

### Systemd Service

**Install:**
```bash
make install-systemd
systemctl --user enable smart-downloadd
systemctl --user start smart-downloadd
```

**Service file location:**
`~/.config/systemd/user/smart-downloadd.service`

**Logs:**
```bash
journalctl --user -u smart-downloadd -f
```

### File Locations

- **Binaries:** `~/.local/bin/smart-downloadd`, `~/.local/bin/smd`
- **Database:** `~/.local/share/smart-download/downloads.db`
- **Socket:** `$XDG_RUNTIME_DIR/smart-download.sock` (typically `/run/user/1000/`)
- **Output:** `~/Downloads/download_video/<platform>/`
- **Cookies:** `~/Documents/cookies/<platform>.txt`

## Troubleshooting

### Daemon won't start

**Check dependencies:**
```bash
yt-dlp --version
gallery-dl --version
ffmpeg -version
```

**Check socket path:**
```bash
echo $XDG_RUNTIME_DIR
# Should be /run/user/1000 (or your UID)
```

### Downloads fail

**Test manually:**
```bash
yt-dlp <url>
```

**Check database:**
```bash
sqlite3 ~/.local/share/smart-download/downloads.db
SELECT * FROM downloads WHERE status='failed';
```

### FFmpeg errors

**Test conversion manually:**
```bash
ffmpeg -i input.mp4 -c:v libx264 -c:a aac output.mp4
```

**Check codecs:**
```bash
ffprobe -hide_banner input.mp4
```

## Version Control

### Commit Message Format

```
<type>: <subject>

<body>

<footer>
```

**Types:** feat, fix, docs, refactor, test, chore

**Example:**
```
feat: Add video clipping support

- Implement ClipVideo in FFmpeg processor
- Add --clip-start/--clip-end CLI flags
- Update queue manager to handle clipping
- Add tests for time format parsing

Closes #123
```

### Branch Strategy

- `master`: Production-ready code
- Feature branches: `feature/description`
- Bug fixes: `fix/description`

## Current Version

**v0.1.0** - Initial release

### Features Implemented

- ✅ Background queue with parallel workers
- ✅ Platform detection (12+ platforms)
- ✅ Cookie management
- ✅ WhatsApp MP4 auto-conversion
- ✅ GIF generation
- ✅ Video clipping
- ✅ Desktop notifications
- ✅ Clipboard integration
- ✅ SQLite persistence
- ✅ Unix socket IPC
- ✅ Local file conversion (bypass daemon for existing files)

### Known Limitations

- Desktop Linux only (requires systemd + X11/Wayland)
- No web UI (CLI only)
- No download progress tracking (only status: pending/downloading/processing/completed)
- No bandwidth limiting
- No concurrent download limits per platform

### Future Roadmap

- Download progress reporting
- Web UI (optional)
- Bandwidth limiting
- Playlist support improvements
- Audio extraction enhancements
- Custom FFmpeg filter support

## Important Files

- `cmd/smart-downloadd/main.go` - Daemon initialization
- `cmd/smd/main.go` - CLI tool
- `internal/daemon/queue.go` - Core queue logic
- `internal/postprocessor/ffmpeg.go` - Video processing
- `internal/repository/sqlite/download.go` - Database layer
- `Makefile` - Build automation
- `README.md` - User documentation

## Contributing

When adding features:
1. Follow clean architecture pattern
2. Add tests for new functionality
3. Update README.md with usage examples
4. Create database migrations if schema changes
5. Update CLAUDE.md with architecture notes
6. Follow Go conventions (gofmt, golint)

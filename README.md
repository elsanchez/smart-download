# Smart Download Daemon

Professional video/media download manager with queue system, cookie management, and automatic WhatsApp-compatible MP4 conversion.

## Features

- ✅ **Background Queue**: Download multiple files in parallel (3 workers by default)
- ✅ **Platform Detection**: Auto-detects 12+ platforms (YouTube, Twitter, Instagram, etc.)
- ✅ **Cookie Management**: Account rotation for private content
- ✅ **Smart Naming**: Auto-generates filenames with platform/username/date
- ✅ **Post-Processing**: Automatic WhatsApp MP4 conversion, GIF creation, video clipping
- ✅ **Desktop Integration**: Clipboard copy, desktop notifications
- ✅ **SQLite Database**: Persistent queue and download history
- ✅ **Unix Socket IPC**: Fast communication between daemon and CLI

## Requirements

### System Dependencies

```bash
# Required
sudo apt install python3 python3-pip ffmpeg

# Python tools
pip install yt-dlp gallery-dl

# Desktop integration (already installed on Desktop Linux)
# xsel or xclip (clipboard)
# notify-send (notifications)
```

## Installation

```bash
cd ~/.local/src/smart-download

# Build and install
make install

# Or with systemd integration
make install-systemd
systemctl --user enable smart-downloadd
systemctl --user start smart-downloadd
```

## Usage

### Start Daemon

```bash
# Foreground (for testing)
smart-downloadd

# Or via systemd
systemctl --user start smart-downloadd
systemctl --user status smart-downloadd
```

### CLI Commands

```bash
# Add download to queue
smd add https://youtube.com/watch?v=xxx
smd https://youtube.com/watch?v=xxx  # shorthand

# Convert local files to WhatsApp MP4
smd convert video.mp4
smd convert *.mp4
smd convert /path/to/videos/ --recursive

# Check status
smd status 123

# List recent downloads
smd list
smd list 10  # limit to 10

# Queue statistics
smd stats

# Version
smd version
```

### Post-Processing Options

```bash
# Automatic WhatsApp MP4 conversion (enabled by default)
smd add https://youtube.com/watch?v=xxx

# Skip auto-conversion
smd add https://youtube.com/watch?v=xxx --no-convert

# Clip video segment (HH:MM:SS or seconds)
smd add https://youtube.com/watch?v=xxx --clip-start 10 --clip-end 30
smd add https://youtube.com/watch?v=xxx --clip-start 00:01:30 --clip-end 00:02:00

# Convert to GIF (with custom width)
smd add https://youtube.com/watch?v=xxx --gif --gif-width 480

# Combine clipping + GIF
smd add https://youtube.com/watch?v=xxx --clip-start 5 --clip-end 10 --gif

# Download with specific resolution
smd add https://youtube.com/watch?v=xxx --resolution 720p

# Extract audio only
smd add https://youtube.com/watch?v=xxx --audio-only
```

### WhatsApp MP4 Conversion

All downloaded videos are **automatically converted** to WhatsApp-compatible MP4 format:

- **Video codec**: H.264 (libx264)
- **Audio codec**: AAC
- **Max resolution**: 1920x1080 (auto-scaled if needed)
- **Faststart**: Enabled for web streaming
- **Smart processing**: Stream copy when already compatible (no re-encoding)

Example output:
```
youtube_25122025_Me_at_the_zoo_whatsapp.mp4  # Auto-converted
```

To skip conversion:
```bash
smd add <url> --no-convert
```

### GIF Conversion

Create high-quality GIFs with optimized palettes:

```bash
# Default width (480px)
smd add https://youtube.com/watch?v=xxx --gif

# Custom width
smd add https://youtube.com/watch?v=xxx --gif --gif-width 320

# Clip + GIF (useful for short animations)
smd add https://youtube.com/watch?v=xxx --clip-start 5 --clip-end 10 --gif
```

**Features**:
- Two-pass palette generation for better colors
- 15 FPS for smooth playback
- Bayer dithering
- Configurable width (maintains aspect ratio)

Example output:
```
youtube_25122025_Me_at_the_zoo.gif  # 480x360, 30MB
```

### Video Clipping

Extract specific segments without re-encoding:

```bash
# Using seconds
smd add <url> --clip-start 10 --clip-end 30

# Using HH:MM:SS format
smd add <url> --clip-start 00:01:30 --clip-end 00:02:00
```

**Features**:
- Fast stream copy (no quality loss)
- Supports both time formats
- Auto-converts to WhatsApp MP4 after clipping

Example output:
```
youtube_25122025_Me_at_the_zoo_clip_5_10_whatsapp.mp4  # 5 second clip
```

### Local File Conversion

Convert existing video files to WhatsApp-compatible MP4 format without downloading:

```bash
# Convert single file
smd convert video.mp4

# Convert multiple files
smd convert video1.mp4 video2.mkv video3.avi

# Convert all videos in directory
smd convert /path/to/videos/

# Convert recursively (includes subdirectories)
smd convert /path/to/videos/ --recursive

# Check which files need conversion (no actual conversion)
smd convert /path/to/videos/ --check-only

# Specify output directory
smd convert video.mp4 --output /path/to/output/

# Clip and convert (extract segment)
smd convert video.mp4 --clip-start 10 --clip-end 30
smd convert video.mp4 --clip-start 00:01:00 --clip-end 00:02:00
```

**Features**:
- Processes local video files directly (no daemon queue)
- Smart compatibility detection (skips already-compatible files)
- Supports 11 video formats (.mp4, .mkv, .avi, .mov, .webm, .flv, .wmv, .m4v, .mpg, .mpeg, .3gp)
- Progress reporting with file counts
- Conversion summary statistics
- Video clipping with `--clip-start` and `--clip-end` (supports both seconds and HH:MM:SS format)

**Supported formats**: All major video formats are auto-detected and converted to H.264 + AAC

Example output:
```
Found 3 video file(s)

[1/3] Processing: video.mkv
  → Converting to WhatsApp MP4...
    Reason: video codec is hevc (needs h264)
  ✓ Converted: video_whatsapp.mp4

[2/3] Processing: clip.mp4
  ✓ Already compatible (H.264 + AAC)

[3/3] Processing: recording.avi
  → Converting to WhatsApp MP4...
    Reason: video codec is mpeg4 (needs h264)
  ✓ Converted: recording_whatsapp.mp4

==================================================
Conversion Summary:
  Total files:      3
  Converted:        2
  Already compatible: 1
==================================================
```

## Architecture

```
smart-downloadd (daemon)
  ├── Unix Socket Server (/run/user/UID/smart-download.sock)
  ├── Queue Manager (3 parallel workers)
  │   └── Processing pipeline: pending → downloading → processing → completed
  ├── Database Layer (SQLite with migrations)
  ├── Downloader Manager
  │   ├── yt-dlp wrapper (1800+ sites)
  │   └── gallery-dl wrapper (100+ sites)
  ├── Post-Processor (FFmpeg)
  │   ├── WhatsApp MP4 converter (H.264 + AAC)
  │   ├── GIF generator (palette-based)
  │   └── Video clipper (stream copy)
  └── Repository Pattern (clean architecture)

smd (CLI client)
  └── Communicates via Unix socket
```

## Directory Structure

```
~/.local/share/smart-download/   # Database and temp files
  ├── downloads.db               # SQLite database
  └── temp/                      # Temporary files (palettes, etc.)
~/Downloads/download_video/       # Output files
  ├── youtube/
  │   ├── video_whatsapp.mp4    # Auto-converted
  │   ├── video_clip_5_10_whatsapp.mp4
  │   └── video.gif
  ├── twitter/
  ├── instagram/
  └── [platform]/
~/Documents/cookies/              # Cookie files
  ├── twitter.txt
  ├── instagram.txt
  └── [platform].txt
```

## Configuration

### Workers

Edit `cmd/smart-downloadd/main.go`:
```go
workers := 3 // Change to desired number
```

### Output Directory

Default: `~/Downloads/download_video/`

Edit `cmd/smart-downloadd/main.go`:
```go
outputDir := filepath.Join(homeDir, "Downloads", "download_video")
```

## Development

```bash
# Build
make build

# Run tests
make test

# Run daemon in foreground
make run

# Stop daemon
make stop

# Clean
make clean
```

## Database Schema

```sql
-- Downloads
CREATE TABLE downloads (
    id INTEGER PRIMARY KEY,
    url TEXT NOT NULL,
    platform TEXT,
    username TEXT,
    status TEXT DEFAULT 'pending',
    output_path TEXT,
    options TEXT,
    account_id INTEGER,
    created_at INTEGER,
    completed_at INTEGER,
    error_message TEXT
);

-- Accounts
CREATE TABLE accounts (
    id INTEGER PRIMARY KEY,
    platform TEXT NOT NULL,
    name TEXT NOT NULL,
    cookie_path TEXT NOT NULL,
    is_active INTEGER DEFAULT 0,
    last_used INTEGER,
    created_at INTEGER,
    UNIQUE(platform, name)
);
```

## API (Unix Socket)

### Add Download

```json
{
  "action": "add",
  "payload": {
    "url": "https://youtube.com/watch?v=xxx",
    "options": {
      "resolution": "1080p",
      "audio_only": false,
      "clip_start": "10",
      "clip_end": "30",
      "convert_to_gif": false,
      "gif_width": 480,
      "no_convert": false
    }
  }
}
```

**Options**:
- `resolution`: Video quality (1080p, 720p, 480p)
- `audio_only`: Extract audio only (boolean)
- `clip_start`: Start time for clipping (HH:MM:SS or seconds)
- `clip_end`: End time for clipping (HH:MM:SS or seconds)
- `convert_to_gif`: Convert to GIF (boolean)
- `gif_width`: GIF width in pixels (default: 480)
- `no_convert`: Skip WhatsApp MP4 conversion (boolean)

### Get Status

```json
{
  "action": "status",
  "payload": {
    "id": 123
  }
}
```

### List Downloads

```json
{
  "action": "list",
  "payload": {
    "limit": 50
  }
}
```

### Get Stats

```json
{
  "action": "stats"
}
```

## Troubleshooting

### Daemon won't start

```bash
# Check dependencies
yt-dlp --version
gallery-dl --version

# Check logs
journalctl --user -u smart-downloadd -f
```

### Socket connection refused

```bash
# Ensure daemon is running
systemctl --user status smart-downloadd

# Check socket exists
ls -la $XDG_RUNTIME_DIR/smart-download.sock
```

### Downloads fail

```bash
# Check download status
smd status <id>

# Test manually
yt-dlp <url>
```

## License

MIT

## Version

0.1.0 - Initial release

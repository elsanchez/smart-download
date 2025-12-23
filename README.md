# Smart Download Daemon

Professional video/media download manager with queue system, cookie management, and automatic WhatsApp-compatible MP4 conversion.

## Features

- ✅ **Background Queue**: Download multiple files in parallel (3 workers by default)
- ✅ **Platform Detection**: Auto-detects 12+ platforms (YouTube, Twitter, Instagram, etc.)
- ✅ **Cookie Management**: Account rotation for private content
- ✅ **Smart Naming**: Auto-generates filenames with platform/username/date
- ✅ **Post-Processing**: WhatsApp MP4 conversion, GIF creation, video clipping (coming soon)
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

## Architecture

```
smart-downloadd (daemon)
  ├── Unix Socket Server (/run/user/UID/smart-download.sock)
  ├── Queue Manager (3 parallel workers)
  ├── Database Layer (SQLite with migrations)
  ├── Downloader Manager
  │   ├── yt-dlp wrapper (1800+ sites)
  │   └── gallery-dl wrapper (100+ sites)
  └── Repository Pattern (clean architecture)

smd (CLI client)
  └── Communicates via Unix socket
```

## Directory Structure

```
~/.local/share/smart-download/   # Database
~/Downloads/download_video/       # Output files
  ├── youtube/
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
      "audio_only": false
    }
  }
}
```

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

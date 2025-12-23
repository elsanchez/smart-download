package downloader

import "testing"

func TestDetectPlatform(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "youtube"},
		{"https://youtu.be/dQw4w9WgXcQ", "youtube"},
		{"https://twitter.com/user/status/123", "twitter"},
		{"https://x.com/user/status/123", "twitter"},
		{"https://www.instagram.com/p/ABC123/", "instagram"},
		{"https://www.tiktok.com/@user/video/123", "tiktok"},
		{"https://vimeo.com/123456789", "vimeo"},
		{"https://www.reddit.com/r/videos/comments/abc/", "reddit"},
		{"https://pixiv.net/en/artworks/123456", "pixiv"},
		{"https://unknown-site.com/video", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := DetectPlatform(tt.url)
			if result != tt.expected {
				t.Errorf("DetectPlatform(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestNeedsGalleryDL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://www.youtube.com/watch?v=123", false},
		{"https://twitter.com/user/status/123", false},
		{"https://pixiv.net/en/artworks/123456", true},
		{"https://fanbox.cc/@user/posts/123", true},
		{"https://www.reddit.com/r/pics/comments/abc/", true},
		{"https://imgur.com/gallery/abc123", true},
		{"https://kemono.party/patreon/user/123", true},
		{"https://www.instagram.com/p/ABC123/", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := NeedsGalleryDL(tt.url)
			if result != tt.expected {
				t.Errorf("NeedsGalleryDL(%q) = %v, want %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestExtractUsername(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://twitter.com/elonmusk/status/123", "elonmusk"},
		{"https://x.com/NASA/status/456", "NASA"},
		{"https://www.instagram.com/natgeo/", "natgeo"},
		{"https://www.instagram.com/stories/natgeo/123", "natgeo"},
		{"https://www.tiktok.com/@billieeilish/video/123", "billieeilish"},
		{"https://www.youtube.com/@MrBeast/videos", "MrBeast"},
		{"https://www.reddit.com/user/spez/", "spez"},
		{"https://unknown-site.com/path/to/video", "path"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			result := ExtractUsername(tt.url)
			if result != tt.expected {
				t.Errorf("ExtractUsername(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

func TestGetCookieRequirementLevel(t *testing.T) {
	tests := []struct {
		platform string
		expected int
	}{
		{"twitter", 2},
		{"instagram", 2},
		{"pixiv", 2},
		{"reddit", 1},
		{"youtube", 0},
		{"vimeo", 0},
		{"unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.platform, func(t *testing.T) {
			result := GetCookieRequirementLevel(tt.platform)
			if result != tt.expected {
				t.Errorf("GetCookieRequirementLevel(%q) = %d, want %d", tt.platform, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"valid_name123", "valid_name123"},
		{"name with spaces", "namewithspaces"},
		{"name@with#special$chars", "namewithspecialchars"},
		{"name.with.dots", "name.with.dots"},
		{"name-with-dashes", "name-with-dashes"},
		{"émojis🔥are❌removed", "mojisareremoved"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

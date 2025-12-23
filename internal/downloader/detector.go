package downloader

import (
	"net/url"
	"regexp"
	"strings"
)

// Plataformas que funcionan mejor con gallery-dl
var galleryDLSites = []string{
	"pixiv.net",
	"artstation.com",
	"deviantart.com",
	"fanbox.cc",
	"fantia.jp",
	"kemono.party",
	"coomer.party",
	"reddit.com",
	"imgur.com",
	"pinterest.com",
	"tumblr.com",
	"flickr.com",
	"danbooru.donmai.us",
	"gelbooru.com",
	"rule34.xxx",
	"subscribestar.com",
	"gumroad.com",
	"discord.com",
}

// DetectPlatform detecta la plataforma desde la URL
func DetectPlatform(urlStr string) string {
	urlStr = strings.ToLower(urlStr)

	switch {
	case strings.Contains(urlStr, "youtube.com"), strings.Contains(urlStr, "youtu.be"):
		return "youtube"
	case strings.Contains(urlStr, "twitter.com"), strings.Contains(urlStr, "x.com"):
		return "twitter"
	case strings.Contains(urlStr, "instagram.com"):
		return "instagram"
	case strings.Contains(urlStr, "tiktok.com"):
		return "tiktok"
	case strings.Contains(urlStr, "vimeo.com"):
		return "vimeo"
	case strings.Contains(urlStr, "dailymotion.com"):
		return "dailymotion"
	case strings.Contains(urlStr, "twitch.tv"):
		return "twitch"
	case strings.Contains(urlStr, "reddit.com"):
		return "reddit"
	case strings.Contains(urlStr, "imgur.com"):
		return "imgur"
	case strings.Contains(urlStr, "pixiv.net"):
		return "pixiv"
	case strings.Contains(urlStr, "fanbox.cc"):
		return "fanbox"
	case strings.Contains(urlStr, "fantia.jp"):
		return "fantia"
	default:
		return "other"
	}
}

// NeedsGalleryDL verifica si la URL debe usar gallery-dl en lugar de yt-dlp
func NeedsGalleryDL(urlStr string) bool {
	urlStr = strings.ToLower(urlStr)

	for _, site := range galleryDLSites {
		if strings.Contains(urlStr, site) {
			return true
		}
	}

	return false
}

// ExtractUsername extrae el username desde la URL
func ExtractUsername(urlStr string) string {
	// Patrones de regex para diferentes plataformas
	patterns := map[string]*regexp.Regexp{
		"twitter":   regexp.MustCompile(`(?:twitter\.com|x\.com)/([^/]+)`),
		"instagram": regexp.MustCompile(`instagram\.com/(?:stories/)?([^/]+)`),
		"tiktok":    regexp.MustCompile(`tiktok\.com/@([^/]+)`),
		"youtube":   regexp.MustCompile(`youtube\.com/(?:@|c/|user/)([^/]+)`),
		"reddit":    regexp.MustCompile(`reddit\.com/(?:u|user)/([^/]+)`),
	}

	for _, re := range patterns {
		if matches := re.FindStringSubmatch(urlStr); len(matches) > 1 {
			username := strings.Trim(matches[1], "@")
			// Sanitizar username
			username = sanitizeFilename(username)
			if username != "" {
				return username
			}
		}
	}

	// Fallback: intentar extraer desde URL path
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return "user"
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) > 0 {
		username := sanitizeFilename(parts[0])
		if username != "" {
			return username
		}
	}

	return "user"
}

// sanitizeFilename sanitiza un string para usarlo como nombre de archivo
func sanitizeFilename(s string) string {
	// Remover caracteres no permitidos
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]`)
	s = re.ReplaceAllString(s, "")

	// Limitar longitud
	if len(s) > 50 {
		s = s[:50]
	}

	return s
}

// GetCookieRequirementLevel retorna el nivel de requerimiento de cookies
// 0 = opcional, 1 = recomendado, 2 = obligatorio
func GetCookieRequirementLevel(platform string) int {
	switch platform {
	case "twitter":
		return 2 // Twitter es obligatorio
	case "instagram", "pixiv", "fanbox", "fantia":
		return 2 // Contenido privado común
	case "reddit", "imgur":
		return 1 // Recomendado para NSFW
	default:
		return 0 // Opcional
	}
}

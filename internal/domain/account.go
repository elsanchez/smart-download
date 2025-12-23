package domain

import "time"

// Account representa una cuenta de plataforma con cookies
type Account struct {
	ID         int64
	Platform   string
	Name       string
	CookiePath string
	IsActive   bool
	LastUsed   *time.Time
	CreatedAt  time.Time
}

// Platform constants para las plataformas soportadas
const (
	PlatformTwitter     = "twitter"
	PlatformInstagram   = "instagram"
	PlatformPixiv       = "pixiv"
	PlatformFanbox      = "fanbox"
	PlatformFantia      = "fantia"
	PlatformDiscord     = "discord"
	PlatformYouTube     = "youtube"
	PlatformTikTok      = "tiktok"
	PlatformReddit      = "reddit"
	PlatformSubscribeStar = "subscribestar"
)

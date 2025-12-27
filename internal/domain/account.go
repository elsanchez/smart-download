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

	// Validation fields
	ValidatedAt      *time.Time
	ValidationStatus string
	ValidationError  *string
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

// Validation status constants
const (
	ValidationStatusValid   = "valid"
	ValidationStatusExpired = "expired"
	ValidationStatusInvalid = "invalid"
	ValidationStatusUnknown = "unknown"
)

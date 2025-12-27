package cookies

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/elsanchez/smart-download/internal/domain"
)

// ValidationResult contains the result of cookie validation
type ValidationResult struct {
	IsValid   bool
	Status    string // "valid", "expired", "invalid"
	Message   string
	ExpiresAt *time.Time
}

// CookieValidator handles validation of cookies
type CookieValidator struct {
	parser     *CookieParser
	httpClient *http.Client
}

// NewCookieValidator creates a new cookie validator
func NewCookieValidator() *CookieValidator {
	return &CookieValidator{
		parser: NewCookieParser(),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ValidateFile validates a cookie file by checking expiration timestamps
func (v *CookieValidator) ValidateFile(path string) (*ValidationResult, error) {
	// Parse cookie file
	cookies, err := v.parser.ParseFile(path)
	if err != nil {
		return &ValidationResult{
			IsValid: false,
			Status:  domain.ValidationStatusInvalid,
			Message: fmt.Sprintf("failed to parse cookie file: %v", err),
		}, nil
	}

	// Validate expiration
	return v.ValidateExpiration(cookies), nil
}

// ValidateExpiration checks if cookies are expired
func (v *CookieValidator) ValidateExpiration(cookies []NetscapeCookie) *ValidationResult {
	if len(cookies) == 0 {
		return &ValidationResult{
			IsValid: false,
			Status:  domain.ValidationStatusInvalid,
			Message: "no cookies found",
		}
	}

	now := time.Now().Unix()
	expiredCount := 0
	var earliestExpiration int64

	// Find earliest expiration and count expired cookies
	for i, cookie := range cookies {
		if i == 0 || cookie.Expiration < earliestExpiration {
			earliestExpiration = cookie.Expiration
		}

		if cookie.Expiration < now {
			expiredCount++
		}
	}

	expiresAt := time.Unix(earliestExpiration, 0)

	// All cookies expired
	if expiredCount == len(cookies) {
		return &ValidationResult{
			IsValid:   false,
			Status:    domain.ValidationStatusExpired,
			Message:   fmt.Sprintf("all %d cookies expired", len(cookies)),
			ExpiresAt: &expiresAt,
		}
	}

	// Some cookies expired
	if expiredCount > 0 {
		return &ValidationResult{
			IsValid:   false,
			Status:    domain.ValidationStatusExpired,
			Message:   fmt.Sprintf("%d of %d cookies expired", expiredCount, len(cookies)),
			ExpiresAt: &expiresAt,
		}
	}

	// All cookies valid
	return &ValidationResult{
		IsValid:   true,
		Status:    domain.ValidationStatusValid,
		Message:   fmt.Sprintf("all %d cookies valid, expires %s", len(cookies), expiresAt.Format("2006-01-02")),
		ExpiresAt: &expiresAt,
	}
}

// ValidateHTTP performs HTTP validation by making a test request to the platform
// This is optional and platform-specific
func (v *CookieValidator) ValidateHTTP(ctx context.Context, platform string, cookiePath string) (*ValidationResult, error) {
	// Platform-specific test endpoints that require authentication
	testEndpoints := map[string]string{
		domain.PlatformTwitter:       "https://api.twitter.com/1.1/account/verify_credentials.json",
		domain.PlatformInstagram:     "https://i.instagram.com/api/v1/users/web_profile_info/",
		domain.PlatformPixiv:         "https://www.pixiv.net/ajax/user/self",
		domain.PlatformYouTube:       "https://www.youtube.com/feed/account",
		domain.PlatformFanbox:        "https://api.fanbox.cc/user.me",
		domain.PlatformFantia:        "https://fantia.jp/api/v1/me",
		domain.PlatformDiscord:       "https://discord.com/api/v9/users/@me",
		domain.PlatformTikTok:        "https://www.tiktok.com/api/user/detail/",
		domain.PlatformReddit:        "https://oauth.reddit.com/api/v1/me",
		domain.PlatformSubscribeStar: "https://www.subscribestar.com/api/graphql/user",
		"facebook":                   "https://www.facebook.com/me",
	}

	endpoint, ok := testEndpoints[platform]
	if !ok {
		return &ValidationResult{
			IsValid: false,
			Status:  domain.ValidationStatusUnknown,
			Message: fmt.Sprintf("HTTP validation not supported for platform: %s", platform),
		}, nil
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Load cookies from file
	cookies, err := v.parser.ParseFile(cookiePath)
	if err != nil {
		return &ValidationResult{
			IsValid: false,
			Status:  domain.ValidationStatusInvalid,
			Message: fmt.Sprintf("failed to load cookies: %v", err),
		}, nil
	}

	// Add cookies to request
	// Skip cookies with invalid characters (backslashes, quotes, etc.)
	for _, cookie := range cookies {
		// Skip cookies with problematic characters that Go's net/http doesn't accept
		if strings.ContainsAny(cookie.Value, "\\\"") {
			continue
		}

		req.AddCookie(&http.Cookie{
			Name:   cookie.Name,
			Value:  cookie.Value,
			Domain: cookie.Domain,
			Path:   cookie.Path,
			Secure: cookie.Secure,
		})
	}

	// Make request
	resp, err := v.httpClient.Do(req)
	if err != nil {
		return &ValidationResult{
			IsValid: false,
			Status:  domain.ValidationStatusInvalid,
			Message: fmt.Sprintf("HTTP request failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode == http.StatusOK {
		return &ValidationResult{
			IsValid: true,
			Status:  domain.ValidationStatusValid,
			Message: "HTTP validation successful",
		}, nil
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &ValidationResult{
			IsValid: false,
			Status:  domain.ValidationStatusInvalid,
			Message: fmt.Sprintf("authentication failed (HTTP %d)", resp.StatusCode),
		}, nil
	}

	return &ValidationResult{
		IsValid: false,
		Status:  domain.ValidationStatusInvalid,
		Message: fmt.Sprintf("unexpected HTTP status: %d", resp.StatusCode),
	}, nil
}

// ValidateAccount validates an account's cookies by checking expiration timestamps
func (v *CookieValidator) ValidateAccount(account *domain.Account) (*ValidationResult, error) {
	return v.ValidateFile(account.CookiePath)
}

// ValidateAccountHTTP validates an account's cookies by making an HTTP request to the platform
// This is more reliable but slower than expiration-based validation
// Note: Always performs HTTP validation even if some cookies are expired,
// since non-critical cookies (UI preferences) may expire while auth cookies remain valid
func (v *CookieValidator) ValidateAccountHTTP(ctx context.Context, account *domain.Account) (*ValidationResult, error) {
	// Perform HTTP validation regardless of expiration status
	// The HTTP request will tell us if the cookies actually work
	return v.ValidateHTTP(ctx, account.Platform, account.CookiePath)
}

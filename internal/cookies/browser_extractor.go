package cookies

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/chrome"
	_ "github.com/browserutils/kooky/browser/chromium"
	_ "github.com/browserutils/kooky/browser/firefox"
	_ "github.com/browserutils/kooky/browser/edge"
	_ "github.com/browserutils/kooky/browser/opera"
)

// BrowserExtractor handles extraction of cookies from web browsers
type BrowserExtractor struct {
	parser *CookieParser
}

// NewBrowserExtractor creates a new browser cookie extractor
func NewBrowserExtractor() *BrowserExtractor {
	return &BrowserExtractor{
		parser: NewCookieParser(),
	}
}

// SupportedBrowsers returns a list of supported browser names
func (e *BrowserExtractor) SupportedBrowsers() []string {
	return []string{
		"chrome",
		"chromium",
		"firefox",
		"edge",
		"opera",
	}
}

// ExtractOptions contains options for browser cookie extraction
type ExtractOptions struct {
	Browser    string // Browser name (chrome, firefox, etc.)
	Domain     string // Domain to filter cookies (e.g., "facebook.com")
	OutputPath string // Path to save cookies in Netscape format
}

// Extract extracts cookies from a browser and saves them in Netscape format
func (e *BrowserExtractor) Extract(opts ExtractOptions) ([]NetscapeCookie, error) {
	// Normalize browser name
	browser := strings.ToLower(opts.Browser)

	// Create kooky filter for domain
	var filters []kooky.Filter
	if opts.Domain != "" {
		// Match cookies for domain and its subdomains
		filters = append(filters, kooky.DomainHasSuffix(opts.Domain))
	}

	// Read cookies from browser
	ctx := context.Background()
	cookies, err := kooky.ReadCookies(ctx, filters...)
	if err != nil {
		return nil, fmt.Errorf("read cookies from browser: %w", err)
	}

	if len(cookies) == 0 {
		return nil, fmt.Errorf("no cookies found for domain: %s", opts.Domain)
	}

	// Convert kooky cookies to Netscape format
	netscapeCookies := make([]NetscapeCookie, 0, len(cookies))
	for _, cookie := range cookies {
		// Skip cookies from different browsers if specified
		if browser != "" && cookie.Browser != nil {
			cookieBrowser := strings.ToLower(cookie.Browser.Browser())
			if !strings.Contains(cookieBrowser, browser) {
				continue
			}
		}

		// Convert to Netscape format
		domain := cookie.Domain
		if !strings.HasPrefix(domain, ".") && domain != "" {
			domain = "." + domain
		}

		httpOnly := "FALSE"
		if cookie.HttpOnly {
			httpOnly = "TRUE"
		}

		expiration := cookie.Expires.Unix()
		if expiration < 0 {
			expiration = 0
		}

		netscapeCookies = append(netscapeCookies, NetscapeCookie{
			Domain:     domain,
			Flag:       httpOnly,
			Path:       cookie.Path,
			Secure:     cookie.Secure,
			Expiration: expiration,
			Name:       cookie.Name,
			Value:      cookie.Value,
		})
	}

	if len(netscapeCookies) == 0 {
		return nil, fmt.Errorf("no cookies found for browser '%s' and domain '%s'", browser, opts.Domain)
	}

	// Save to file if output path provided
	if opts.OutputPath != "" {
		if err := e.saveCookies(netscapeCookies, opts.OutputPath); err != nil {
			return nil, fmt.Errorf("save cookies: %w", err)
		}
	}

	return netscapeCookies, nil
}

// saveCookies saves cookies to a file in Netscape format
func (e *BrowserExtractor) saveCookies(cookies []NetscapeCookie, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	// Write Netscape cookie file header
	if _, err := file.WriteString("# Netscape HTTP Cookie File\n"); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Write cookies
	for _, cookie := range cookies {
		secure := "FALSE"
		if cookie.Secure {
			secure = "TRUE"
		}

		line := fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			cookie.Domain,
			cookie.Flag,
			cookie.Path,
			secure,
			cookie.Expiration,
			cookie.Name,
			cookie.Value,
		)

		if _, err := file.WriteString(line); err != nil {
			return fmt.Errorf("write cookie: %w", err)
		}
	}

	return nil
}

// GetBrowserCookieCount returns the number of cookies for a domain in a browser
func (e *BrowserExtractor) GetBrowserCookieCount(browser, domain string) (int, error) {
	ctx := context.Background()
	filters := []kooky.Filter{kooky.DomainHasSuffix(domain)}
	cookies, err := kooky.ReadCookies(ctx, filters...)
	if err != nil {
		return 0, fmt.Errorf("read cookies: %w", err)
	}

	count := 0
	browser = strings.ToLower(browser)
	for _, cookie := range cookies {
		if cookie.Browser != nil {
			cookieBrowser := strings.ToLower(cookie.Browser.Browser())
			if strings.Contains(cookieBrowser, browser) {
				count++
			}
		}
	}

	return count, nil
}

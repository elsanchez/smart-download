package cookies

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// NetscapeCookie represents a single cookie from Netscape format
type NetscapeCookie struct {
	Domain     string
	Flag       string
	Path       string
	Secure     bool
	Expiration int64  // Unix timestamp
	Name       string
	Value      string
}

// CookieParser handles parsing of Netscape cookie format files
type CookieParser struct{}

// NewCookieParser creates a new cookie parser
func NewCookieParser() *CookieParser {
	return &CookieParser{}
}

// ParseFile parses a Netscape format cookie file
// Format: domain	flag	path	secure	expiration	name	value
func (p *CookieParser) ParseFile(path string) ([]NetscapeCookie, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open cookie file: %w", err)
	}
	defer file.Close()

	var cookies []NetscapeCookie
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		// Parse tab-separated values
		fields := strings.Split(line, "\t")
		if len(fields) < 7 {
			// Try space-separated as fallback
			fields = strings.Fields(line)
			if len(fields) < 7 {
				return nil, fmt.Errorf("line %d: invalid format (expected 7 fields, got %d)", lineNum, len(fields))
			}
		}

		// Parse expiration timestamp
		expiration, err := strconv.ParseInt(fields[4], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("line %d: invalid expiration timestamp: %w", lineNum, err)
		}

		// Parse secure flag
		secure := strings.ToUpper(fields[3]) == "TRUE"

		// Clean cookie value - remove surrounding quotes if present
		value := fields[6]
		if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
			value = strings.Trim(value, "\"")
		}

		cookie := NetscapeCookie{
			Domain:     fields[0],
			Flag:       fields[1],
			Path:       fields[2],
			Secure:     secure,
			Expiration: expiration,
			Name:       fields[5],
			Value:      value,
		}

		cookies = append(cookies, cookie)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read cookie file: %w", err)
	}

	if len(cookies) == 0 {
		return nil, fmt.Errorf("no valid cookies found in file")
	}

	return cookies, nil
}

// FindEarliestExpiration returns the earliest expiration time from a list of cookies
func (p *CookieParser) FindEarliestExpiration(cookies []NetscapeCookie) time.Time {
	if len(cookies) == 0 {
		return time.Time{}
	}

	earliest := cookies[0].Expiration
	for _, cookie := range cookies[1:] {
		if cookie.Expiration < earliest {
			earliest = cookie.Expiration
		}
	}

	return time.Unix(earliest, 0)
}

// DetectPlatform attempts to detect the platform from cookie domains
func (p *CookieParser) DetectPlatform(cookies []NetscapeCookie) string {
	if len(cookies) == 0 {
		return ""
	}

	// Count domain occurrences
	domainCounts := make(map[string]int)
	for _, cookie := range cookies {
		domain := strings.TrimPrefix(cookie.Domain, ".")
		domainCounts[domain]++
	}

	// Platform detection heuristics
	platformMap := map[string]string{
		"twitter.com":        "twitter",
		"x.com":              "twitter",
		"instagram.com":      "instagram",
		"pixiv.net":          "pixiv",
		"fanbox.cc":          "fanbox",
		"fantia.jp":          "fantia",
		"discord.com":        "discord",
		"youtube.com":        "youtube",
		"tiktok.com":         "tiktok",
		"reddit.com":         "reddit",
		"subscribestar.com":  "subscribestar",
		"subscribestar.adult": "subscribestar",
	}

	// Find most common matching domain
	maxCount := 0
	detectedPlatform := ""

	for domain, count := range domainCounts {
		if platform, ok := platformMap[domain]; ok {
			if count > maxCount {
				maxCount = count
				detectedPlatform = platform
			}
		}
	}

	return detectedPlatform
}

// CountCookies returns the total number of cookies
func (p *CookieParser) CountCookies(cookies []NetscapeCookie) int {
	return len(cookies)
}

// GetDomains returns a list of unique domains in the cookies
func (p *CookieParser) GetDomains(cookies []NetscapeCookie) []string {
	domainSet := make(map[string]bool)
	for _, cookie := range cookies {
		domain := strings.TrimPrefix(cookie.Domain, ".")
		domainSet[domain] = true
	}

	domains := make([]string, 0, len(domainSet))
	for domain := range domainSet {
		domains = append(domains, domain)
	}

	return domains
}

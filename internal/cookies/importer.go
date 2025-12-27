package cookies

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/elsanchez/smart-download/internal/domain"
	"github.com/elsanchez/smart-download/internal/repository"
)

// ImportOptions contains options for importing a cookie file
type ImportOptions struct {
	FilePath string
	Platform string
	Name     string
	Activate bool
	Validate bool
	Force    bool // Overwrite existing account
}

// CookieImporter orchestrates the cookie import workflow
type CookieImporter struct {
	parser      *CookieParser
	validator   *CookieValidator
	accountRepo repository.AccountRepository
}

// NewCookieImporter creates a new cookie importer
func NewCookieImporter(accountRepo repository.AccountRepository) *CookieImporter {
	return &CookieImporter{
		parser:      NewCookieParser(),
		validator:   NewCookieValidator(),
		accountRepo: accountRepo,
	}
}

// Import imports a cookie file to the database
func (i *CookieImporter) Import(ctx context.Context, opts ImportOptions) (*domain.Account, error) {
	// 1. Validate file path exists
	if _, err := os.Stat(opts.FilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("cookie file not found: %s", opts.FilePath)
	}

	// 2. Parse cookies to detect platform if not provided
	cookies, err := i.parser.ParseFile(opts.FilePath)
	if err != nil {
		return nil, fmt.Errorf("parse cookie file: %w", err)
	}

	// 3. Auto-detect platform if not provided
	platform := opts.Platform
	if platform == "" {
		platform = i.parser.DetectPlatform(cookies)
		if platform == "" {
			return nil, fmt.Errorf("could not auto-detect platform, please specify --platform")
		}
	}

	// 4. Generate account name if not provided
	name := opts.Name
	if name == "" {
		name, err = i.generateUniqueName(ctx, platform, "account")
		if err != nil {
			return nil, fmt.Errorf("generate account name: %w", err)
		}
	}

	// 5. Check for existing account
	existing, err := i.accountRepo.GetAll(ctx, platform)
	if err != nil {
		return nil, fmt.Errorf("check existing accounts: %w", err)
	}

	for _, acc := range existing {
		if acc.Name == name {
			if !opts.Force {
				return nil, fmt.Errorf("account already exists: %s/%s (use --force to overwrite)", platform, name)
			}
			// Delete existing account
			if err := i.accountRepo.Delete(ctx, acc.ID); err != nil {
				return nil, fmt.Errorf("delete existing account: %w", err)
			}
			break
		}
	}

	// 6. Copy cookie file to standard location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}

	cookieDir := filepath.Join(homeDir, "Documents", "cookies")
	if err := os.MkdirAll(cookieDir, 0755); err != nil {
		return nil, fmt.Errorf("create cookie directory: %w", err)
	}

	// Generate unique filename: platform_name.txt
	cookieFileName := fmt.Sprintf("%s_%s.txt", platform, name)
	cookiePath := filepath.Join(cookieDir, cookieFileName)

	// Copy file if source is different from destination
	absFilePath, _ := filepath.Abs(opts.FilePath)
	absCookiePath, _ := filepath.Abs(cookiePath)

	if absFilePath != absCookiePath {
		sourceData, err := os.ReadFile(opts.FilePath)
		if err != nil {
			return nil, fmt.Errorf("read source cookie file: %w", err)
		}

		if err := os.WriteFile(cookiePath, sourceData, 0600); err != nil {
			return nil, fmt.Errorf("write cookie file: %w", err)
		}
	}

	// 7. Validate cookies if requested
	var validationStatus string
	var validationError *string

	if opts.Validate {
		result, err := i.validator.ValidateFile(cookiePath)
		if err != nil {
			errMsg := err.Error()
			validationError = &errMsg
			validationStatus = domain.ValidationStatusInvalid
		} else {
			validationStatus = result.Status
			if !result.IsValid {
				validationError = &result.Message
			}
		}
	} else {
		validationStatus = domain.ValidationStatusUnknown
	}

	// 8. Create account in database
	account := &domain.Account{
		Platform:         platform,
		Name:             name,
		CookiePath:       cookiePath,
		IsActive:         false, // Will be set by SetActive if requested
		ValidationStatus: validationStatus,
		ValidationError:  validationError,
	}

	id, err := i.accountRepo.Create(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}

	account.ID = id

	// 9. Update validation if performed
	if opts.Validate {
		if err := i.accountRepo.UpdateValidation(ctx, id, validationStatus, validationError); err != nil {
			return nil, fmt.Errorf("update validation: %w", err)
		}
	}

	// 10. Set as active if requested
	if opts.Activate {
		if err := i.accountRepo.SetActive(ctx, platform, name); err != nil {
			return nil, fmt.Errorf("set active: %w", err)
		}
		account.IsActive = true
	}

	return account, nil
}

// generateUniqueName generates a unique account name
func (i *CookieImporter) generateUniqueName(ctx context.Context, platform string, baseName string) (string, error) {
	existing, err := i.accountRepo.GetAll(ctx, platform)
	if err != nil {
		return "", err
	}

	// Build set of existing names
	existingNames := make(map[string]bool)
	for _, acc := range existing {
		existingNames[acc.Name] = true
	}

	// Try baseName first
	if !existingNames[baseName] {
		return baseName, nil
	}

	// Try baseName_2, baseName_3, etc.
	for i := 2; i < 1000; i++ {
		name := fmt.Sprintf("%s_%d", baseName, i)
		if !existingNames[name] {
			return name, nil
		}
	}

	return "", fmt.Errorf("could not generate unique name after 1000 attempts")
}

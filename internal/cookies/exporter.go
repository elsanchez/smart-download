package cookies

import (
	"context"
	"fmt"
	"os"

	"github.com/elsanchez/smart-download/internal/repository"
)

// CookieExporter handles exporting cookies from the database
type CookieExporter struct {
	accountRepo repository.AccountRepository
}

// NewCookieExporter creates a new cookie exporter
func NewCookieExporter(accountRepo repository.AccountRepository) *CookieExporter {
	return &CookieExporter{
		accountRepo: accountRepo,
	}
}

// Export exports a cookie file from the database to the specified path
func (e *CookieExporter) Export(ctx context.Context, platform, name, outputPath string) error {
	// Get all accounts for platform
	accounts, err := e.accountRepo.GetAll(ctx, platform)
	if err != nil {
		return fmt.Errorf("get accounts: %w", err)
	}

	// Find account by name
	var account *struct {
		CookiePath string
	}

	for _, acc := range accounts {
		if acc.Name == name {
			account = &struct{ CookiePath string }{
				CookiePath: acc.CookiePath,
			}
			break
		}
	}

	if account == nil {
		return fmt.Errorf("account not found: %s/%s", platform, name)
	}

	// Check source cookie file exists
	if _, err := os.Stat(account.CookiePath); os.IsNotExist(err) {
		return fmt.Errorf("cookie file not found: %s", account.CookiePath)
	}

	// Read source file
	data, err := os.ReadFile(account.CookiePath)
	if err != nil {
		return fmt.Errorf("read cookie file: %w", err)
	}

	// Write to output path
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}

	return nil
}

// ExportByID exports a cookie file by account ID
func (e *CookieExporter) ExportByID(ctx context.Context, accountID int64, outputPath string) error {
	// Get account by ID
	account, err := e.accountRepo.GetByID(ctx, accountID)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}

	// Check source cookie file exists
	if _, err := os.Stat(account.CookiePath); os.IsNotExist(err) {
		return fmt.Errorf("cookie file not found: %s", account.CookiePath)
	}

	// Read source file
	data, err := os.ReadFile(account.CookiePath)
	if err != nil {
		return fmt.Errorf("read cookie file: %w", err)
	}

	// Write to output path
	if err := os.WriteFile(outputPath, data, 0600); err != nil {
		return fmt.Errorf("write output file: %w", err)
	}

	return nil
}

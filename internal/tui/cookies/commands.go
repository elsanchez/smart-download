package cookies

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/elsanchez/smart-download/internal/cookies"
	"github.com/elsanchez/smart-download/internal/domain"
	"github.com/elsanchez/smart-download/internal/repository"
)

// Async commands that return tea.Msg

func loadAccounts(repo repository.AccountRepository) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()

		platforms, err := repo.ListPlatforms(ctx)
		if err != nil {
			return accountsLoadedMsg{err: err}
		}

		var allAccounts []*domain.Account
		for _, platform := range platforms {
			accounts, err := repo.GetAll(ctx, platform)
			if err != nil {
				return accountsLoadedMsg{err: err}
			}
			allAccounts = append(allAccounts, accounts...)
		}

		return accountsLoadedMsg{accounts: allAccounts}
	}
}

func loadPlatforms(repo repository.AccountRepository) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		platforms, err := repo.ListPlatforms(ctx)
		return platformsLoadedMsg{platforms: platforms, err: err}
	}
}

func importCookie(importer *cookies.CookieImporter, opts cookies.ImportOptions) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		account, err := importer.Import(ctx, opts)
		return importCompleteMsg{account: account, err: err}
	}
}

func validateAccounts(validator *cookies.CookieValidator, repo repository.AccountRepository, accounts []*domain.Account, useHTTP bool) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		results := make(map[int64]*validationResult)

		for _, acc := range accounts {
			var result *cookies.ValidationResult
			var err error

			if useHTTP {
				result, err = validator.ValidateAccountHTTP(ctx, acc)
			} else {
				result, err = validator.ValidateAccount(acc)
			}

			if err != nil {
				results[acc.ID] = &validationResult{
					AccountID: acc.ID,
					Status:    "invalid",
					Message:   err.Error(),
					IsValid:   false,
				}
				continue
			}

			results[acc.ID] = &validationResult{
				AccountID: acc.ID,
				Status:    result.Status,
				Message:   result.Message,
				IsValid:   result.IsValid,
			}

			// Update validation in database
			var validationErr *string
			if !result.IsValid {
				validationErr = &result.Message
			}

			repo.UpdateValidation(ctx, acc.ID, result.Status, validationErr)
		}

		return validationCompleteMsg{results: results}
	}
}

func deleteAccount(repo repository.AccountRepository, id int64) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := repo.Delete(ctx, id)
		return deleteCompleteMsg{err: err}
	}
}

func activateAccount(repo repository.AccountRepository, platform, name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := repo.SetActive(ctx, platform, name)
		return activateCompleteMsg{err: err}
	}
}

func exportAccount(exporter *cookies.CookieExporter, platform, name, outputPath string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := exporter.Export(ctx, platform, name, outputPath)
		return exportCompleteMsg{path: outputPath, err: err}
	}
}

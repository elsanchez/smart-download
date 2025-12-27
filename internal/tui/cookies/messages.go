package cookies

import "github.com/elsanchez/smart-download/internal/domain"

// Message types for async operations

type accountsLoadedMsg struct {
	accounts []*domain.Account
	err      error
}

type platformsLoadedMsg struct {
	platforms []string
	err       error
}

type importCompleteMsg struct {
	account *domain.Account
	err     error
}

type validationCompleteMsg struct {
	results map[int64]*validationResult
}

type validationResult struct {
	AccountID int64
	Status    string
	Message   string
	IsValid   bool
}

type deleteCompleteMsg struct {
	err error
}

type activateCompleteMsg struct {
	err error
}

type exportCompleteMsg struct {
	path string
	err  error
}

type errorMsg struct {
	err error
}

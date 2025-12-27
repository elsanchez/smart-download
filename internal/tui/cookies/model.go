package cookies

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/elsanchez/smart-download/internal/cookies"
	"github.com/elsanchez/smart-download/internal/domain"
	"github.com/elsanchez/smart-download/internal/repository"
)

// view represents different screens in the TUI
type view int

const (
	viewList view = iota
	viewImport
	viewValidation
	viewHelp
)

// Model is the Bubbletea model for the cookie manager
type Model struct {
	// Navigation
	currentView view
	width       int
	height      int
	quitting    bool

	// Dependencies
	accountRepo repository.AccountRepository
	importer    *cookies.CookieImporter
	validator   *cookies.CookieValidator
	exporter    *cookies.CookieExporter

	// State
	accounts        []*domain.Account
	platforms       []string
	selectedAccount *domain.Account
	cursor          int

	// Components
	accountList   list.Model
	pathInput     textinput.Model
	platformInput textinput.Model
	nameInput     textinput.Model
	spinner       spinner.Model

	// Import state
	importPath      string
	importPlatform  string
	importName      string
	importActivate  bool
	importValidate  bool
	importFocusedField int

	// Validation state
	validationResults map[int64]*validationResult

	// UI state
	loading       bool
	statusMessage string
	errorMessage  string
}

// NewModel creates a new cookie manager TUI model
func NewModel(accountRepo repository.AccountRepository) Model {
	// Create text inputs
	pathInput := textinput.New()
	pathInput.Placeholder = "Path to cookie file"
	pathInput.Focus()
	pathInput.CharLimit = 256
	pathInput.Width = 60

	platformInput := textinput.New()
	platformInput.Placeholder = "Platform (auto-detect if empty)"
	platformInput.CharLimit = 50
	platformInput.Width = 40

	nameInput := textinput.New()
	nameInput.Placeholder = "Account name (auto-generate if empty)"
	nameInput.CharLimit = 50
	nameInput.Width = 40

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	// Create list
	accountList := list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	accountList.Title = "Cookie Accounts"
	accountList.SetShowStatusBar(false)
	accountList.SetFilteringEnabled(false)

	return Model{
		currentView:       viewList,
		accountRepo:       accountRepo,
		importer:          cookies.NewCookieImporter(accountRepo),
		validator:         cookies.NewCookieValidator(),
		exporter:          cookies.NewCookieExporter(accountRepo),
		accountList:       accountList,
		pathInput:         pathInput,
		platformInput:     platformInput,
		nameInput:         nameInput,
		spinner:           s,
		validationResults: make(map[int64]*validationResult),
		importActivate:    false,
		importValidate:    true,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadAccounts(m.accountRepo),
		loadPlatforms(m.accountRepo),
		m.spinner.Tick,
	)
}

// accountItem implements list.Item for the account list
type accountItem struct {
	account *domain.Account
}

func (i accountItem) Title() string {
	status := ""
	if i.account.IsActive {
		status = "⭐ "
	}

	validationIcon := ""
	switch i.account.ValidationStatus {
	case domain.ValidationStatusValid:
		validationIcon = "✓ "
	case domain.ValidationStatusExpired:
		validationIcon = "⚠ "
	case domain.ValidationStatusInvalid:
		validationIcon = "✗ "
	case domain.ValidationStatusUnknown:
		validationIcon = "❓ "
	}

	return status + validationIcon + i.account.Platform + "/" + i.account.Name
}

func (i accountItem) Description() string {
	desc := i.account.CookiePath
	if i.account.ValidationError != nil {
		desc += " - " + *i.account.ValidationError
	}
	return desc
}

func (i accountItem) FilterValue() string {
	return i.account.Platform + " " + i.account.Name
}

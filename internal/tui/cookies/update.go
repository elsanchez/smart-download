package cookies

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/elsanchez/smart-download/internal/cookies"
)

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Clear previous messages on keypress
		m.errorMessage = ""
		m.statusMessage = ""

		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.accountList.SetSize(msg.Width-4, msg.Height-10)
		return m, nil

	case accountsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.accounts = msg.accounts
		return m, nil

	case platformsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.platforms = msg.platforms
		return m, nil

	case importCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.statusMessage = "✓ Cookie imported successfully"
		m.currentView = viewList
		return m, tea.Batch(
			loadAccounts(m.accountRepo),
			loadPlatforms(m.accountRepo),
		)

	case validationCompleteMsg:
		m.loading = false
		m.validationResults = msg.results
		m.currentView = viewValidation
		return m, nil

	case deleteCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.statusMessage = "✓ Account deleted"
		return m, loadAccounts(m.accountRepo)

	case activateCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.statusMessage = "✓ Account activated"
		return m, loadAccounts(m.accountRepo)

	case exportCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		m.statusMessage = "✓ Exported to " + msg.path
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update focused input
	switch m.currentView {
	case viewImport:
		switch m.importFocusedField {
		case 0:
			m.pathInput, cmd = m.pathInput.Update(msg)
			cmds = append(cmds, cmd)
		case 1:
			m.platformInput, cmd = m.platformInput.Update(msg)
			cmds = append(cmds, cmd)
		case 2:
			m.nameInput, cmd = m.nameInput.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.currentView {
	case viewList:
		return m.handleListKeys(msg)
	case viewImport:
		return m.handleImportKeys(msg)
	case viewValidation, viewHelp:
		return m.handleDialogKeys(msg)
	}
	return m, nil
}

// handleListKeys handles keys in the list view
func (m Model) handleListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
		m.quitting = true
		return m, tea.Quit

	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.cursor < len(m.accounts)-1 {
			m.cursor++
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("i"))):
		// Import
		m.currentView = viewImport
		m.importFocusedField = 0
		m.pathInput.Focus()
		m.platformInput.Blur()
		m.nameInput.Blur()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("v"))):
		// Validate (expiration only)
		m.loading = true
		return m, validateAccounts(m.validator, m.accountRepo, m.accounts, false)

	case key.Matches(msg, key.NewBinding(key.WithKeys("V"))):
		// Validate HTTP (Shift+V)
		m.loading = true
		return m, validateAccounts(m.validator, m.accountRepo, m.accounts, true)

	case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
		// Activate selected
		if len(m.accounts) > 0 && m.cursor < len(m.accounts) {
			acc := m.accounts[m.cursor]
			m.loading = true
			return m, activateAccount(m.accountRepo, acc.Platform, acc.Name)
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
		// Delete selected
		if len(m.accounts) > 0 && m.cursor < len(m.accounts) {
			acc := m.accounts[m.cursor]
			m.loading = true
			return m, deleteAccount(m.accountRepo, acc.ID)
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("e"))):
		// Export selected (to temp file for demo)
		if len(m.accounts) > 0 && m.cursor < len(m.accounts) {
			acc := m.accounts[m.cursor]
			outputPath := "/tmp/cookie_export_" + acc.Platform + "_" + acc.Name + ".txt"
			m.loading = true
			return m, exportAccount(m.exporter, acc.Platform, acc.Name, outputPath)
		}
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("?"))):
		// Help
		m.currentView = viewHelp
		return m, nil
	}

	return m, nil
}

// handleImportKeys handles keys in the import view
func (m Model) handleImportKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
		// Cancel import
		m.currentView = viewList
		m.pathInput.SetValue("")
		m.platformInput.SetValue("")
		m.nameInput.SetValue("")
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		// Next field
		m.importFocusedField = (m.importFocusedField + 1) % 3
		m.updateImportFocus()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
		// Previous field
		m.importFocusedField--
		if m.importFocusedField < 0 {
			m.importFocusedField = 2
		}
		m.updateImportFocus()
		return m, nil

	case key.Matches(msg, key.NewBinding(key.WithKeys(" "))):
		// Toggle checkboxes only - let space pass through to text inputs
		if m.importFocusedField == 3 {
			m.importActivate = !m.importActivate
			return m, nil
		} else if m.importFocusedField == 4 {
			m.importValidate = !m.importValidate
			return m, nil
		}
		// Don't return here - let space propagate to text inputs

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		// Import
		if m.pathInput.Value() == "" {
			m.errorMessage = "Cookie file path is required"
			return m, nil
		}

		opts := cookies.ImportOptions{
			FilePath: m.pathInput.Value(),
			Platform: m.platformInput.Value(),
			Name:     m.nameInput.Value(),
			Activate: m.importActivate,
			Validate: m.importValidate,
			Force:    false,
		}

		m.loading = true
		return m, importCookie(m.importer, opts)
	}

	return m, nil
}

// handleDialogKeys handles keys in dialog views (validation, help)
func (m Model) handleDialogKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Any key returns to list
	m.currentView = viewList
	return m, nil
}

// updateImportFocus updates which input field is focused
func (m *Model) updateImportFocus() {
	switch m.importFocusedField {
	case 0:
		m.pathInput.Focus()
		m.platformInput.Blur()
		m.nameInput.Blur()
	case 1:
		m.pathInput.Blur()
		m.platformInput.Focus()
		m.nameInput.Blur()
	case 2:
		m.pathInput.Blur()
		m.platformInput.Blur()
		m.nameInput.Focus()
	}
}

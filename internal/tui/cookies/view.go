package cookies

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/elsanchez/smart-download/internal/domain"
)

// Styles with adaptive colors for light/dark backgrounds
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "63", Dark: "205"}).
			MarginLeft(2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "240", Dark: "250"})

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "160", Dark: "9"}).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "34", Dark: "10"}).
			Bold(true)

	spinnerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "63", Dark: "205"})

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: "63", Dark: "63"}).
			Padding(1, 2)

	activeInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "63", Dark: "205"})

	inactiveInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "240", Dark: "250"})
)

// View renders the current view
func (m Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	var content string

	switch m.currentView {
	case viewList:
		content = m.viewList()
	case viewImport:
		content = m.viewImport()
	case viewValidation:
		content = m.viewValidation()
	case viewHelp:
		content = m.viewHelp()
	default:
		content = m.viewList()
	}

	// Add status/error messages
	if m.errorMessage != "" {
		content += "\n" + errorStyle.Render("Error: "+m.errorMessage)
	} else if m.statusMessage != "" {
		content += "\n" + successStyle.Render(m.statusMessage)
	}

	if m.loading {
		content += "\n" + m.spinner.View() + " Loading..."
	}

	return content
}

// viewList renders the account list view
func (m Model) viewList() string {
	title := titleStyle.Render("🍪 Cookie Manager")

	// Group accounts by platform
	platformGroups := make(map[string][]*accountItem)
	for _, acc := range m.accounts {
		item := &accountItem{account: acc}
		platformGroups[acc.Platform] = append(platformGroups[acc.Platform], item)
	}

	var content strings.Builder
	content.WriteString(title + "\n\n")

	if len(m.accounts) == 0 {
		content.WriteString("  No accounts found. Press 'i' to import cookies.\n")
	} else {
		content.WriteString(fmt.Sprintf("  %d accounts across %d platforms\n\n", len(m.accounts), len(platformGroups)))

		// Render accounts by platform
		for _, platform := range m.platforms {
			items, ok := platformGroups[platform]
			if !ok {
				continue
			}

			content.WriteString(fmt.Sprintf("  %s (%d):\n", platform, len(items)))

			for _, item := range items {
				cursor := "  "
				if m.cursor == m.findAccountIndex(item.account) {
					cursor = "▸ "
				}

				status := ""
				if item.account.IsActive {
					status = "⭐"
				}

				validIcon := ""
				switch item.account.ValidationStatus {
				case "valid":
					validIcon = "✓"
				case "expired":
					validIcon = "⚠"
				case "invalid":
					validIcon = "✗"
				default:
					validIcon = "❓"
				}

				content.WriteString(fmt.Sprintf("  %s%s %s %-20s\n",
					cursor, status, validIcon, item.account.Name))

				if item.account.ValidationError != nil && m.cursor == m.findAccountIndex(item.account) {
					content.WriteString(fmt.Sprintf("     %s\n", helpStyle.Render(*item.account.ValidationError)))
				}
			}
			content.WriteString("\n")
		}
	}

	// Help
	help := "\n" + helpStyle.Render(
		"  ↑/k up • ↓/j down • i import • v validate • V HTTP validate • a activate • d delete • e export • ? help • q quit",
	)

	return content.String() + help
}

// viewImport renders the import form
func (m Model) viewImport() string {
	title := titleStyle.Render("Import Cookie File")

	var b strings.Builder
	b.WriteString(title + "\n\n")

	// Path input
	if m.importFocusedField == 0 {
		b.WriteString(activeInputStyle.Render("  Cookie File Path:") + "\n")
	} else {
		b.WriteString(inactiveInputStyle.Render("  Cookie File Path:") + "\n")
	}
	b.WriteString("  " + m.pathInput.View() + "\n\n")

	// Platform input
	if m.importFocusedField == 1 {
		b.WriteString(activeInputStyle.Render("  Platform (optional):") + "\n")
	} else {
		b.WriteString(inactiveInputStyle.Render("  Platform (optional):") + "\n")
	}
	b.WriteString("  " + m.platformInput.View() + "\n\n")

	// Name input
	if m.importFocusedField == 2 {
		b.WriteString(activeInputStyle.Render("  Account Name (optional):") + "\n")
	} else {
		b.WriteString(inactiveInputStyle.Render("  Account Name (optional):") + "\n")
	}
	b.WriteString("  " + m.nameInput.View() + "\n\n")

	// Checkboxes
	activateBox := "[ ]"
	if m.importActivate {
		activateBox = "[✓]"
	}

	validateBox := "[ ]"
	if m.importValidate {
		validateBox = "[✓]"
	}

	b.WriteString(fmt.Sprintf("  %s Set as active\n", activateBox))
	b.WriteString(fmt.Sprintf("  %s Validate cookies\n\n", validateBox))

	// Help
	help := helpStyle.Render("  Tab next field • Enter import • Esc cancel • Space toggle checkbox")

	return boxStyle.Render(b.String()) + "\n\n" + help
}

// viewValidation renders the validation results
func (m Model) viewValidation() string {
	title := titleStyle.Render("Validation Results")

	var b strings.Builder
	b.WriteString(title + "\n\n")

	if len(m.validationResults) == 0 {
		b.WriteString("  No validation results available.\n")
	} else {
		validCount := 0
		expiredCount := 0
		invalidCount := 0

		b.WriteString("  Platform     Account              Status    Message\n")
		b.WriteString("  " + strings.Repeat("─", 70) + "\n")

		for _, acc := range m.accounts {
			result, ok := m.validationResults[acc.ID]
			if !ok {
				continue
			}

			icon := "?"
			switch result.Status {
			case "valid":
				icon = "✓"
				validCount++
			case "expired":
				icon = "⚠"
				expiredCount++
			case "invalid":
				icon = "✗"
				invalidCount++
			}

			b.WriteString(fmt.Sprintf("  %-12s %-20s %s %-8s %s\n",
				acc.Platform,
				acc.Name,
				icon,
				result.Status,
				result.Message,
			))
		}

		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  Summary: %d valid, %d expired, %d invalid\n",
			validCount, expiredCount, invalidCount))
	}

	help := "\n" + helpStyle.Render("  Press any key to return to list")

	return b.String() + help
}

// viewHelp renders the help screen
func (m Model) viewHelp() string {
	title := titleStyle.Render("Help")

	help := `
  Navigation:
    ↑/k        Move up
    ↓/j        Move down
    Enter      Select
    Esc        Go back / Cancel
    q          Quit

  Actions (from list view):
    i          Import new cookie
    v          Validate expiration (fast)
    V          Validate HTTP (slow but reliable)
    a          Activate selected
    d          Delete selected
    e          Export selected
    ?          Show this help

  Import Form:
    Tab        Next field
    Shift+Tab  Previous field
    Space      Toggle checkbox
    Enter      Import

  Tips:
    - Platform auto-detection from cookie domains
    - Account names auto-generated if not provided
    - Validation checks cookie expiration timestamps
    - Active account is used for downloads
`

	return title + "\n" + help + "\n" + helpStyle.Render("  Press any key to return")
}

// Helper function to find account index
func (m Model) findAccountIndex(acc *domain.Account) int {
	for i, a := range m.accounts {
		if a.ID == acc.ID {
			return i
		}
	}
	return -1
}

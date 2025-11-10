package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rebeliceyang/lazypg/internal/models"
	"github.com/rebeliceyang/lazypg/internal/ui/theme"
)

// FavoritesMode represents the dialog mode
type FavoritesMode int

const (
	FavoritesModeList FavoritesMode = iota
	FavoritesModeAdd
	FavoritesModeEdit
)

// ExecuteFavoriteMsg is sent when a favorite should be executed
type ExecuteFavoriteMsg struct {
	Favorite models.Favorite
}

// CloseFavoritesDialogMsg is sent when dialog should close
type CloseFavoritesDialogMsg struct{}

// FavoritesDialog manages favorite queries
type FavoritesDialog struct {
	Width  int
	Height int
	Theme  theme.Theme

	// State
	mode      FavoritesMode
	favorites []models.Favorite
	selected  int
	offset    int

	// Add/Edit state
	nameInput        string
	descriptionInput string
	queryInput       string
	tagsInput        string
	currentField     int // 0=name, 1=description, 2=query, 3=tags

	// Search
	searchQuery string
}

// NewFavoritesDialog creates a new favorites dialog
func NewFavoritesDialog(th theme.Theme) *FavoritesDialog {
	return &FavoritesDialog{
		Width:     80,
		Height:    30,
		Theme:     th,
		mode:      FavoritesModeList,
		favorites: []models.Favorite{},
		selected:  0,
		offset:    0,
	}
}

// SetFavorites updates the favorites list
func (fd *FavoritesDialog) SetFavorites(favorites []models.Favorite) {
	fd.favorites = favorites
	fd.selected = 0
	fd.offset = 0
}

// Update handles keyboard input
func (fd *FavoritesDialog) Update(msg tea.KeyMsg) (*FavoritesDialog, tea.Cmd) {
	switch fd.mode {
	case FavoritesModeList:
		return fd.handleListMode(msg)
	case FavoritesModeAdd, FavoritesModeEdit:
		return fd.handleEditMode(msg)
	}
	return fd, nil
}

func (fd *FavoritesDialog) handleListMode(msg tea.KeyMsg) (*FavoritesDialog, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		return fd, func() tea.Msg {
			return CloseFavoritesDialogMsg{}
		}
	case "up", "k":
		if fd.selected > 0 {
			fd.selected--
			if fd.selected < fd.offset {
				fd.offset = fd.selected
			}
		}
	case "down", "j":
		if fd.selected < len(fd.favorites)-1 {
			fd.selected++
			visibleHeight := fd.Height - 10
			if fd.selected >= fd.offset+visibleHeight {
				fd.offset = fd.selected - visibleHeight + 1
			}
		}
	case "enter":
		// Execute selected favorite
		if fd.selected < len(fd.favorites) {
			fav := fd.favorites[fd.selected]
			return fd, func() tea.Msg {
				return ExecuteFavoriteMsg{Favorite: fav}
			}
		}
	case "a", "n":
		// Add new favorite
		fd.mode = FavoritesModeAdd
		fd.nameInput = ""
		fd.descriptionInput = ""
		fd.queryInput = ""
		fd.tagsInput = ""
		fd.currentField = 0
	case "e":
		// Edit selected favorite
		if fd.selected < len(fd.favorites) {
			fav := fd.favorites[fd.selected]
			fd.mode = FavoritesModeEdit
			fd.nameInput = fav.Name
			fd.descriptionInput = fav.Description
			fd.queryInput = fav.Query
			fd.tagsInput = strings.Join(fav.Tags, ", ")
			fd.currentField = 0
		}
	case "d", "x":
		// Delete - handled by parent
	}
	return fd, nil
}

func (fd *FavoritesDialog) handleEditMode(msg tea.KeyMsg) (*FavoritesDialog, tea.Cmd) {
	switch msg.String() {
	case "esc":
		fd.mode = FavoritesModeList
	case "tab":
		fd.currentField = (fd.currentField + 1) % 4
	case "shift+tab":
		fd.currentField = (fd.currentField - 1 + 4) % 4
	case "backspace":
		fd.deleteChar()
	case "enter":
		if fd.currentField == 3 {
			// Save and close
			fd.mode = FavoritesModeList
			// Parent will handle actual save
		} else {
			fd.currentField++
		}
	default:
		if len(msg.String()) == 1 {
			fd.addChar(msg.String())
		}
	}
	return fd, nil
}

func (fd *FavoritesDialog) addChar(ch string) {
	switch fd.currentField {
	case 0:
		fd.nameInput += ch
	case 1:
		fd.descriptionInput += ch
	case 2:
		fd.queryInput += ch
	case 3:
		fd.tagsInput += ch
	}
}

func (fd *FavoritesDialog) deleteChar() {
	switch fd.currentField {
	case 0:
		if len(fd.nameInput) > 0 {
			fd.nameInput = fd.nameInput[:len(fd.nameInput)-1]
		}
	case 1:
		if len(fd.descriptionInput) > 0 {
			fd.descriptionInput = fd.descriptionInput[:len(fd.descriptionInput)-1]
		}
	case 2:
		if len(fd.queryInput) > 0 {
			fd.queryInput = fd.queryInput[:len(fd.queryInput)-1]
		}
	case 3:
		if len(fd.tagsInput) > 0 {
			fd.tagsInput = fd.tagsInput[:len(fd.tagsInput)-1]
		}
	}
}

// View renders the dialog
func (fd *FavoritesDialog) View() string {
	switch fd.mode {
	case FavoritesModeList:
		return fd.renderList()
	case FavoritesModeAdd, FavoritesModeEdit:
		return fd.renderEdit()
	}
	return ""
}

func (fd *FavoritesDialog) renderList() string {
	var sections []string

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(fd.Theme.Foreground).
		Background(fd.Theme.Info).
		Padding(0, 1).
		Bold(true)
	sections = append(sections, titleStyle.Render("Favorite Queries"))

	// Instructions
	instrStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		Padding(0, 1)
	sections = append(sections, instrStyle.Render("↑↓: Navigate  Enter: Execute  a: Add  e: Edit  d: Delete  Esc: Close"))

	// Favorites list
	if len(fd.favorites) == 0 {
		sections = append(sections, "\nNo favorites yet. Press 'a' to add one.")
	} else {
		sections = append(sections, "")
		visibleStart := fd.offset
		visibleEnd := fd.offset + fd.Height - 10
		if visibleEnd > len(fd.favorites) {
			visibleEnd = len(fd.favorites)
		}

		for i := visibleStart; i < visibleEnd; i++ {
			fav := fd.favorites[i]

			// Format favorite entry
			name := fav.Name
			if len(name) > 40 {
				name = name[:37] + "..."
			}

			desc := fav.Description
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}

			line := fmt.Sprintf("%s\n  %s", name, desc)
			if len(fav.Tags) > 0 {
				line += fmt.Sprintf(" [%s]", strings.Join(fav.Tags, ", "))
			}

			style := lipgloss.NewStyle().Padding(0, 1)
			if i == fd.selected {
				style = style.Background(fd.Theme.Selection).Foreground(fd.Theme.Foreground)
			}
			sections = append(sections, style.Render(line))
		}
	}

	// Container
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(fd.Theme.Border).
		Width(fd.Width).
		Height(fd.Height).
		Padding(1)

	return containerStyle.Render(strings.Join(sections, "\n"))
}

func (fd *FavoritesDialog) renderEdit() string {
	var sections []string

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(fd.Theme.Foreground).
		Background(fd.Theme.Info).
		Padding(0, 1).
		Bold(true)

	title := "Add Favorite"
	if fd.mode == FavoritesModeEdit {
		title = "Edit Favorite"
	}
	sections = append(sections, titleStyle.Render(title))

	// Instructions
	instrStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#a6adc8")).
		Padding(0, 1)
	sections = append(sections, instrStyle.Render("Tab: Next field  Enter: Save  Esc: Cancel"))

	// Fields
	sections = append(sections, "")
	sections = append(sections, fd.renderField("Name:", fd.nameInput, fd.currentField == 0))
	sections = append(sections, fd.renderField("Description:", fd.descriptionInput, fd.currentField == 1))
	sections = append(sections, fd.renderField("Query:", fd.queryInput, fd.currentField == 2))
	sections = append(sections, fd.renderField("Tags (comma separated):", fd.tagsInput, fd.currentField == 3))

	// Container
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(fd.Theme.Border).
		Width(fd.Width).
		Height(fd.Height).
		Padding(1)

	return containerStyle.Render(strings.Join(sections, "\n"))
}

func (fd *FavoritesDialog) renderField(label, value string, active bool) string {
	style := lipgloss.NewStyle().Padding(0, 1)
	if active {
		style = style.Background(fd.Theme.Selection).Foreground(fd.Theme.Foreground)
		value = value + "_"
	}
	return style.Render(fmt.Sprintf("%s %s", label, value))
}

// GetEditData returns the current edit data
func (fd *FavoritesDialog) GetEditData() (name, description, query string, tags []string) {
	name = fd.nameInput
	description = fd.descriptionInput
	query = fd.queryInput

	// Parse tags
	if fd.tagsInput != "" {
		parts := strings.Split(fd.tagsInput, ",")
		for _, part := range parts {
			tag := strings.TrimSpace(part)
			if tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	return
}

// Package ui contains the root Bubble Tea application: a navigation stack
// of Screens plus the persistent status bar, help overlay, and error banner.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hashkrish/jira-tui/internal/jiraclient"
	"github.com/hashkrish/jira-tui/internal/ui/components/errorbanner"
	"github.com/hashkrish/jira-tui/internal/ui/components/filters"
	"github.com/hashkrish/jira-tui/internal/ui/components/helpbar"
	"github.com/hashkrish/jira-tui/internal/ui/components/projectlist"
	"github.com/hashkrish/jira-tui/internal/ui/components/statusbar"
	"github.com/hashkrish/jira-tui/internal/ui/keys"
	"github.com/hashkrish/jira-tui/internal/ui/screen"
	"github.com/hashkrish/jira-tui/internal/ui/styles"
)

// App is the root tea.Model. It owns the navigation stack and chrome
// (status bar, help, error banner) common to every screen.
type App struct {
	client *jiraclient.Client
	stack  []screen.Screen
	width  int
	height int

	status statusbar.Model
	help   helpbar.Model
	errBar errorbanner.Model

	// showHelp displays the full keybinding reference as a centered popup
	// that replaces the screen while open, dismissed by any key, rather
	// than an expanded multi-line footer.
	showHelp bool

	quitting bool
}

// New builds the root app with an initial screen and the connected user's
// display name / host for the status bar. client is used for global
// navigation shortcuts (e.g. jumping to the project list) that aren't owned
// by any single screen.
func New(client *jiraclient.Client, initial screen.Screen, userDisplayName, host string) App {
	st := statusbar.New()
	st.UserDisplayName = userDisplayName
	st.Host = host
	st.Breadcrumb = initial.Title()

	return App{
		client: client,
		stack:  []screen.Screen{initial},
		status: st,
		help:   helpbar.New(),
		errBar: errorbanner.New(),
	}
}

func (a App) Init() tea.Cmd {
	return a.top().Init()
}

func (a App) top() screen.Screen {
	return a.stack[len(a.stack)-1]
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width, a.height = msg.Width, msg.Height
		var cmds []tea.Cmd
		var cmd tea.Cmd
		a.status, cmd = a.status.Update(msg)
		cmds = append(cmds, cmd)
		a.help, cmd = a.help.Update(msg)
		cmds = append(cmds, cmd)
		a.errBar, cmd = a.errBar.Update(msg)
		cmds = append(cmds, cmd)
		newTop, cmd := a.top().Update(msg)
		a.stack[len(a.stack)-1] = newTop
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)

	case tea.KeyMsg:
		if key.Matches(msg, keys.Global.Quit) {
			a.quitting = true
			return a, tea.Quit
		}
		if a.showHelp {
			// The popup is modal: any other key just closes it rather than
			// reaching the underlying screen.
			a.showHelp = false
			return a, nil
		}

		// Any key press dismisses a shown error, not just ones that fall
		// through to the underlying screen below — otherwise pressing one
		// of the global shortcuts (?, esc, P, F) left a stale error banner
		// on screen indefinitely, since each of those returns early.
		if a.errBar.Visible() {
			a.errBar = a.errBar.Show(nil)
		}

		switch {
		case key.Matches(msg, keys.Global.Help):
			a.showHelp = true
			return a, nil
		case key.Matches(msg, keys.Global.Back) && len(a.stack) > 1:
			a.stack = a.stack[:len(a.stack)-1]
			a.status.Breadcrumb = a.breadcrumbTrail()
			return a, nil
		case key.Matches(msg, keys.Global.Projects):
			return a, screen.Push(projectlist.New(a.client))
		case key.Matches(msg, keys.Global.Filters):
			return a, screen.Push(filters.New(a.client))
		}

		newTop, cmd := a.top().Update(msg)
		a.stack[len(a.stack)-1] = newTop
		return a, cmd

	case screen.PushMsg:
		// Screens are only sized via tea.WindowSizeMsg broadcasts from the
		// runtime, which only fire on an actual terminal resize. A freshly
		// pushed screen needs the current size fed to it once up front, or
		// any size-dependent component (table, list, viewport) renders at
		// zero width/height.
		sized, sizeCmd := msg.Screen.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
		a.stack = append(a.stack, sized)
		a.status.Breadcrumb = a.breadcrumbTrail()
		return a, tea.Batch(sizeCmd, sized.Init())

	case screen.PopMsg:
		if len(a.stack) > 1 {
			a.stack = a.stack[:len(a.stack)-1]
			a.status.Breadcrumb = a.breadcrumbTrail()
		}
		return a, nil

	case screen.ErrMsg:
		a.errBar = a.errBar.Show(msg.Err)
		a.status = a.status.StopLoading()
		return a, nil

	default:
		var cmds []tea.Cmd
		var cmd tea.Cmd
		a.status, cmd = a.status.Update(msg)
		cmds = append(cmds, cmd)
		newTop, cmd := a.top().Update(msg)
		a.stack[len(a.stack)-1] = newTop
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)
	}
}

func (a App) breadcrumbTrail() string {
	titles := make([]string, len(a.stack))
	for i, s := range a.stack {
		titles[i] = s.Title()
	}
	return strings.Join(titles, " › ")
}

func (a App) View() string {
	if a.quitting {
		return ""
	}
	if a.showHelp {
		return a.helpPopupView()
	}

	body := a.top().View()

	var bottom []string
	if a.errBar.Visible() {
		bottom = append(bottom, a.errBar.View())
	}
	bottom = append(bottom, a.status.View())
	bottom = append(bottom, a.help.View())

	return body + "\n" + strings.Join(bottom, "\n")
}

// helpPopupView renders the full keybinding reference as a bordered box
// centered in the terminal, replacing the rest of the screen while open.
func (a App) helpPopupView() string {
	groups := keys.Global.FullHelp()

	var lines []string
	lines = append(lines, styles.Title.Render("Keybindings"))
	for _, group := range groups {
		lines = append(lines, "")
		for _, b := range group {
			h := b.Help()
			lines = append(lines, fmt.Sprintf("%-12s %s", h.Key, h.Desc))
		}
	}
	lines = append(lines, "")
	lines = append(lines, styles.Faint.Render("press any key to close"))

	box := styles.Border.Padding(1, 3).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, box)
}

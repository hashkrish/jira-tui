// Package ui contains the root Bubble Tea application: a navigation stack
// of Screens plus the persistent status bar, help overlay, and error banner.
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/hashkrish/jira-tui/internal/jiraclient"
	"github.com/hashkrish/jira-tui/internal/ui/components/errorbanner"
	"github.com/hashkrish/jira-tui/internal/ui/components/filters"
	"github.com/hashkrish/jira-tui/internal/ui/components/helpbar"
	"github.com/hashkrish/jira-tui/internal/ui/components/issuedetail"
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

	// showGotoIssue displays a centered prompt (gotoIssueInput) for jumping
	// straight to an issue by key, bypassing JQL search entirely. It's
	// opened by the "gt" chord (see pendingG below), not a single key,
	// since "G" is reserved (vim-style) for a future "go to last" binding.
	showGotoIssue  bool
	gotoIssueInput textinput.Model

	// pendingG records that a lone "g" was just pressed and we're waiting
	// on the next keystroke to complete a "g"-prefixed chord (currently
	// only "gt"). A non-"t" keystroke cancels the chord and falls through
	// to normal handling instead of being swallowed.
	pendingG bool

	quitting bool
}

// New builds the root app with an initial screen and the connected Jira
// host for the status bar. client is used for global navigation shortcuts
// (e.g. jumping to the project list) that aren't owned by any single screen.
func New(client *jiraclient.Client, initial screen.Screen, host string) App {
	st := statusbar.New()
	st.Host = host
	st.Breadcrumb = initial.Title()

	gi := textinput.New()
	gi.Placeholder = "issue key (e.g. PROJ-123)"
	gi.Prompt = "Go to: "

	return App{
		client:         client,
		stack:          []screen.Screen{initial},
		status:         st,
		help:           helpbar.New(),
		errBar:         errorbanner.New(),
		gotoIssueInput: gi,
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

		if a.showGotoIssue {
			switch msg.String() {
			case "esc":
				a.showGotoIssue = false
				a.gotoIssueInput.Blur()
				return a, nil
			case "enter":
				issueKey := strings.TrimSpace(a.gotoIssueInput.Value())
				a.showGotoIssue = false
				a.gotoIssueInput.Blur()
				if issueKey == "" {
					return a, nil
				}
				return a, screen.Push(issuedetail.New(a.client, issueKey))
			}
			var cmd tea.Cmd
			a.gotoIssueInput, cmd = a.gotoIssueInput.Update(msg)
			return a, cmd
		}

		// "g" is a chord prefix (vim-style): "gt" opens the go-to-issue
		// prompt. A lone "g" is otherwise unbound, so swallow it here and
		// wait for the next keystroke; anything other than "t" cancels the
		// chord and that keystroke falls through to normal handling below.
		if a.pendingG {
			a.pendingG = false
			if msg.String() == "t" {
				a.gotoIssueInput.SetValue("")
				a.gotoIssueInput.Focus()
				a.showGotoIssue = true
				return a, textinput.Blink
			}
		} else if msg.String() == "g" {
			a.pendingG = true
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
			a.status.Breadcrumb = a.top().Title()
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
		a.status.Breadcrumb = a.top().Title()
		return a, tea.Batch(sizeCmd, sized.Init())

	case screen.PopMsg:
		if len(a.stack) > 1 {
			a.stack = a.stack[:len(a.stack)-1]
			a.status.Breadcrumb = a.top().Title()
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
		if a.showGotoIssue {
			a.gotoIssueInput, cmd = a.gotoIssueInput.Update(msg)
			cmds = append(cmds, cmd)
		}
		newTop, cmd := a.top().Update(msg)
		a.stack[len(a.stack)-1] = newTop
		cmds = append(cmds, cmd)
		return a, tea.Batch(cmds...)
	}
}

func (a App) View() string {
	if a.quitting {
		return ""
	}
	if a.showHelp {
		return a.helpPopupView()
	}
	if a.showGotoIssue {
		return a.gotoIssuePopupView()
	}

	body := a.top().View()

	st := a.status
	if p, ok := a.top().(interface{ PageInfo() string }); ok {
		st.PageInfo = p.PageInfo()
	} else {
		st.PageInfo = ""
	}

	var bottom []string
	if a.errBar.Visible() {
		bottom = append(bottom, a.errBar.View())
	}
	bottom = append(bottom, st.View())
	bottom = append(bottom, a.help.View())
	footer := strings.Join(bottom, "\n")

	// Pin the footer to the terminal's actual last row(s), the way htop,
	// k9s, lazygit, etc. do: any leftover space (when a screen's content is
	// shorter than the terminal) shows as empty rows in the content area
	// above the footer, rather than as blank rows below it before whatever
	// comes after our program (e.g. tmux's status line). There's no way to
	// make the leftover space disappear entirely — it has to render as
	// blank somewhere when content < terminal height — so this is a
	// deliberate choice of where, not an attempt to eliminate it.
	if a.height > 0 {
		contentLines := strings.Count(body, "\n") + 1
		footerLines := strings.Count(footer, "\n") + 1
		if pad := a.height - contentLines - footerLines; pad > 0 {
			body += strings.Repeat("\n", pad)
		}
	}

	return body + "\n" + footer
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

// gotoIssuePopupView renders the "go to issue key" prompt as a centered
// box, the same way helpPopupView does, while the input is active.
func (a App) gotoIssuePopupView() string {
	lines := []string{
		styles.Title.Render("Go to issue"),
		"",
		a.gotoIssueInput.View(),
		"",
		styles.Faint.Render("enter to go, esc to cancel"),
	}

	box := styles.Border.Padding(1, 3).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(a.width, a.height, lipgloss.Center, lipgloss.Center, box)
}

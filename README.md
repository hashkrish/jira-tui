# jira-tui

Terminal UI client for Jira. Read-only in this phase: browse issues, run JQL
searches, view issue detail (description, comments, worklog, history),
browse projects/boards, and view the current sprint as a kanban board.

## Setup

1. Generate an API token at https://id.atlassian.com/manage-profile/security/api-tokens
2. Provide credentials via environment variables:

```sh
export JIRA_TUI_BASE_URL="https://yourcompany.atlassian.net"
export JIRA_TUI_EMAIL="you@example.com"
export JIRA_TUI_API_TOKEN="<api token>"
```

   (For Jira Data Center with a Personal Access Token, set `JIRA_TUI_AUTH_MODE=bearer`
   and omit `JIRA_TUI_EMAIL`.)

3. Verify connectivity:

```sh
go run ./cmd/jira-tui auth test
```

4. Launch the TUI:

```sh
go run ./cmd/jira-tui
```

## Keybindings

| Key | Action |
|---|---|
| `↑`/`k`, `↓`/`j` | move selection |
| `enter` | open selected item |
| `esc` / `backspace` | back |
| `/` | edit JQL (issue list) |
| `r` | refresh current screen |
| `n` / `p` | next / previous page (issue list) |
| `tab` / `shift+tab` | cycle tabs (issue detail) |
| `h`/`l`, `←`/`→` | switch column (sprint board) |
| `P` | jump to project list |
| `F` | jump to favorite filters |
| `gt` | go to issue by key |
| `?` | show full keybinding reference (popup, closed by any key) |
| `q` / `ctrl+c` | quit |

The footer always stays a single line; `?` opens the full reference as a
centered popup instead of expanding the footer.

Note: Jira's JQL search API rejects unbounded queries (no filter clause), so
the default view scopes to `assignee = currentUser()`. Press `/` to run any
other JQL.

## Status

Read-only feature set complete (M0–M7 of the project plan in
`docs/ARCHITECTURE.md`): auth, JQL search with pagination, issue detail with
ADF-rendered description/comments/worklog/history, projects → boards →
sprint kanban board, and favorite filters. Creating/editing issues,
transitions, and other mutating operations are out of scope for this phase.

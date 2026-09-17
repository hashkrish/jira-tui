# jira-tui — Read-Only TUI Client (Go)

## 1. Goal & Scope (Phase 1)

Build a terminal UI for browsing Jira — search, view issues, comments, worklogs, boards/sprints, projects — with **no mutating operations** yet (no create/edit/transition/comment-post). Auth via **Jira API token** (email + token, Basic Auth against Jira Cloud REST API v3, or PAT Bearer for Jira Data Center).

## 2. Tech Stack

| Concern | Choice | Why |
|---|---|---|
| Language | Go | |
| TUI framework | Bubble Tea (Elm architecture) | Best-in-class Go TUI, composable |
| Widgets | Bubbles (list, table, viewport, spinner, textinput, paginator, help) | |
| Styling | Lipgloss | |
| Markdown/ADF rendering | Glamour + custom ADF→Markdown converter | Jira Cloud stores descriptions/comments as Atlassian Document Format (ADF) JSON |
| HTTP client | stdlib net/http + small wrapper | avoid heavy deps |
| Config | hand-rolled TOML via BurntSushi/toml | store base URL, email, default project/JQL |
| Secrets | OS keyring via zalando/go-keyring, fallback to env var | never store token in plaintext if avoidable |
| CLI flags/entry | spf13/cobra | version/config subcommands |
| Logging | log/slog to file | TUI owns the terminal, no stdout logging |
| Testing | stdlib testing + httptest for API mocks; teatest for TUI flows | |

## 3. Authentication Design

- Jira Cloud: `Authorization: Basic base64(email:api_token)`.
- Jira Data Center/Server: `Authorization: Bearer <PAT>` (`auth_mode = bearer`).
- Config resolution order: env vars > OS keyring > config file (`~/.config/jira-tui/config.toml`).
- First-run flow: if no credentials found, launch a setup screen; validate via `/rest/api/3/myself`.
- `jira-tui auth test` subcommand for headless credential verification. (Implemented in M0.)

## 4. Package Layout

```
jira-tui/
  cmd/jira-tui/main.go
  internal/
    config/          # load/save config, keyring integration
    jiraclient/       # REST client: auth, http, rate-limit/backoff, pagination
    model/            # domain structs decoupled from raw API JSON
    cache/            # in-memory + optional on-disk TTL cache
    ui/
      app.go
      keys/
      styles/
      components/
        issuelist/
        issuedetail/
        search/
        boards/
        sprintboard/
        projectlist/
        filters/
        statusbar/
        helpbar/
        errorbanner/
        placeholder/
      screen/           # Screen interface + navigation messages (leaf package, avoids import cycles)
  docs/ARCHITECTURE.md
```

## 5. Data Model (internal/model)

`Issue` (with `IssueRef`, `IssueLink`, `HistoryEntry`/`FieldChange` for changelog), `Comment`, `Worklog`, `Project`, `Board`, `Sprint`, `Filter`.

## 6. Screens / Components (all read-only)

1. Startup / Auth check (`auth test` CLI subcommand + real launch auth check; a first-run interactive setup wizard is not yet implemented — see §13)
2. Inline JQL search bar (toggled in the issue list, not a separate pushed screen)
3. Issue List (table)
4. Issue Detail (metadata + description + comments/worklog/history tabs)
5. Project List
6. Board Selector
7. Sprint/Kanban Board View
8. Favorite Filters list
9. Status bar
10. Help overlay
11. Error/toast banner

## 7. Navigation / Keymap

Vim-style: `j/k`/`↑↓`, `enter`, `esc`/`backspace` back, `/` search (JQL), `r` refresh, `n`/`p` page next/prev (issue list), `h/l`/`←→` switch column (sprint board), `tab`/`shift+tab` cycle detail tabs, `P` jump to project list (global), `F` jump to favorite filters (global), `?` help, `q`/`ctrl+c` quit. Navigation stack (`[]Screen`) in root model, common `Screen` interface (`Init/Update/View/Title`) defined in its own leaf package (`internal/ui/screen`) to avoid import cycles with the screen components that implement it.

## 8. Jira API Endpoints Used (read-only)

`GET /rest/api/3/myself`, `/search/jql`, `/issue/{key}?expand=changelog`, `/issue/{key}/comment`, `/issue/{key}/worklog`, `/project/search`, `/filter/favourite`, `/rest/agile/1.0/board`, `/board/{id}/sprint`, `/sprint/{id}/issue`.

Note: `/rest/api/3/search/jql` (not the classic `/rest/api/3/search`, which Atlassian removed — see [CHANGE-2046](https://developer.atlassian.com/changelog/#CHANGE-2046)) uses forward-only `nextPageToken` pagination with no total count or offset. `internal/ui/components/issuelist` keeps a stack of visited page tokens so "p" (previous page) can step back without the API supporting random access. The Agile API (`/rest/agile/1.0/...`) is unaffected and still uses `startAt`/`maxResults`.

## 9. Cross-Cutting Concerns

Rate limiting/backoff (429 + Retry-After), short-TTL caching for metadata, all network calls via `tea.Cmd`/`tea.Batch`, ADF→Markdown renderer with golden-file tests, distinguish auth/transient/not-found errors, responsive resize handling, config validation.

## 10. Testing Strategy

`jiraclient`: httptest fixtures. `adf`: golden-file tests. UI: `teatest` scripted sequences. `config`: round-trip + keyring fallback. Manual smoke test against real Jira instance per milestone.

## 11. Milestones

- **M0 — Scaffolding** ✅: go.mod, cobra entrypoint, config loader, keyring, `auth test` working.
- **M1 — API client core** ✅: search/issue/comment/worklog/project, with tests + fixtures.
- **M2 — ADF renderer** ✅: standalone, tested independent of UI.
- **M3 — Bubble Tea shell** ✅: root app, screen-stack navigation, status bar, help overlay, error banner.
- **M4 — Issue list + search** ✅: JQL bar, table, pagination, real API integration.
- **M5 — Issue detail** ✅: metadata, description render, comments/worklog/history tabs.
- **M6 — Projects/Boards/Sprints** ✅: project list, board selector, kanban sprint board view.
- **M7 — Polish** ✅: adaptive light/dark theming, saved/favorite filters screen (`F`), empty-state messages, tiny-terminal resize regression tests, finished README/keymap docs.

## 12. Out of Scope for Phase 1

Creating/editing issues, posting comments, transitions/workflow execution, assigning users, attachments upload, webhooks, notifications, offline mode.

## 13. Known Gaps / Deferred

- No interactive first-run credential setup screen — currently config must be set via env vars or a hand-edited `~/.config/jira-tui/config.toml` (see README). `cfg.Validate()` gives a clear error either way.
- Rate-limit/backoff on 429 is implemented per-request in `jiraclient` but not surfaced to the user as a distinct status (it just retries silently once).
- `internal/cache` (short-TTL metadata caching) from the original plan was not needed in practice — every screen's own loaded state already avoids redundant refetches within a session — and was not built to avoid unused complexity.

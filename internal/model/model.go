// Package model holds domain types decoupled from raw Jira API JSON shapes.
package model

import "encoding/json"

// Issue is a flattened, UI-friendly view of a Jira issue.
type Issue struct {
	Key            string
	Summary        string
	Status         string
	StatusCategory string
	IssueType      string
	Priority       string
	Assignee       string
	AssigneeEmail  string
	Reporter       string
	Labels         []string
	Components     []string
	FixVersions    []string
	// Description is the raw Atlassian Document Format (ADF) JSON node.
	// Rendering to Markdown happens in internal/jiraclient/adf (see M2).
	Description json.RawMessage
	Created     string
	Updated     string
	DueDate     string
	ParentKey   string
	Subtasks    []IssueRef
	Links       []IssueLink
	// Changelog is ordered newest-first.
	Changelog []HistoryEntry
}

// HistoryEntry is one changelog entry (one or more field changes made
// together in a single edit).
type HistoryEntry struct {
	Author  string
	Created string
	Changes []FieldChange
}

// FieldChange describes a single field's before/after value within a
// HistoryEntry.
type FieldChange struct {
	Field string
	From  string
	To    string
}

// IssueRef is a lightweight pointer to another issue (subtask, parent).
type IssueRef struct {
	Key     string
	Summary string
	Status  string
}

// IssueLink describes a relationship to another issue (e.g. "blocks", "relates to").
type IssueLink struct {
	Type      string // e.g. "Blocks"
	Direction string // "inward" or "outward"
	Key       string
	Summary   string
}

// SearchResult is a page of JQL search results. The /rest/api/3/search/jql
// endpoint (the replacement for the removed /rest/api/3/search) uses
// forward-only token pagination: there is no stable total count or
// random-access offset. NextPageToken is empty on the last page.
type SearchResult struct {
	Issues        []Issue
	NextPageToken string
}

// Comment is a single issue comment.
type Comment struct {
	Author  string
	Body    json.RawMessage // ADF
	Created string
	Updated string
}

// Worklog is a single work log entry on an issue.
type Worklog struct {
	Author    string
	TimeSpent string
	Started   string
	Comment   json.RawMessage // ADF, may be empty
}

// Project is a Jira project summary.
type Project struct {
	Key  string
	Name string
	Lead string
}

// Filter is a saved/favorite JQL search.
type Filter struct {
	ID   string
	Name string
	JQL  string
}

// Board is an Agile board (Scrum or Kanban).
type Board struct {
	ID   int
	Name string
	Type string
}

// Sprint is a sprint on a Scrum board.
type Sprint struct {
	ID        int
	Name      string
	State     string // "active", "future", "closed"
	StartDate string
	EndDate   string
}

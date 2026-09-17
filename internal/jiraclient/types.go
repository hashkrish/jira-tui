package jiraclient

import (
	"encoding/json"

	"github.com/hashkrish/jira-tui/internal/model"
)

// Raw JSON shapes returned by the Jira REST API. These are intentionally
// permissive (most fields optional/nullable) and mapped into the clean
// internal/model types before being handed to the rest of the app.

type rawNamed struct {
	Name string `json:"name"`
}

type rawUser struct {
	DisplayName  string `json:"displayName"`
	EmailAddress string `json:"emailAddress"`
}

type rawIssueRef struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string   `json:"summary"`
		Status  rawNamed `json:"status"`
	} `json:"fields"`
}

type rawIssueLink struct {
	Type struct {
		Name    string `json:"name"`
		Inward  string `json:"inward"`
		Outward string `json:"outward"`
	} `json:"type"`
	InwardIssue  *rawIssueRef `json:"inwardIssue"`
	OutwardIssue *rawIssueRef `json:"outwardIssue"`
}

type rawFields struct {
	Summary string `json:"summary"`
	Status  struct {
		Name           string `json:"name"`
		StatusCategory struct {
			Name string `json:"name"`
		} `json:"statusCategory"`
	} `json:"status"`
	IssueType   rawNamed        `json:"issuetype"`
	Priority    rawNamed        `json:"priority"`
	Assignee    *rawUser        `json:"assignee"`
	Reporter    *rawUser        `json:"reporter"`
	Labels      []string        `json:"labels"`
	Components  []rawNamed      `json:"components"`
	FixVersions []rawNamed      `json:"fixVersions"`
	Description json.RawMessage `json:"description"`
	Created     string          `json:"created"`
	Updated     string          `json:"updated"`
	DueDate     string          `json:"duedate"`
	Parent      *rawIssueRef    `json:"parent"`
	Subtasks    []rawIssueRef   `json:"subtasks"`
	IssueLinks  []rawIssueLink  `json:"issuelinks"`
}

type rawHistoryItem struct {
	Field      string `json:"field"`
	FromString string `json:"fromString"`
	ToString   string `json:"toString"`
}

type rawHistory struct {
	Author  rawUser          `json:"author"`
	Created string           `json:"created"`
	Items   []rawHistoryItem `json:"items"`
}

type rawChangelog struct {
	Histories []rawHistory `json:"histories"`
}

type rawIssue struct {
	Key       string        `json:"key"`
	Fields    rawFields     `json:"fields"`
	Changelog *rawChangelog `json:"changelog"`
}

func namesOf(named []rawNamed) []string {
	if named == nil {
		return nil
	}
	out := make([]string, 0, len(named))
	for _, n := range named {
		out = append(out, n.Name)
	}
	return out
}

func displayName(u *rawUser) string {
	if u == nil {
		return ""
	}
	return u.DisplayName
}

func emailOf(u *rawUser) string {
	if u == nil {
		return ""
	}
	return u.EmailAddress
}

func toModelIssue(raw rawIssue) model.Issue {
	issue := model.Issue{
		Key:            raw.Key,
		Summary:        raw.Fields.Summary,
		Status:         raw.Fields.Status.Name,
		StatusCategory: raw.Fields.Status.StatusCategory.Name,
		IssueType:      raw.Fields.IssueType.Name,
		Priority:       raw.Fields.Priority.Name,
		Assignee:       displayName(raw.Fields.Assignee),
		AssigneeEmail:  emailOf(raw.Fields.Assignee),
		Reporter:       displayName(raw.Fields.Reporter),
		Labels:         raw.Fields.Labels,
		Components:     namesOf(raw.Fields.Components),
		FixVersions:    namesOf(raw.Fields.FixVersions),
		Description:    raw.Fields.Description,
		Created:        raw.Fields.Created,
		Updated:        raw.Fields.Updated,
		DueDate:        raw.Fields.DueDate,
	}

	if raw.Fields.Parent != nil {
		issue.ParentKey = raw.Fields.Parent.Key
	}

	for _, st := range raw.Fields.Subtasks {
		issue.Subtasks = append(issue.Subtasks, model.IssueRef{
			Key:     st.Key,
			Summary: st.Fields.Summary,
			Status:  st.Fields.Status.Name,
		})
	}

	for _, link := range raw.Fields.IssueLinks {
		if link.OutwardIssue != nil {
			issue.Links = append(issue.Links, model.IssueLink{
				Type:      link.Type.Outward,
				Direction: "outward",
				Key:       link.OutwardIssue.Key,
				Summary:   link.OutwardIssue.Fields.Summary,
			})
		}
		if link.InwardIssue != nil {
			issue.Links = append(issue.Links, model.IssueLink{
				Type:      link.Type.Inward,
				Direction: "inward",
				Key:       link.InwardIssue.Key,
				Summary:   link.InwardIssue.Fields.Summary,
			})
		}
	}

	if raw.Changelog != nil {
		// Jira returns histories oldest-first; present newest-first.
		for i := len(raw.Changelog.Histories) - 1; i >= 0; i-- {
			h := raw.Changelog.Histories[i]
			entry := model.HistoryEntry{
				Author:  h.Author.DisplayName,
				Created: h.Created,
			}
			for _, item := range h.Items {
				entry.Changes = append(entry.Changes, model.FieldChange{
					Field: item.Field,
					From:  item.FromString,
					To:    item.ToString,
				})
			}
			issue.Changelog = append(issue.Changelog, entry)
		}
	}

	return issue
}

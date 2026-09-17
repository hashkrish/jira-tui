package jiraclient

import (
	"fmt"

	"github.com/krishnan/jira-tui/internal/model"
)

// GetIssue fetches full detail for a single issue by key (e.g. "PROJ-123"),
// including its changelog history.
func (c *Client) GetIssue(key string) (*model.Issue, error) {
	var raw rawIssue
	if err := c.get(fmt.Sprintf("/rest/api/3/issue/%s?expand=changelog", key), &raw); err != nil {
		return nil, err
	}
	issue := toModelIssue(raw)
	return &issue, nil
}

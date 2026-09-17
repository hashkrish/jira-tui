package jiraclient

import "github.com/hashkrish/jira-tui/internal/model"

type rawFilter struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	JQL  string `json:"jql"`
}

// ListFavoriteFilters returns the authenticated user's favorite/saved
// filters. Unlike most Jira Cloud list endpoints this one returns a bare
// JSON array rather than a paginated {values: [...]} envelope.
func (c *Client) ListFavoriteFilters() ([]model.Filter, error) {
	var raw []rawFilter
	if err := c.get("/rest/api/3/filter/favourite", &raw); err != nil {
		return nil, err
	}

	filters := make([]model.Filter, 0, len(raw))
	for _, rf := range raw {
		filters = append(filters, model.Filter{ID: rf.ID, Name: rf.Name, JQL: rf.JQL})
	}
	return filters, nil
}

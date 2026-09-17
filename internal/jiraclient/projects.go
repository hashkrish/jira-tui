package jiraclient

import (
	"net/url"
	"strconv"

	"github.com/hashkrish/jira-tui/internal/model"
)

type rawProject struct {
	Key  string  `json:"key"`
	Name string  `json:"name"`
	Lead rawUser `json:"lead"`
}

type rawProjectSearchResult struct {
	IsLast bool         `json:"isLast"`
	Values []rawProject `json:"values"`
}

// ListProjects returns all projects visible to the authenticated user,
// transparently paging through the search endpoint.
func (c *Client) ListProjects() ([]model.Project, error) {
	var all []model.Project
	startAt := 0
	const pageSize = 50

	for {
		params := url.Values{}
		params.Set("startAt", strconv.Itoa(startAt))
		params.Set("maxResults", strconv.Itoa(pageSize))

		var raw rawProjectSearchResult
		if err := c.getQuery("/rest/api/3/project/search", params, &raw); err != nil {
			return nil, err
		}

		for _, rp := range raw.Values {
			all = append(all, model.Project{
				Key:  rp.Key,
				Name: rp.Name,
				Lead: rp.Lead.DisplayName,
			})
		}

		if raw.IsLast || len(raw.Values) == 0 {
			break
		}
		startAt += len(raw.Values)
	}

	return all, nil
}

package jiraclient

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/krishnan/jira-tui/internal/model"
)

type rawSprint struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	State     string `json:"state"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

type rawSprintPage struct {
	IsLast bool        `json:"isLast"`
	Values []rawSprint `json:"values"`
}

// ListSprints returns sprints for a board, newest-active-first is not
// guaranteed by the API; callers should filter by State as needed.
func (c *Client) ListSprints(boardID int) ([]model.Sprint, error) {
	var all []model.Sprint
	startAt := 0
	const pageSize = 50

	for {
		params := url.Values{}
		params.Set("startAt", strconv.Itoa(startAt))
		params.Set("maxResults", strconv.Itoa(pageSize))

		var raw rawSprintPage
		path := fmt.Sprintf("/rest/agile/1.0/board/%d/sprint", boardID)
		if err := c.getQuery(path, params, &raw); err != nil {
			return nil, err
		}

		for _, rs := range raw.Values {
			all = append(all, model.Sprint{
				ID:        rs.ID,
				Name:      rs.Name,
				State:     rs.State,
				StartDate: rs.StartDate,
				EndDate:   rs.EndDate,
			})
		}

		if raw.IsLast || len(raw.Values) == 0 {
			break
		}
		startAt += len(raw.Values)
	}

	return all, nil
}

type rawSprintIssuePage struct {
	StartAt    int        `json:"startAt"`
	MaxResults int        `json:"maxResults"`
	Total      int        `json:"total"`
	Issues     []rawIssue `json:"issues"`
}

// GetSprintIssues returns one page of issues on a sprint.
func (c *Client) GetSprintIssues(sprintID, startAt, maxResults int) (*model.SearchResult, error) {
	params := url.Values{}
	params.Set("startAt", strconv.Itoa(startAt))
	params.Set("maxResults", strconv.Itoa(maxResults))
	params.Set("fields", strings.Join(DefaultSearchFields, ","))

	var raw rawSprintIssuePage
	path := fmt.Sprintf("/rest/agile/1.0/sprint/%d/issue", sprintID)
	if err := c.getQuery(path, params, &raw); err != nil {
		return nil, err
	}

	result := &model.SearchResult{}
	for _, ri := range raw.Issues {
		result.Issues = append(result.Issues, toModelIssue(ri))
	}
	return result, nil
}

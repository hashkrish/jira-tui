package jiraclient

import (
	"net/url"
	"strconv"

	"github.com/hashkrish/jira-tui/internal/model"
)

type rawBoard struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type rawBoardPage struct {
	IsLast bool       `json:"isLast"`
	Values []rawBoard `json:"values"`
}

// ListBoards returns all Agile boards, optionally filtered to a single
// project (pass "" for all boards visible to the user).
func (c *Client) ListBoards(projectKey string) ([]model.Board, error) {
	var all []model.Board
	startAt := 0
	const pageSize = 50

	for {
		params := url.Values{}
		params.Set("startAt", strconv.Itoa(startAt))
		params.Set("maxResults", strconv.Itoa(pageSize))
		if projectKey != "" {
			params.Set("projectKeyOrId", projectKey)
		}

		var raw rawBoardPage
		if err := c.getQuery("/rest/agile/1.0/board", params, &raw); err != nil {
			return nil, err
		}

		for _, rb := range raw.Values {
			all = append(all, model.Board{ID: rb.ID, Name: rb.Name, Type: rb.Type})
		}

		if raw.IsLast || len(raw.Values) == 0 {
			break
		}
		startAt += len(raw.Values)
	}

	return all, nil
}

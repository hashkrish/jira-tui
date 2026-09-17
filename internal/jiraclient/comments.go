package jiraclient

import (
	"encoding/json"
	"fmt"

	"github.com/krishnan/jira-tui/internal/model"
)

type rawComment struct {
	Author  rawUser         `json:"author"`
	Body    json.RawMessage `json:"body"`
	Created string          `json:"created"`
	Updated string          `json:"updated"`
}

type rawCommentPage struct {
	Comments []rawComment `json:"comments"`
}

// GetComments returns all comments on an issue (single page; Jira's default
// maxResults for this endpoint is generous enough for typical issues — pagination
// can be added if needed).
func (c *Client) GetComments(key string) ([]model.Comment, error) {
	var raw rawCommentPage
	if err := c.get(fmt.Sprintf("/rest/api/3/issue/%s/comment", key), &raw); err != nil {
		return nil, err
	}

	comments := make([]model.Comment, 0, len(raw.Comments))
	for _, rc := range raw.Comments {
		comments = append(comments, model.Comment{
			Author:  rc.Author.DisplayName,
			Body:    rc.Body,
			Created: rc.Created,
			Updated: rc.Updated,
		})
	}
	return comments, nil
}

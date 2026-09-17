package jiraclient

import (
	"encoding/json"
	"fmt"

	"github.com/krishnan/jira-tui/internal/model"
)

type rawWorklog struct {
	Author    rawUser         `json:"author"`
	TimeSpent string          `json:"timeSpent"`
	Started   string          `json:"started"`
	Comment   json.RawMessage `json:"comment"`
}

type rawWorklogPage struct {
	Worklogs []rawWorklog `json:"worklogs"`
}

// GetWorklogs returns all work log entries on an issue.
func (c *Client) GetWorklogs(key string) ([]model.Worklog, error) {
	var raw rawWorklogPage
	if err := c.get(fmt.Sprintf("/rest/api/3/issue/%s/worklog", key), &raw); err != nil {
		return nil, err
	}

	logs := make([]model.Worklog, 0, len(raw.Worklogs))
	for _, rw := range raw.Worklogs {
		logs = append(logs, model.Worklog{
			Author:    rw.Author.DisplayName,
			TimeSpent: rw.TimeSpent,
			Started:   rw.Started,
			Comment:   rw.Comment,
		})
	}
	return logs, nil
}

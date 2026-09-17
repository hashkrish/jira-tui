package jiraclient

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/hashkrish/jira-tui/internal/model"
)

// DefaultSearchFields limits the payload size of search responses to only
// what the issue list / detail views need.
var DefaultSearchFields = []string{
	"summary", "status", "issuetype", "priority", "assignee", "reporter",
	"labels", "components", "fixVersions", "created", "updated", "duedate",
	"parent", "subtasks", "issuelinks",
}

type rawSearchResult struct {
	Issues        []rawIssue `json:"issues"`
	NextPageToken string     `json:"nextPageToken"`
}

// SearchIssues runs a JQL query and returns one page of results using the
// /rest/api/3/search/jql endpoint. The classic /rest/api/3/search endpoint
// (offset-based startAt/total pagination) was removed by Atlassian in 2025
// (see https://developer.atlassian.com/changelog/#CHANGE-2046); its
// replacement uses forward-only token pagination instead, so there is no
// stable "total" count or random-access page offset — pass the token from
// the previous SearchResult.NextPageToken to fetch the next page, and ""
// for the first page.
func (c *Client) SearchIssues(jql, pageToken string, maxResults int, fields []string) (*model.SearchResult, error) {
	if fields == nil {
		fields = DefaultSearchFields
	}

	params := url.Values{}
	params.Set("jql", jql)
	params.Set("maxResults", strconv.Itoa(maxResults))
	params.Set("fields", strings.Join(fields, ","))
	if pageToken != "" {
		params.Set("nextPageToken", pageToken)
	}

	var raw rawSearchResult
	if err := c.getQuery("/rest/api/3/search/jql", params, &raw); err != nil {
		return nil, err
	}

	result := &model.SearchResult{NextPageToken: raw.NextPageToken}
	for _, ri := range raw.Issues {
		result.Issues = append(result.Issues, toModelIssue(ri))
	}
	return result, nil
}

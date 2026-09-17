// Package adf converts Atlassian Document Format (ADF) JSON — used by Jira
// Cloud for issue descriptions, comments, and worklog comments — into
// Markdown suitable for rendering with Glamour.
package adf

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type node struct {
	Type    string         `json:"type"`
	Text    string         `json:"text"`
	Attrs   map[string]any `json:"attrs"`
	Marks   []mark         `json:"marks"`
	Content []node         `json:"content"`
}

type mark struct {
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs"`
}

// Render converts a raw ADF document (or empty/nil input) into Markdown.
func Render(raw json.RawMessage) (string, error) {
	if len(raw) == 0 {
		return "", nil
	}

	var doc node
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", fmt.Errorf("parsing ADF: %w", err)
	}

	blocks := make([]string, 0, len(doc.Content))
	for _, child := range doc.Content {
		if b := renderBlock(child, 0); b != "" {
			blocks = append(blocks, b)
		}
	}
	return strings.Join(blocks, "\n\n"), nil
}

func renderBlock(n node, depth int) string {
	switch n.Type {
	case "paragraph":
		return renderInline(n.Content)
	case "heading":
		level := intAttr(n.Attrs, "level", 1)
		return strings.Repeat("#", level) + " " + renderInline(n.Content)
	case "bulletList":
		return renderList(n, depth, false)
	case "orderedList":
		return renderList(n, depth, true)
	case "codeBlock":
		lang, _ := n.Attrs["language"].(string)
		return "```" + lang + "\n" + renderInline(n.Content) + "\n```"
	case "blockquote":
		inner := renderChildren(n.Content, depth)
		lines := strings.Split(inner, "\n")
		for i, l := range lines {
			lines[i] = "> " + l
		}
		return strings.Join(lines, "\n")
	case "panel":
		panelType, _ := n.Attrs["panelType"].(string)
		inner := renderChildren(n.Content, depth)
		lines := strings.Split(inner, "\n")
		for i, l := range lines {
			lines[i] = "> " + l
		}
		return fmt.Sprintf("> **%s**\n%s", strings.ToUpper(panelType), strings.Join(lines, "\n"))
	case "rule":
		return "---"
	case "table":
		return renderTable(n)
	case "mediaSingle", "mediaGroup":
		return "[attachment]"
	default:
		// Unknown block type: render children as best-effort fallback.
		return renderChildren(n.Content, depth)
	}
}

func renderChildren(children []node, depth int) string {
	parts := make([]string, 0, len(children))
	for _, c := range children {
		if b := renderBlock(c, depth); b != "" {
			parts = append(parts, b)
		}
	}
	return strings.Join(parts, "\n\n")
}

func renderList(n node, depth int, ordered bool) string {
	var lines []string
	for i, item := range n.Content {
		indent := strings.Repeat("  ", depth)
		marker := "-"
		if ordered {
			marker = strconv.Itoa(i+1) + "."
		}

		// A listItem's content is typically [paragraph, optional nested list...].
		var text string
		var nested []string
		for _, c := range item.Content {
			switch c.Type {
			case "bulletList":
				nested = append(nested, renderList(c, depth+1, false))
			case "orderedList":
				nested = append(nested, renderList(c, depth+1, true))
			default:
				if s := renderBlock(c, depth+1); s != "" {
					if text == "" {
						text = s
					} else {
						nested = append(nested, s)
					}
				}
			}
		}

		lines = append(lines, indent+marker+" "+text)
		lines = append(lines, nested...)
	}
	return strings.Join(lines, "\n")
}

func renderTable(n node) string {
	var rows [][]string
	for _, row := range n.Content {
		if row.Type != "tableRow" {
			continue
		}
		var cells []string
		for _, cell := range row.Content {
			cells = append(cells, renderChildren(cell.Content, 0))
		}
		rows = append(rows, cells)
	}
	if len(rows) == 0 {
		return ""
	}

	var b strings.Builder
	for i, row := range rows {
		b.WriteString("| " + strings.Join(row, " | ") + " |\n")
		if i == 0 {
			sep := make([]string, len(row))
			for j := range sep {
				sep[j] = "---"
			}
			b.WriteString("| " + strings.Join(sep, " | ") + " |\n")
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderInline(nodes []node) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n.Type {
		case "text":
			b.WriteString(applyMarks(n.Text, n.Marks))
		case "hardBreak":
			b.WriteString("  \n")
		case "mention":
			text, _ := n.Attrs["text"].(string)
			if text == "" {
				text, _ = n.Attrs["id"].(string)
			}
			b.WriteString("@" + strings.TrimPrefix(text, "@"))
		case "emoji":
			short, _ := n.Attrs["shortName"].(string)
			b.WriteString(short)
		case "inlineCard":
			url, _ := n.Attrs["url"].(string)
			b.WriteString(fmt.Sprintf("[%s](%s)", url, url))
		default:
			b.WriteString(renderInline(n.Content))
		}
	}
	return b.String()
}

func applyMarks(text string, marks []mark) string {
	var link string
	for _, m := range marks {
		switch m.Type {
		case "strong":
			text = "**" + text + "**"
		case "em":
			text = "_" + text + "_"
		case "code":
			text = "`" + text + "`"
		case "strike":
			text = "~~" + text + "~~"
		case "link":
			if href, ok := m.Attrs["href"].(string); ok {
				link = href
			}
		}
	}
	if link != "" {
		return fmt.Sprintf("[%s](%s)", text, link)
	}
	return text
}

func intAttr(attrs map[string]any, key string, def int) int {
	v, ok := attrs[key]
	if !ok {
		return def
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return def
	}
}

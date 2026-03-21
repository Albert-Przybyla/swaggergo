package gin

import (
	"go/ast"
	"strings"
)

func parseCommentMetadata(doc *ast.CommentGroup) (string, string) {
	if doc == nil {
		return "", ""
	}

	var summary string
	var descriptionLines []string
	var plainLines []string

	for _, line := range commentLines(doc) {
		switch {
		case strings.HasPrefix(line, "@Title "):
			summary = strings.TrimSpace(strings.TrimPrefix(line, "@Title "))
		case strings.HasPrefix(line, "@Summary "):
			summary = strings.TrimSpace(strings.TrimPrefix(line, "@Summary "))
		case strings.HasPrefix(line, "@Description "):
			descriptionLines = append(descriptionLines, strings.TrimSpace(strings.TrimPrefix(line, "@Description ")))
		default:
			plainLines = append(plainLines, line)
		}
	}

	if summary == "" && len(plainLines) > 0 {
		summary = plainLines[0]
		plainLines = plainLines[1:]
	}

	if len(descriptionLines) == 0 && len(plainLines) > 0 {
		descriptionLines = plainLines
	}

	return summary, strings.Join(descriptionLines, "\n")
}

func parseDescription(doc *ast.CommentGroup) string {
	summary, description := parseCommentMetadata(doc)
	if description != "" {
		return description
	}
	return summary
}

func commentLines(doc *ast.CommentGroup) []string {
	if doc == nil {
		return nil
	}

	var lines []string
	for _, comment := range doc.List {
		text := strings.TrimSpace(comment.Text)
		text = strings.TrimPrefix(text, "//")
		text = strings.TrimPrefix(text, "/*")
		text = strings.TrimSuffix(text, "*/")
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		lines = append(lines, text)
	}

	return lines
}

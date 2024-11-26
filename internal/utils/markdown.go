package utils

import (
	"fmt"
	"strings"

	"github.com/etkecc/go-kit/format"
)

func markdownURL(label, link string) string {
	label = strings.TrimSpace(label)
	link = strings.TrimSpace(link)

	return fmt.Sprintf("[%s](%s)", label, link)
}

// MarkdownEmail returns markdown link to email
func MarkdownEmail(email string) string {
	email = strings.TrimSpace(email)
	return markdownURL(email, "mailto:"+email)
}

// MarkdownMXID returns markdown link to MXID
func MarkdownMXID(mxid string) string {
	mxid = strings.TrimSpace(mxid)
	return markdownURL(mxid, "https://matrix.to/#/"+mxid)
}

// MarkdownLink returns markdown link
func MarkdownLink(link string) string {
	return markdownURL(link, link)
}

// MarkdownRender coverts markdown text into text and html forms
func MarkdownRender(mdtext string) (text, html string) {
	html = format.Render(mdtext)

	return mdtext, html
}

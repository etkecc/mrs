package utils

import (
	"fmt"
	"strings"

	"maunium.net/go/mautrix/format"
)

func markdownURL(label, link string) string {
	label = strings.TrimSpace(label)
	link = strings.TrimSpace(link)

	return fmt.Sprintf("[%s](%s)", label, link)
}

func MarkdownEmail(email string) string {
	email = strings.TrimSpace(email)
	return markdownURL(email, "mailto:"+email)
}

func MarkdownMXID(mxid string) string {
	mxid = strings.TrimSpace(mxid)
	return markdownURL(mxid, "https://matrix.to/#/"+mxid)
}

func MarkdownLink(link string) string {
	return markdownURL(link, link)
}

func MarkdownRender(mdtext string) (text, html string) {
	content := format.RenderMarkdown(mdtext, true, true)

	return content.Body, content.FormattedBody
}

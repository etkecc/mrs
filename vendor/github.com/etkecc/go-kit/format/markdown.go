// Package format converts Markdown to HTML using Goldmark, adapted from mautrix-go/format.
package format

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

const (
	pStart = "<p>"
	pEnd   = "</p>"
)

var (
	// Extensions enables Strikethrough (~~text~~) and Table rendering.
	Extensions = goldmark.WithExtensions(extension.Strikethrough, extension.Table)
	// RendererOptions enables HardWraps (newlines become <br>) and Unsafe (raw HTML passthrough).
	RendererOptions = goldmark.WithRendererOptions(html.WithHardWraps(), html.WithUnsafe())
	// ParserOptions registers LinksTransformer at priority 1000, ahead of other AST transformations.
	ParserOptions = goldmark.WithParserOptions(parser.WithASTTransformers(util.Prioritized(&LinksTransformer{}, 1000)))

	// Renderer is the package-level Goldmark renderer, safe for concurrent use; replace it to customize rendering.
	Renderer = goldmark.New(Extensions, RendererOptions, ParserOptions)
)

// Render converts Markdown to HTML, unwrapping outer <p> tags for single-paragraph content.
func Render(markdown string) (htmlString string) {
	var buf bytes.Buffer
	if err := Renderer.Convert([]byte(markdown), &buf); err != nil {
		return fmt.Sprintf("<p>Error rendering markdown: %s</p>", err)
	}

	htmlString = strings.TrimRight(buf.String(), "\n")
	if strings.HasPrefix(htmlString, pStart) && strings.HasSuffix(htmlString, pEnd) {
		htmlNoP := htmlString[len(pStart) : len(htmlString)-len(pEnd)]
		if !strings.Contains(htmlNoP, pStart) {
			return htmlNoP
		}
	}
	return htmlString
}

// LinksTransformer is a goldmark AST transformer that adds target="_blank" to every link node.
type LinksTransformer struct{}

// Transform adds target="_blank" to every link node so rendered links open in a new tab.
func (t *LinksTransformer) Transform(node *ast.Document, _ text.Reader, _ parser.Context) {
	//nolint:errcheck // interface doesn't return errors
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if v, ok := n.(*ast.Link); ok {
			v.SetAttributeString("target", "_blank")
		}

		return ast.WalkContinue, nil
	})
}

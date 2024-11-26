// package format (markdown) is a markdown to html converter, heavily inspired by the mautrix-go/format package
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
	// Extensions is the default set of extensions to use with Goldmark.
	Extensions = goldmark.WithExtensions(extension.Strikethrough, extension.Table)
	// RendererOptions is the default set of renderer options to use with Goldmark.
	RendererOptions = goldmark.WithRendererOptions(html.WithHardWraps(), html.WithUnsafe())
	// ParserOptions is the default set of parser options to use with Goldmark.
	ParserOptions = goldmark.WithParserOptions(parser.WithASTTransformers(util.Prioritized(&LinksTransformer{}, 1000)))

	// Renderer is the default Goldmark renderer.
	Renderer = goldmark.New(Extensions, RendererOptions, ParserOptions)
)

// Render renders the given markdown to HTML.
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

// LinksTransformer is a transformer that adds target="_blank" to all links.
type LinksTransformer struct{}

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

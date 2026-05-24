// Package format converts Markdown to HTML using Goldmark.
// It is inspired by the mautrix-go/format package and is intended for rendering
// user-facing content in Matrix-adjacent contexts.
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
	// It enables Strikethrough (~~text~~) and Table rendering.
	Extensions = goldmark.WithExtensions(extension.Strikethrough, extension.Table)
	// RendererOptions is the default set of renderer options to use with Goldmark.
	// It enables HardWraps (converts newlines to <br> tags) and Unsafe (allows raw HTML passthrough).
	RendererOptions = goldmark.WithRendererOptions(html.WithHardWraps(), html.WithUnsafe())
	// ParserOptions is the default set of parser options to use with Goldmark.
	// It registers the LinksTransformer at priority 1000 (higher priority = earlier in transformation chain)
	// to ensure all links are processed before other transformations.
	ParserOptions = goldmark.WithParserOptions(parser.WithASTTransformers(util.Prioritized(&LinksTransformer{}, 1000)))

	// Renderer is the default package-level Goldmark renderer singleton.
	// It is safe for concurrent use as Goldmark renderers are stateless after construction.
	// It can be replaced to customize rendering behavior for the entire package.
	Renderer = goldmark.New(Extensions, RendererOptions, ParserOptions)
)

// Render converts the given Markdown string to HTML.
// Goldmark wraps single-paragraph content in <p>...</p> tags by default.
// This function unwraps the outer <p> tags when the content is a single paragraph
// (i.e., contains no nested <p> elements), returning the inner HTML directly.
// If the content spans multiple paragraphs, the full HTML with all <p> tags is returned.
// On rendering error, it returns an HTML error paragraph describing the failure.
// Empty string input produces empty string output (no wrapper tags).
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

// LinksTransformer is a Goldmark AST transformer that adds target="_blank" to all link nodes.
// It implements goldmark's parser.ASTTransformer interface and walks the entire parsed AST,
// modifying each Link node to open in a new browser tab.
// It is registered in ParserOptions at priority 1000 and runs automatically during Markdown parsing.
type LinksTransformer struct{}

// Transform walks the parsed Markdown AST and adds target="_blank" to every link node.
// This ensures all links open in a new tab when rendered to HTML.
// The //nolint directive is preserved because the interface signature requires an error return value,
// but the ast.Walk callback intentionally never returns a non-nil error in this implementation.
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

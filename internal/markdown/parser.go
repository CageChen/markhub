// Package markdown provides Goldmark-based Markdown parsing with GFM extensions and syntax highlighting.
package markdown

import (
	"bytes"
	"regexp"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// TOCItem represents a table of contents entry
type TOCItem struct {
	Level  int    `json:"level"`
	Title  string `json:"title"`
	Anchor string `json:"anchor"`
}

// ParseResult contains the parsed markdown result
type ParseResult struct {
	HTML  string    `json:"html"`
	TOC   []TOCItem `json:"toc"`
	Title string    `json:"title"`
}

// Parser handles markdown parsing with goldmark
type Parser struct {
	md goldmark.Markdown
}

// NewParser creates a new markdown parser with extensions
func NewParser() *Parser {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			extension.Table,
			extension.Strikethrough,
			extension.TaskList,
			extension.Typographer,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(
					chromahtml.WithClasses(true),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			html.WithUnsafe(),
		),
	)

	return &Parser{md: md}
}

// Parse converts markdown source to HTML and extracts metadata
func (p *Parser) Parse(source []byte) (*ParseResult, error) {
	var buf bytes.Buffer
	if err := p.md.Convert(source, &buf); err != nil {
		return nil, err
	}

	toc := p.extractTOC(source)
	title := ""
	if len(toc) > 0 {
		title = toc[0].Title
	}

	return &ParseResult{
		HTML:  buf.String(),
		TOC:   toc,
		Title: title,
	}, nil
}

// extractTOC walks the AST to extract headings
func (p *Parser) extractTOC(source []byte) []TOCItem {
	reader := text.NewReader(source)
	doc := p.md.Parser().Parse(reader)

	var toc []TOCItem
	err := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if heading, ok := n.(*ast.Heading); ok {
			title := extractText(heading, source)
			anchor := generateAnchor(title)
			toc = append(toc, TOCItem{
				Level:  heading.Level,
				Title:  title,
				Anchor: anchor,
			})
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil
	}

	return toc
}

// extractText extracts text content from a node
func extractText(n ast.Node, source []byte) string {
	var buf bytes.Buffer
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		if text, ok := child.(*ast.Text); ok {
			buf.Write(text.Segment.Value(source))
		}
	}
	return buf.String()
}

// generateAnchor creates a URL-safe anchor from text
func generateAnchor(text string) string {
	// Convert to lowercase
	anchor := strings.ToLower(text)
	// Replace spaces with hyphens
	anchor = strings.ReplaceAll(anchor, " ", "-")
	// Remove non-alphanumeric characters except hyphens
	reg := regexp.MustCompile(`[^a-z0-9\-\p{Han}\p{Hiragana}\p{Katakana}]`)
	anchor = reg.ReplaceAllString(anchor, "")
	// Remove multiple consecutive hyphens
	reg = regexp.MustCompile(`-+`)
	anchor = reg.ReplaceAllString(anchor, "-")
	// Trim hyphens from start and end
	anchor = strings.Trim(anchor, "-")
	return anchor
}

package robots

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The lexer in this file is based on the model presented by Rob Pike.
// There are important amendments and subtractions, but the model is
// his work.
//
// https://www.youtube.com/watch?v=HxaD_trXwRE

type membertype int

const (
	itemError = membertype(iota)
	itemUserAgent
	itemDisallow
	itemAllow
	itemSitemap
)

const eof = -1

var membertypes = map[string]membertype{
	"user-agent": itemUserAgent,
	"disallow":   itemDisallow,
	"allow":      itemAllow,
	"sitemap":    itemSitemap,
}

type item struct {
	typ membertype
	val string
}

type lexer struct {
	typ   membertype
	input string
	start int
	pos   int
	width int
	items chan *item
}

func (l *lexer) nextItem() *item {
	return <-l.items
}

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, w := utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = w
	l.pos += w
	return r
}

func (l *lexer) backup() {
	l.pos -= l.width
}

func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func (l *lexer) emit() {
	l.items <- &item{
		typ: l.typ,
		val: strings.TrimRightFunc(l.input[l.start:l.pos], unicode.IsSpace),
	}
	l.start = l.pos
}

func (l *lexer) ignore() {
	l.start = l.pos
}

// Unlike model lexer, does not terminate lexing.  Per
// https://developers.google.com/search/reference/robots_txt#abstract,
// simply accept lines that are valid and silently discard those that
// are not (even if received content is HTML).
func (l *lexer) errorf(format string, args ...interface{}) {
	l.items <- &item{
		typ: itemError,
		val: fmt.Sprintf(format, args...),
	}
}

func (l *lexer) run() {
	for fn := lexStart; fn != nil; fn = fn(l) {
	}
	close(l.items)
}

func stripBOM(s string) string {
	if r, w := utf8.DecodeRuneInString(s); r == '\ufeff' {
		return s[w:]
	}
	return s
}

func lex(in string) []*item {
	l := &lexer{
		input: stripBOM(in),
		items: make(chan *item),
	}
	go l.run()
	items := []*item{}
	for item := l.nextItem(); item != nil; item = l.nextItem() {
		items = append(items, item)
	}
	return items
}

type lexfn func(*lexer) lexfn

func lexStart(l *lexer) lexfn {
	c := l.peek()
	switch {
	case c == eof:
		return nil
	case c == '#':
		return lexComment
	case unicode.IsSpace(c):
		skipLWS(l)
		return lexStart
	default:
		return lexField
	}
}

func lexField(l *lexer) lexfn {
	for field, typ := range membertypes {
		if len(l.input[l.start:]) < len(field) {
			// The remaining input is shorter than the
			// field specifier.
			continue
		}
		if strings.EqualFold(field, l.input[l.start:l.start+len(field)]) {
			l.typ = typ
			l.pos += len(field)
			l.ignore()
			return lexSep
		}
	}
	// The input did not match a field. We emit an error and continue.
	l.errorf("unexpected field type: %s", l.input[l.start:])
	return lexNextLine
}

// If we're here, the beginning of this line did not match a
// specifier, and therefore the rest of the line cannot match
// anything.
func lexNextLine(l *lexer) lexfn {
	for c := l.next(); c != '\n' && c != eof; c = l.next() {
	}
	l.ignore()
	return lexStart
}

// Check for a separator, optionally with LWS on both sides.
func lexSep(l *lexer) lexfn {
	skipLWS(l)
	if c := l.next(); c != ':' {
		l.errorf("expected separator betweeen field and value")
		return lexNextLine
	}
	skipLWS(l)
	return lexValue
}

func lexValue(l *lexer) lexfn {
	// Per Google's specification, a value can consist of any character
	// other than a control character or '#'.
	for c := l.next(); !isCTL(c) && c != '#' && c != eof; c = l.next() {
	}
	l.backup()
	l.emit()
	return lexComment
}

func lexComment(l *lexer) lexfn {
	more := skipLWS(l)
	if !more {
		// New line of input
		return lexStart
	}
	// We ran into something on the same logical line. What is it?
	if c := l.peek(); c == '#' {
		for c := l.next(); c != '\n' && c != eof; c = l.next() {
		}
		l.backup()
		l.ignore()
		return lexEOL
	}
	// The current line looked like it would continue, but it didn't.
	// There was no comment. Therefore it actually ended on a newline.
	// We should treat the next line as fresh input.
	return lexStart
}

func lexEOL(l *lexer) lexfn {
	c := l.next()
	if c == '\n' {
		l.ignore()
		return lexStart
	}
	if c == eof {
		l.ignore()
		return lexStart
	}
	l.errorf("expected EOL")
	return lexNextLine
}

// Control characters are defined in RFC 1945, so use that definition
// instead of Unicode.
//
// See: http://www.ietf.org/rfc/rfc1945.txt
func isCTL(r rune) bool {
	if r < 32 || r == 127 {
		return true
	}
	return false
}

// LWS is defined in RFC 1945. "Linear whitespace" includes space and
// tab characters, but optionally continues a logical line as long as
// the continuation line starts with a space or tab.
//
// skipLWS advances the lexer past any LWS, whether or not it
// folds. The fold return value specifies whether the LWS spanned a
// fold. If fold is true, the next input is a logical continuation of
// the previous line. If fold is false, the next character represents
// the start of a new line of input.
//
// See: http://www.ietf.org/rfc/rfc1945.txt
func skipLWS(l *lexer) (fold bool) {
	afterEOL := false
	fold = true
	for c := l.next(); ; c = l.next() {
		if c == '\n' {
			afterEOL = true
			continue
		}
		if afterEOL && !(c == ' ' || c == '\t') {
			fold = false
			break
		}
		if !unicode.IsSpace(c) {
			break
		}
		afterEOL = false
	}
	l.backup()
	l.ignore()
	return fold
}

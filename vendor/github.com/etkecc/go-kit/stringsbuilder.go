package kit

import "strings"

// StringsBuilder wraps strings.Builder with fluent chaining (S, B, R) and implements io.Writer/StringWriter/RuneWriter.
type StringsBuilder struct {
	b strings.Builder
}

// NewStringsBuilder creates a new StringsBuilder; the zero value works too, this is for clarity.
func NewStringsBuilder() StringsBuilder {
	return StringsBuilder{}
}

// S is a shorthand for WriteString; returns the receiver for chaining.
func (sb *StringsBuilder) S(s string) *StringsBuilder {
	sb.b.WriteString(s)
	return sb
}

// B is a shorthand for WriteByte; returns the receiver for chaining.
func (sb *StringsBuilder) B(b byte) *StringsBuilder {
	sb.b.WriteByte(b)
	return sb
}

// R is a shorthand for WriteRune; returns the receiver for chaining.
func (sb *StringsBuilder) R(r rune) *StringsBuilder {
	sb.b.WriteRune(r)
	return sb
}

// String returns the accumulated string built by this StringsBuilder.
func (sb *StringsBuilder) String() string {
	return sb.b.String()
}

// Unwrap returns a copy of the underlying strings.Builder; mutating it does not affect the StringsBuilder.
func (sb *StringsBuilder) Unwrap() strings.Builder {
	return sb.b
}

// WriteString appends s and satisfies io.StringWriter; the error is always nil.
func (sb *StringsBuilder) WriteString(s string) (int, error) {
	return sb.b.WriteString(s)
}

// Write appends p and satisfies io.Writer; the error is always nil.
func (sb *StringsBuilder) Write(p []byte) (int, error) {
	return sb.b.Write(p)
}

// WriteRune appends r's UTF-8 encoding and satisfies io.RuneWriter; the error is always nil.
func (sb *StringsBuilder) WriteRune(r rune) (int, error) {
	return sb.b.WriteRune(r)
}

// Cap returns the current capacity of the accumulated string buffer.
func (sb *StringsBuilder) Cap() int {
	return sb.b.Cap()
}

// Grow grows capacity for n more bytes and returns the receiver for chaining, unlike strings.Builder.Grow.
func (sb *StringsBuilder) Grow(n int) *StringsBuilder {
	sb.b.Grow(n)
	return sb
}

// Len returns the current length of the accumulated string.
func (sb *StringsBuilder) Len() int {
	return sb.b.Len()
}

// Reset empties the StringsBuilder and returns the receiver for chaining, unlike strings.Builder.Reset.
func (sb *StringsBuilder) Reset() *StringsBuilder {
	sb.b.Reset()
	return sb
}

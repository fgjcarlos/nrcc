package service

import (
	"sort"
	"strings"
)

// Issue #882 — the scanner the logging model uses.
//
// settings.js is executable JavaScript, not a JSON document. The logging
// model has to know where the entries of an object literal are, so it can
// rewrite the properties the operator submitted and leave everything else
// alone. A regular expression cannot do that: a nested object, a string
// literal, a template literal or a comment can carry the same text as the
// property the model is looking for.
//
// The scanner below is depth aware. It tokenises the region instead of
// matching it, so a property that repeats deeper in the tree is never
// mistaken for a sibling, and a comment never hides an entry.

// jsEdit is one replacement inside a JavaScript source region.
type jsEdit struct {
	start int
	end   int
	text  string
}

// applyJSEdits applies the edits to the region. The edits are applied from
// the last to the first so the offsets of the pending edits stay valid.
func applyJSEdits(region string, edits []jsEdit) (string, bool) {
	if len(edits) == 0 {
		return region, false
	}
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start > edits[j].start })
	patched := region
	for _, edit := range edits {
		patched = patched[:edit.start] + edit.text + patched[edit.end:]
	}
	return patched, true
}

// jsObjectEntry is one direct child entry of a JavaScript object literal.
type jsObjectEntry struct {
	// key is the entry's property name.
	key string
	// start is the offset of the property name.
	start int
	// valueStart and valueEnd delimit the entry's value.
	valueStart int
	valueEnd   int
	// commaEnd is the offset just past the entry's trailing comma. It
	// equals valueEnd when the entry carries no comma.
	commaEnd int
}

// scanJSObjectEntries returns the direct child entries of the object
// literal whose body is `body` — the text between the braces.
//
// The scan is depth aware: a nested object, array or call expression is
// one value, and string literals, template literals and comments are
// skipped as a unit. Only the direct children are returned, so a property
// that repeats deeper in the tree is never mistaken for a sibling.
func scanJSObjectEntries(body string) []jsObjectEntry {
	var entries []jsObjectEntry
	i := 0
	for i < len(body) {
		switch {
		case body[i] == ',':
			i++
			continue
		case isJSSpace(body[i]):
			i++
			continue
		case body[i] == '/' && i+1 < len(body) && body[i+1] == '/':
			i = skipJSLineComment(body, i)
			continue
		case body[i] == '/' && i+1 < len(body) && body[i+1] == '*':
			i = skipJSBlockComment(body, i)
			continue
		}
		key, next, ok := scanJSObjectKey(body, i)
		if !ok {
			i++
			continue
		}
		colon := skipJSSpaceAndComments(body, next)
		if colon >= len(body) || body[colon] != ':' {
			i = next
			continue
		}
		valueStart := skipJSSpaceAndComments(body, colon+1)
		valueEnd := scanJSValueEnd(body, valueStart)
		commaEnd := valueEnd
		probe := valueEnd
		for probe < len(body) && isJSSpace(body[probe]) {
			probe++
		}
		if probe < len(body) && body[probe] == ',' {
			commaEnd = probe + 1
		}
		entries = append(entries, jsObjectEntry{
			key:        key,
			start:      i,
			valueStart: valueStart,
			valueEnd:   valueEnd,
			commaEnd:   commaEnd,
		})
		i = commaEnd
	}
	return entries
}

// firstJSEntryByKey maps the property names of the entries onto their
// first occurrence. A duplicated property keeps its first occurrence, the
// one JavaScript itself would read.
func firstJSEntryByKey(entries []jsObjectEntry) map[string]jsObjectEntry {
	byName := make(map[string]jsObjectEntry, len(entries))
	for _, entry := range entries {
		if _, seen := byName[entry.key]; !seen {
			byName[entry.key] = entry
		}
	}
	return byName
}

// insertJSEntries inserts the rendered entries into an object literal body.
//
// The insertion keeps the layout of the body: a multi-line body receives
// the entries on their own lines with the indentation of their siblings, a
// single-line body receives them inline, and an empty object literal
// receives them as its body. The returned offsets delimit the inserted
// text.
func insertJSEntries(body string, entries []jsObjectEntry, literals []string) (string, int, int, bool) {
	if len(literals) == 0 {
		return "", 0, 0, false
	}
	// The core is the body without its trailing whitespace: the insertion
	// point, so the whitespace before the closing brace survives.
	core := strings.TrimRight(body, " \t\r\n")
	if len(entries) == 0 {
		// An empty object literal: the entries become its body.
		if !strings.Contains(body, "\n") {
			return " " + strings.Join(literals, ", ") + " ", 0, len(body), true
		}
		closing := jsIndentOf(body)
		indent := closing + "  "
		text := "\n" + indent + strings.Join(literals, ",\n"+indent) + ",\n" + closing
		return text, 0, len(body), true
	}
	last := entries[len(entries)-1]
	if strings.Contains(body, "\n") {
		// A multi-line body: the entries go on their own lines, with the
		// indentation of their siblings.
		indent := jsLineIndent(body, last.start)
		text := "\n" + indent + strings.Join(literals, ",\n"+indent) + ","
		if !strings.HasSuffix(core, ",") {
			// The last entry carries no trailing comma: it becomes the
			// separator of the entries that follow. Without it the
			// rendered block would not be valid JavaScript.
			text = "," + text
		}
		return text, len(core), len(core), true
	}
	// A single-line body: the entries follow the last value inline. The
	// trailing comma of the last entry becomes their separator.
	separator := ", "
	if strings.HasSuffix(core, ",") {
		separator = " "
	}
	return separator + strings.Join(literals, ", "), len(core), len(core), true
}

// jsObjectBody returns the text between the braces of an object literal.
func jsObjectBody(text string) (string, bool) {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "{") {
		return "", false
	}
	closeIdx, _ := findMatchingBrace(trimmed, 0)
	if closeIdx < 0 {
		return "", false
	}
	return trimmed[1:closeIdx], true
}

// jsBlockBody returns the text between the braces of the first object
// literal in the text. Unlike jsObjectBody it accepts the whole entry of a
// block (`logging: { ... }`), not only a bare literal.
func jsBlockBody(text string) (string, bool) {
	open := strings.Index(text, "{")
	if open < 0 {
		return "", false
	}
	closeIdx, _ := findMatchingBrace(text, open)
	if closeIdx < 0 {
		return "", false
	}
	return text[open+1 : closeIdx], true
}

// jsLineIndent returns the indentation of the line that holds the offset.
func jsLineIndent(text string, offset int) string {
	lineStart := strings.LastIndexByte(text[:offset], '\n') + 1
	line := text[lineStart:offset]
	return line[:leadingWhitespaceEnd(line, 0)]
}

// jsIndentOf returns the indentation of the last line of the text.
func jsIndentOf(text string) string {
	line := text
	if idx := strings.LastIndexByte(text, '\n'); idx >= 0 {
		line = text[idx+1:]
	}
	return line[:leadingWhitespaceEnd(line, 0)]
}

// isJSSpace reports whether the byte is JavaScript whitespace.
func isJSSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// skipJSLineComment returns the offset just past the line comment.
func skipJSLineComment(text string, i int) int {
	if idx := strings.IndexByte(text[i:], '\n'); idx >= 0 {
		return i + idx + 1
	}
	return len(text)
}

// skipJSBlockComment returns the offset just past the block comment.
func skipJSBlockComment(text string, i int) int {
	if idx := strings.Index(text[i+2:], "*/"); idx >= 0 {
		return i + 2 + idx + 2
	}
	return len(text)
}

// skipJSSpaceAndComments returns the offset of the next significant byte.
func skipJSSpaceAndComments(text string, i int) int {
	for i < len(text) {
		switch {
		case isJSSpace(text[i]):
			i++
		case text[i] == '/' && i+1 < len(text) && text[i+1] == '/':
			i = skipJSLineComment(text, i)
		case text[i] == '/' && i+1 < len(text) && text[i+1] == '*':
			i = skipJSBlockComment(text, i)
		default:
			return i
		}
	}
	return i
}

// scanJSObjectKey reads the property name at the offset.
func scanJSObjectKey(text string, i int) (string, int, bool) {
	if i >= len(text) {
		return "", i, false
	}
	switch c := text[i]; {
	case c == '\'' || c == '"':
		end := skipJSQuoted(text, i, c)
		if end < 0 {
			return "", i, false
		}
		return text[i+1 : end-1], end, true
	case isJSIdentifierStart(c):
		j := i
		for j < len(text) && isJSIdentifierPart(text[j]) {
			j++
		}
		return text[i:j], j, true
	default:
		return "", i, false
	}
}

// isJSIdentifierStart reports whether the byte starts a JavaScript
// identifier.
func isJSIdentifierStart(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// isJSIdentifierPart reports whether the byte continues a JavaScript
// identifier.
func isJSIdentifierPart(c byte) bool {
	return isJSIdentifierStart(c) || (c >= '0' && c <= '9')
}

// skipJSQuoted returns the offset just past the quoted literal that starts
// at the offset, or -1 when the literal is not closed.
func skipJSQuoted(text string, i int, quote byte) int {
	for j := i + 1; j < len(text); j++ {
		switch text[j] {
		case '\\':
			j++
		case quote:
			return j + 1
		}
	}
	return -1
}

// scanJSValueEnd returns the offset just past the value that starts at the
// offset.
//
// The value ends at the first comma that is not nested inside an object,
// array or call expression, or at the end of the region.
func scanJSValueEnd(text string, i int) int {
	depth := 0
	for i < len(text) {
		switch c := text[i]; c {
		case '\'', '"':
			end := skipJSQuoted(text, i, c)
			if end < 0 {
				return len(text)
			}
			i = end
			continue
		case '`':
			end := skipJSTemplate(text, i)
			if end < 0 {
				return len(text)
			}
			i = end
			continue
		case '/':
			if i+1 < len(text) && text[i+1] == '/' {
				i = skipJSLineComment(text, i)
				continue
			}
			if i+1 < len(text) && text[i+1] == '*' {
				i = skipJSBlockComment(text, i)
				continue
			}
		case '{', '[', '(':
			depth++
		case '}', ']', ')':
			if depth == 0 {
				return i
			}
			depth--
		case ',':
			if depth == 0 {
				return trimJSTrailingSpace(text, i)
			}
		}
		i++
	}
	return trimJSTrailingSpace(text, len(text))
}

// trimJSTrailingSpace returns the offset just past the last significant
// byte of the region, so a value span never carries the whitespace that
// separates it from the next entry.
func trimJSTrailingSpace(text string, end int) int {
	for end > 0 && isJSSpace(text[end-1]) {
		end--
	}
	return end
}

// skipJSTemplate returns the offset just past the template literal that
// starts at the offset, or -1 when the literal is not closed.
func skipJSTemplate(text string, i int) int {
	for j := i + 1; j < len(text); j++ {
		switch text[j] {
		case '\\':
			j++
		case '`':
			return j + 1
		case '$':
			if j+1 < len(text) && text[j+1] == '{' {
				end := skipJSBraces(text, j+1)
				if end < 0 {
					return -1
				}
				j = end - 1
			}
		}
	}
	return -1
}

// skipJSBraces returns the offset just past the '}' that matches the '{' at
// the offset, or -1 when the braces are not closed.
func skipJSBraces(text string, openIdx int) int {
	depth := 0
	for j := openIdx; j < len(text); j++ {
		switch c := text[j]; c {
		case '\'', '"':
			end := skipJSQuoted(text, j, c)
			if end < 0 {
				return -1
			}
			j = end - 1
		case '`':
			end := skipJSTemplate(text, j)
			if end < 0 {
				return -1
			}
			j = end - 1
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return j + 1
			}
		}
	}
	return -1
}

package service

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Issue #882 — render the logging values the operator submitted.
//
// The renderer used to write a constant block (level: 'info', metrics:
// false, audit: false) whatever the operator submitted, so the logging
// controls reported a successful persistence while Node-RED kept running
// with the default level.
//
// Node-RED's contract (node-red 5.0.4, @node-red/util/lib/log.js →
// init(settings)):
//
//   - settings.logging is an object of handler name -> handler config.
//   - A handler becomes live when the name is "console" or the config
//     exposes a handler function. Any other name without a handler
//     function is ignored by Node-RED.
//   - The supported per-handler properties are level, metrics and audit.
//   - The levels are off, fatal, error, warn, info, debug and trace.
//
// The model keeps the logging block as a map of handler name to handler
// properties. The properties keep their JSON types, so a property the
// operator did not submit stays distinguishable from one submitted as
// false. The renderer rewrites what the payload carries and keeps
// everything else — another handler, a custom handler function, a
// property NRCC does not model — byte for byte.

// managedLoggingHandlers are the logging handlers NRCC's configuration UI
// edits. They are rendered first, in this order, so the output does not
// depend on the map iteration order.
var managedLoggingHandlers = []string{"console", "internal"}

// loggingProperties are the Node-RED per-handler properties NRCC models,
// in the order they are rendered.
var loggingProperties = []string{"level", "metrics", "audit"}

// loggingModel is the normalized logging block: handler name -> handler
// properties.
type loggingModel map[string]map[string]any

// newLoggingModel normalises the logging value of the structured model
// into the handler/property shape the renderer and the parser work with.
//
// A handler whose value is not an object literal carries no property and
// is dropped: it cannot hold a level, a metrics flag or an audit flag.
func newLoggingModel(value any) loggingModel {
	model := loggingModel{}
	switch typed := value.(type) {
	case nil:
		return model
	case map[string]any:
		for name, raw := range typed {
			if props, ok := loggingPropertiesOf(raw); ok {
				model[name] = props
			}
		}
	case map[string]map[string]any:
		for name, props := range typed {
			model[name] = props
		}
	}
	return model
}

// loggingPropertiesOf normalises one handler's properties.
func loggingPropertiesOf(raw any) (map[string]any, bool) {
	switch typed := raw.(type) {
	case map[string]any:
		return typed, true
	case map[string]string:
		props := make(map[string]any, len(typed))
		for key, value := range typed {
			props[key] = value
		}
		return props, true
	case map[string]map[string]any:
		props := make(map[string]any, len(typed))
		for key, value := range typed {
			props[key] = value
		}
		return props, true
	default:
		return nil, false
	}
}

// value renders the logging model back into the generic shape the
// structured configuration stores.
func (m loggingModel) value() map[string]any {
	value := make(map[string]any, len(m))
	for name, props := range m {
		entry := make(map[string]any, len(props))
		for key, property := range props {
			entry[key] = property
		}
		value[name] = entry
	}
	return value
}

// handlerNames returns the handler names of the model in a stable order:
// the handlers NRCC manages first, then the remaining ones alphabetically.
func (m loggingModel) handlerNames() []string {
	names := make([]string, 0, len(m))
	for _, managed := range managedLoggingHandlers {
		if _, ok := m[managed]; ok {
			names = append(names, managed)
		}
	}
	rest := make([]string, 0, len(m))
	for name := range m {
		if isManagedLoggingHandler(name) {
			continue
		}
		rest = append(rest, name)
	}
	sort.Strings(rest)
	return append(names, rest...)
}

// isManagedLoggingHandler reports whether the handler is one NRCC's
// configuration UI edits.
func isManagedLoggingHandler(name string) bool {
	for _, managed := range managedLoggingHandlers {
		if managed == name {
			return true
		}
	}
	return false
}

// defaultLoggingModel is the logging block NRCC writes when the operator
// has not configured one. It matches Node-RED's own console default, so an
// unconfigured deployment behaves the same with and without the block.
func defaultLoggingModel() loggingModel {
	return loggingModel{"console": defaultConsoleHandler()}
}

// defaultConsoleHandler is the console handler NRCC writes when the
// operator did not configure one.
func defaultConsoleHandler() map[string]any {
	return map[string]any{
		"level":   "info",
		"metrics": false,
		"audit":   false,
	}
}

// withDefaultConsoleHandler makes sure the logging block carries a console
// handler.
//
// Node-RED registers the entries of the block that are named "console" or
// that expose a handler function. A block without a console entry
// therefore leaves the runtime without any log handler at all — the
// runtime goes completely silent, which is never what an operator who
// configures a logging level expects. The default console handler matches
// Node-RED's own default.
func withDefaultConsoleHandler(logging loggingModel) loggingModel {
	if _, ok := logging["console"]; ok {
		return logging
	}
	merged := make(loggingModel, len(logging)+1)
	for name, props := range logging {
		merged[name] = props
	}
	merged["console"] = defaultConsoleHandler()
	return merged
}

// renderLoggingBlock renders the logging block from the operator-submitted
// logging values.
//
// The handlers NRCC manages come first, in the order of
// managedLoggingHandlers; the remaining handlers follow alphabetically, so
// the output does not depend on the map iteration order. An empty string is
// returned when the model holds no renderable handler.
func renderLoggingBlock(logging loggingModel) string {
	logging = withDefaultConsoleHandler(logging)
	entries := make([]string, 0, len(logging))
	for _, name := range logging.handlerNames() {
		entry := renderLoggingHandler(name, logging[name])
		if entry == "" {
			continue
		}
		entries = append(entries, "    "+entry+",")
	}
	if len(entries) == 0 {
		return ""
	}
	return "  logging: {\n" + strings.Join(entries, "\n") + "\n  },"
}

// renderLoggingHandler renders one handler entry of the logging block,
// without its indentation and without its trailing comma.
//
// Only the properties Node-RED supports for a handler are rendered. A
// property the model does not carry is omitted: the renderer never invents
// a value the operator did not submit.
func renderLoggingHandler(name string, props map[string]any) string {
	parts := make([]string, 0, len(loggingProperties))
	for _, property := range loggingProperties {
		value, ok := props[property]
		if !ok {
			continue
		}
		literal, ok := loggingLiteral(property, value)
		if !ok {
			continue
		}
		parts = append(parts, property+": "+literal)
	}
	if len(parts) == 0 {
		return ""
	}
	return name + ": { " + strings.Join(parts, ", ") + " }"
}

// loggingLiteral renders one logging property as a JavaScript literal.
//
// The level is a single-quoted string — the spelling Node-RED's own
// settings.js uses — and the flags are booleans. An unexpected value type
// is not rendered: the renderer never writes a value it cannot vouch for.
func loggingLiteral(property string, value any) (string, bool) {
	switch property {
	case "level":
		level, ok := value.(string)
		if !ok || level == "" {
			return "", false
		}
		return quoteJSString(level), true
	case "metrics", "audit":
		flag, ok := value.(bool)
		if !ok {
			return "", false
		}
		return fmt.Sprintf("%t", flag), true
	default:
		return "", false
	}
}

// quoteJSString renders a string as a single-quoted JavaScript literal.
//
// The value comes from the operator, so it is escaped: a level can never
// terminate the literal and inject JavaScript into settings.js.
func quoteJSString(value string) string {
	var builder strings.Builder
	builder.WriteByte('\'')
	for _, r := range value {
		switch r {
		case '\\':
			builder.WriteString(`\\`)
		case '\'':
			builder.WriteString(`\'`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&builder, `\u%04x`, r)
				continue
			}
			builder.WriteRune(r)
		}
	}
	builder.WriteByte('\'')
	return builder.String()
}

// patchLoggingSettings applies the operator-submitted logging values to a
// settings.js source.
//
// A source that declares no logging block receives the rendered block. A
// source that declares one is patched in place, so the entries and the
// properties outside the edited capability survive the save. A source that
// declares a block while the operator configured nothing is left untouched:
// the operator's own logging block is not NRCC's to rewrite.
func patchLoggingSettings(content string, value any) string {
	logging := newLoggingModel(value)
	if len(logging) == 0 {
		if _, ok := loggingBlockEntry(content); ok {
			return content
		}
		logging = defaultLoggingModel()
	}
	return patchLoggingBlock(content, logging)
}

// patchLoggingBlock applies the operator-submitted logging values to the
// logging block of an existing settings.js source.
//
// The block is patched in place. Only the properties the model carries are
// rewritten, and only for the handlers the model carries. Every other entry
// of the block — a custom handler with its own handler function, a
// property NRCC does not model — is preserved byte for byte.
func patchLoggingBlock(content string, logging loggingModel) string {
	if len(logging) == 0 {
		return content
	}
	entry, ok := loggingBlockEntry(content)
	if !ok {
		block := renderLoggingBlock(logging)
		if block == "" {
			return content
		}
		return appendPropertyBeforeClosingBrace(content, block+"\n")
	}
	patched, changed := patchLoggingEntry(entry, logging)
	if !changed {
		return content
	}
	start, end, _ := findTopLevelBlock(content, "logging")
	return content[:start] + patched + content[end:]
}

// patchLoggingEntry applies the operator-submitted logging values to the
// textual `logging: { ... }` entry of a settings.js source.
func patchLoggingEntry(entry string, logging loggingModel) (string, bool) {
	body, ok := jsBlockBody(entry)
	if !ok {
		return entry, false
	}
	newBody, changed := patchLoggingBody(body, logging)
	if !changed {
		return entry, false
	}
	open := strings.Index(entry, "{")
	closeIdx, _ := findMatchingBrace(entry, open)
	if open < 0 || closeIdx < 0 {
		return entry, false
	}
	return entry[:open+1] + newBody + entry[closeIdx:], true
}

// patchLoggingBody applies the operator-submitted logging values to the
// body of a logging block.
func patchLoggingBody(body string, logging loggingModel) (string, bool) {
	entries := scanJSObjectEntries(body)
	byName := firstJSEntryByKey(entries)

	var edits []jsEdit
	var inserted []string
	for _, name := range logging.handlerNames() {
		props := logging[name]
		if len(props) == 0 {
			continue
		}
		entry, found := byName[name]
		if !found {
			rendered := renderLoggingHandler(name, props)
			if rendered == "" {
				continue
			}
			inserted = append(inserted, rendered)
			continue
		}
		current := body[entry.valueStart:entry.valueEnd]
		merged, ok := mergeLoggingHandlerValue(name, current, props)
		if !ok || merged == current {
			continue
		}
		edits = append(edits, jsEdit{start: entry.valueStart, end: entry.valueEnd, text: merged})
	}
	_, sourceConsole := byName["console"]
	modelConsole := false
	if props, ok := logging["console"]; ok {
		modelConsole = renderLoggingHandler("console", props) != ""
	}
	if !sourceConsole && !modelConsole {
		// The block would leave the runtime without any log handler, so it
		// carries the default console handler.
		inserted = append(inserted, renderLoggingHandler("console", defaultConsoleHandler()))
	}
	if len(inserted) > 0 {
		text, start, end, ok := insertJSEntries(body, entries, inserted)
		if ok {
			edits = append(edits, jsEdit{start: start, end: end, text: text})
		}
	}
	return applyJSEdits(body, edits)
}

// mergeLoggingHandlerValue applies the operator-submitted properties to
// the textual value of one logging handler.
//
// The properties the source declares and NRCC does not model — a custom
// handler function, a property the operator did not submit — are preserved
// byte for byte.
func mergeLoggingHandlerValue(name, value string, props map[string]any) (string, bool) {
	body, ok := jsObjectBody(value)
	if !ok {
		// The handler is not an object literal (for example
		// `console: null`), so it cannot carry the submitted
		// properties: the entry is replaced by the rendered one.
		rendered := renderLoggingHandler(name, props)
		if rendered == "" {
			return value, false
		}
		return rendered, true
	}
	entries := scanJSObjectEntries(body)
	byName := firstJSEntryByKey(entries)

	var edits []jsEdit
	var inserted []string
	for _, property := range loggingProperties {
		propertyValue, ok := props[property]
		if !ok {
			continue
		}
		literal, ok := loggingLiteral(property, propertyValue)
		if !ok {
			continue
		}
		entry, found := byName[property]
		if !found {
			inserted = append(inserted, property+": "+literal)
			continue
		}
		current := body[entry.valueStart:entry.valueEnd]
		if loggingPropertyEqual(property, current, propertyValue) {
			continue
		}
		edits = append(edits, jsEdit{start: entry.valueStart, end: entry.valueEnd, text: literal})
	}
	if len(inserted) > 0 {
		text, start, end, ok := insertJSEntries(body, entries, inserted)
		if ok {
			edits = append(edits, jsEdit{start: start, end: end, text: text})
		}
	}
	merged, changed := applyJSEdits(body, edits)
	if !changed {
		return value, false
	}
	return "{" + merged + "}", true
}

// loggingPropertyEqual reports whether the source already carries the
// submitted value for a property. An equal value is left untouched so a
// save that changes nothing keeps the operator's spelling.
func loggingPropertyEqual(property, source string, value any) bool {
	switch property {
	case "level":
		level, ok := parseJSString(source)
		if !ok {
			return false
		}
		expected, ok := value.(string)
		return ok && level == expected
	case "metrics", "audit":
		flag, ok := parseJSBool(source)
		if !ok {
			return false
		}
		expected, ok := value.(bool)
		return ok && flag == expected
	default:
		return false
	}
}

// parseLoggingBlock reads the logging block out of a settings.js source.
//
// Only the properties Node-RED supports for a handler are read: the level,
// the metrics flag and the audit flag. A handler whose value is not an
// object literal carries no property; it is still reported, so the caller
// can tell the handler exists without inventing values for it.
func parseLoggingBlock(content string) loggingModel {
	model := loggingModel{}
	entry, ok := loggingBlockEntry(content)
	if !ok {
		return model
	}
	body, ok := jsBlockBody(entry)
	if !ok {
		return model
	}
	for _, child := range scanJSObjectEntries(body) {
		props := map[string]any{}
		if inner, ok := jsObjectBody(body[child.valueStart:child.valueEnd]); ok {
			for _, property := range scanJSObjectEntries(inner) {
				raw := inner[property.valueStart:property.valueEnd]
				switch property.key {
				case "level":
					if level, ok := parseJSString(raw); ok {
						props["level"] = level
					}
				case "metrics", "audit":
					if flag, ok := parseJSBool(raw); ok {
						props[property.key] = flag
					}
				}
			}
		}
		model[child.key] = props
	}
	return model
}

// loggingBlockEntry returns the textual `logging: { ... }` entry of a
// settings.js source.
func loggingBlockEntry(content string) (string, bool) {
	start, end, ok := findTopLevelBlock(content, "logging")
	if !ok {
		return "", false
	}
	return content[start:end], true
}

// parseJSString reads a single- or double-quoted JavaScript string literal.
func parseJSString(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) < 2 {
		return "", false
	}
	quote := trimmed[0]
	if quote != '\'' && quote != '"' {
		return "", false
	}
	if trimmed[len(trimmed)-1] != quote {
		return "", false
	}
	return unescapeJSString(trimmed[1 : len(trimmed)-1]), true
}

// unescapeJSString resolves the escape sequences of a JavaScript string
// literal body.
func unescapeJSString(value string) string {
	var builder strings.Builder
	for i := 0; i < len(value); i++ {
		if value[i] != '\\' || i+1 >= len(value) {
			builder.WriteByte(value[i])
			continue
		}
		i++
		switch value[i] {
		case 'n':
			builder.WriteByte('\n')
		case 'r':
			builder.WriteByte('\r')
		case 't':
			builder.WriteByte('\t')
		case 'b':
			builder.WriteByte('\b')
		case 'f':
			builder.WriteByte('\f')
		case '0':
			builder.WriteByte(0)
		case 'u':
			if i+4 < len(value) {
				code, err := strconv.ParseUint(value[i+1:i+5], 16, 16)
				if err != nil {
					builder.WriteByte('u')
					continue
				}
				// A \uXXXX escape holds four hex digits, so the value is
				// at most U+FFFF and always a valid code point.
				builder.WriteRune(rune(code)) // #nosec G115 -- the escape is at most U+FFFF.
				i += 4
				continue
			}
			builder.WriteByte('u')
		default:
			builder.WriteByte(value[i])
		}
	}
	return builder.String()
}

// parseJSBool reads a JavaScript boolean literal.
func parseJSBool(raw string) (bool, bool) {
	switch strings.TrimSpace(raw) {
	case "true":
		return true, true
	case "false":
		return false, true
	default:
		return false, false
	}
}

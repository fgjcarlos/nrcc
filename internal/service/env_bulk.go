package service

import (
	"fmt"
	"strings"

	"github.com/fgjcarlos/nrcc/internal/model"
)

// BulkEnvLine describes one parsed entry from a bulk-import payload.
type BulkEnvLine struct {
	Line  int    `json:"line"`
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
	Type  string `json:"type,omitempty"`
}

// BulkEnvIssue reports one validation problem with a specific line.
type BulkEnvIssue struct {
	Line   int    `json:"line"`
	Key    string `json:"key,omitempty"`
	Reason string `json:"reason"`
}

// BulkEnvResult is the dry-run / post-validation report.
type BulkEnvResult struct {
	Lines   []BulkEnvLine  `json:"lines"`
	Issues  []BulkEnvIssue `json:"issues"`
	Valid   bool           `json:"valid"`
	Summary string         `json:"summary"`
}

// ParseBulkEnv parses a Dokploy-style bulk payload. Each non-empty,
// non-comment line must look like KEY=VALUE[#type]. TYPE defaults to
// "string" when missing. Validation reuses ValidateEnvKey and ValidateValue
// so the rules stay identical to single-entry POST /api/env.
func ParseBulkEnv(content string) BulkEnvResult {
	var (
		lines  []BulkEnvLine
		issues []BulkEnvIssue
	)

	addIssue := func(lineNum int, key, reason string) {
		issues = append(issues, BulkEnvIssue{Line: lineNum, Key: key, Reason: reason})
	}

	seen := map[string]int{}
	for rawLineNum, raw := range strings.Split(content, "\n") {
		lineNum := rawLineNum + 1
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		eq := strings.IndexByte(trimmed, '=')
		if eq <= 0 {
			addIssue(lineNum, "", "missing '=' between key and value")
			continue
		}
		key := strings.TrimSpace(trimmed[:eq])
		rest := trimmed[eq+1:]

		typ := "string"
		if hash := strings.LastIndexByte(rest, '#'); hash >= 0 {
			candidate := strings.TrimSpace(rest[hash+1:])
			if candidate != "" {
				if !validBulkType(candidate) {
					addIssue(lineNum, key, fmt.Sprintf("unknown type %q (expected string|number|boolean|secret)", candidate))
					continue
				}
				typ = candidate
				rest = rest[:hash]
			}
		}
		value := rest // honour spaces inside value

		if err := ValidateEnvKey(key); err != nil {
			addIssue(lineNum, key, err.Error())
			continue
		}
		if err := ValidateValue(value, typ); err != nil {
			addIssue(lineNum, key, err.Error())
			continue
		}
		if prev, ok := seen[key]; ok {
			addIssue(lineNum, key, fmt.Sprintf("duplicate key (first seen on line %d)", prev))
			continue
		}
		seen[key] = lineNum
		lines = append(lines, BulkEnvLine{Line: lineNum, Key: key, Value: value, Type: typ})
	}

	result := BulkEnvResult{Lines: lines, Issues: issues, Valid: len(issues) == 0 && len(lines) > 0}
	switch {
	case len(lines) == 0 && len(issues) == 0:
		result.Summary = "empty input"
	case len(issues) > 0:
		result.Summary = fmt.Sprintf("%d invalid line(s)", len(issues))
	default:
		result.Summary = fmt.Sprintf("%d variable(s) ready", len(lines))
	}
	return result
}

func validBulkType(typ string) bool {
	switch typ {
	case "string", "number", "boolean", "secret":
		return true
	}
	return false
}

// ApplyBulkEnv imports a previously validated bulk payload into NRCC and the
// active Node-RED runtime. Caller guarantees BulkEnvResult.Valid == true.
// Secrets persist encrypted and never reach flows.json; the rest route
// through EnvService.Set which already drives syncNodeRedGlobalEnv.
func (s *EnvService) ApplyBulkEnv(parsed BulkEnvResult, restart func(func() error) (bool, error)) (BulkEnvResult, error) {
	if !parsed.Valid {
		return parsed, fmt.Errorf("bulk payload failed validation")
	}
	for _, line := range parsed.Lines {
		encrypted := line.Type == "secret"
		set := func() error {
			return s.Set(line.Key, line.Value, line.Type, "bulk import", encrypted)
		}
		if restart != nil {
			if _, err := restart(set); err != nil {
				return parsed, fmt.Errorf("line %d (%s): %w", line.Line, line.Key, err)
			}
		} else if err := set(); err != nil {
			return parsed, fmt.Errorf("line %d (%s): %w", line.Line, line.Key, err)
		}
	}
	return parsed, nil
}

// unused but keep package symbol referenced on stripped builds
var _ = model.EnvVar{}

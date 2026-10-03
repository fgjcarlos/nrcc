package service

import (
	"regexp"
	"strings"
)

var (
	ansiSequencePattern    = regexp.MustCompile("\\x1b\\[[0-?]*[ -/]*[@-~]")
	nodeREDLabelPattern    = regexp.MustCompile(`(?i)^\s*Node-RED\s+(.*?)\s*$`)
	semanticVersionPattern = regexp.MustCompile(`^v?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
)

// nodeREDVersionFromOutput extracts only an explicit Node-RED version line or
// a standalone version. Multiline output without a Node-RED label is unknown.
func nodeREDVersionFromOutput(output string) string {
	output = strings.TrimSpace(ansiSequencePattern.ReplaceAllString(output, ""))
	lines := strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n")
	var labeled string
	foundLabel := false
	for _, line := range lines {
		if match := nodeREDLabelPattern.FindStringSubmatch(line); match != nil {
			foundLabel = true
			labeled = match[1]
			break
		}
	}
	if foundLabel {
		return validatedNodeREDVersion(labeled)
	}
	if len(lines) != 1 {
		return ""
	}
	return validatedNodeREDVersion(strings.TrimSpace(lines[0]))
}

func validatedNodeREDVersion(value string) string {
	match := semanticVersionPattern.FindStringSubmatch(strings.TrimSpace(value))
	if match == nil {
		return ""
	}
	if match[4] != "" {
		for _, identifier := range strings.Split(match[4], ".") {
			if len(identifier) > 1 && identifier[0] == '0' && allDigits(identifier) {
				return ""
			}
		}
	}
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}

func allDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return value != ""
}

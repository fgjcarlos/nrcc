package service

import "testing"

func TestNodeREDVersionFromOutput(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"bare with prefix", "v5.0.7", "5.0.7"},
		{"bare", "5.0.7", "5.0.7"},
		{"canonical multiline CRLF", "Node-RED v5.0.7\r\nNode.js v24.20.0\r\nLinux", "5.0.7"},
		{"label and metadata", "Node-RED 5.1.0-rc.2+build.7\nNode.js v24.20.0", "5.1.0-rc.2+build.7"},
		{"ANSI label", "\x1b[32mNode-RED v5.0.7\x1b[0m\n", "5.0.7"},
		{"malformed label", "Node-RED v5.0\nNode.js v24.20.0", ""},
		{"missing Node-RED line", "Node.js v24.20.0\nLinux", ""},
		{"unrelated number", "Linux kernel 6.8.0", ""},
		{"trailing junk", "5.0.7 unexpected", ""},
		{"invalid numeric prerelease", "Node-RED 5.0.7-alpha.01", ""},
		{"invalid core leading zero", "Node-RED 05.0.7", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := nodeREDVersionFromOutput(tt.input); got != tt.want {
				t.Fatalf("nodeREDVersionFromOutput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

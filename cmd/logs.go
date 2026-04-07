package cmd

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nrcc/internal/model"
	"nrcc/internal/service"
)

// runLogs handles the "nrcc logs" command
func runLogs(args []string) error {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	levelFlag := fs.String("level", "", "filter by log level (info, warn, error, debug)")
	sourceFlag := fs.String("source", "", "filter by source prefix")
	linesFlag := fs.Int("lines", 50, "number of lines to show (default 50)")
	followFlag := fs.Bool("follow", false, "tail the log file (like tail -f)")
	fFlag := fs.Bool("f", false, "short for --follow")
	jsonFlag := fs.Bool("json", false, "output as JSONL")
	fs.Parse(args)

	// --f is short for --follow
	if *fFlag {
		*followFlag = true
	}

	envSvc := service.NewEnvironmentService()
	dataDir, err := envSvc.DefaultDataDir()
	if err != nil {
		return err
	}

	logPath := filepath.Join(dataDir, "logs", "app.log")

	if *followFlag {
		return tailLogFile(logPath, *levelFlag, *sourceFlag, *jsonFlag)
	}

	// Read log file and print last N lines
	entries, err := readLogFile(logPath, *linesFlag)
	if err != nil {
		return err
	}

	// Filter by level and source
	filtered := filterLogEntries(entries, *levelFlag, *sourceFlag)

	// Print
	if *jsonFlag {
		for _, entry := range filtered {
			data, _ := json.Marshal(entry)
			fmt.Println(string(data))
		}
	} else {
		for _, entry := range filtered {
			printLogEntry(entry)
		}
	}

	return nil
}

// readLogFile reads the log file and returns all entries
func readLogFile(logPath string, maxLines int) ([]model.LogEntry, error) {
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []model.LogEntry{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var entries []model.LogEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry model.LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // Skip malformed lines
		}
		entries = append(entries, entry)
	}

	// Return last N lines
	if len(entries) > maxLines {
		entries = entries[len(entries)-maxLines:]
	}

	return entries, nil
}

// filterLogEntries filters entries by level and source
func filterLogEntries(entries []model.LogEntry, level, source string) []model.LogEntry {
	if level == "" && source == "" {
		return entries
	}

	var filtered []model.LogEntry
	for _, entry := range entries {
		if level != "" && entry.Level != level {
			continue
		}
		if source != "" && !strings.HasPrefix(entry.Source, source) {
			continue
		}
		filtered = append(filtered, entry)
	}

	return filtered
}

// printLogEntry prints a single log entry in human-readable format
func printLogEntry(entry model.LogEntry) {
	levelStr := fmt.Sprintf("[%s]", strings.ToUpper(entry.Level))
	timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
	source := fmt.Sprintf("[%s]", entry.Source)

	fmt.Printf("%s %s %s %s\n", levelStr, timestamp, source, entry.Message)
}

// tailLogFile tails the log file like `tail -f`
func tailLogFile(logPath string, level, source string, asJSON bool) error {
	// Simple implementation: read file, print lines, wait for new content
	file, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Log file not found: %s\n", logPath)
			return nil
		}
		return err
	}
	defer file.Close()

	// Read all current lines
	scanner := bufio.NewScanner(file)
	var lastOffset int64
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry model.LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Filter and print
		if level != "" && entry.Level != level {
			continue
		}
		if source != "" && !strings.HasPrefix(entry.Source, source) {
			continue
		}

		if asJSON {
			data, _ := json.Marshal(entry)
			fmt.Println(string(data))
		} else {
			printLogEntry(entry)
		}

		lastOffset = int64(scanner.Bytes()[len(scanner.Bytes())-1])
	}

	// Watch for new lines
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		file, err := os.Open(logPath)
		if err != nil {
			continue
		}

		file.Seek(lastOffset, 0)
		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var entry model.LogEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}

			// Filter and print
			if level != "" && entry.Level != level {
				continue
			}
			if source != "" && !strings.HasPrefix(entry.Source, source) {
				continue
			}

			if asJSON {
				data, _ := json.Marshal(entry)
				fmt.Println(string(data))
			} else {
				printLogEntry(entry)
			}
		}

		lastOffset, _ = file.Seek(0, 2)
		file.Close()
	}

	return nil
}

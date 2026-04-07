package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"nrcc/internal/model"
)

// ValidateFullAppConfig validates a FullAppConfig and returns a list of field errors.
// It enforces all validation rules from the specification.
func ValidateFullAppConfig(cfg model.FullAppConfig) []model.FieldError {
	var errors []model.FieldError

	// Server Section
	if cfg.Server.UIPort < 1 || cfg.Server.UIPort > 65535 {
		errors = append(errors, model.FieldError{
			Field:   "server.uiPort",
			Message: "Port must be between 1 and 65535",
		})
	}

	if strings.Contains(cfg.Server.UIHost, " ") {
		errors = append(errors, model.FieldError{
			Field:   "server.uiHost",
			Message: "UIHost must not contain spaces",
		})
	}

	if len(cfg.Server.UIHost) > 253 {
		errors = append(errors, model.FieldError{
			Field:   "server.uiHost",
			Message: "UIHost must not exceed 253 characters",
		})
	}

	if !strings.HasPrefix(cfg.Server.HTTPAdminRoot, "/") {
		errors = append(errors, model.FieldError{
			Field:   "server.httpAdminRoot",
			Message: "HTTPAdminRoot must start with /",
		})
	}

	if strings.Contains(cfg.Server.HTTPAdminRoot, " ") {
		errors = append(errors, model.FieldError{
			Field:   "server.httpAdminRoot",
			Message: "HTTPAdminRoot must not contain spaces",
		})
	}

	if cfg.Server.HTTPNodeRoot != "false" && !strings.HasPrefix(cfg.Server.HTTPNodeRoot, "/") {
		errors = append(errors, model.FieldError{
			Field:   "server.httpNodeRoot",
			Message: "HTTPNodeRoot must start with / or equal \"false\"",
		})
	}

	if strings.Contains(cfg.Server.HTTPNodeRoot, " ") && cfg.Server.HTTPNodeRoot != "false" {
		errors = append(errors, model.FieldError{
			Field:   "server.httpNodeRoot",
			Message: "HTTPNodeRoot must not contain spaces",
		})
	}

	if cfg.Server.HTTPStatic != "" && strings.Contains(cfg.Server.HTTPStatic, "../") {
		errors = append(errors, model.FieldError{
			Field:   "server.httpStatic",
			Message: "HTTPStatic must not contain ../",
		})
	}

	// Security Section
	if cfg.Security.CredentialSecret != "" {
		if len(cfg.Security.CredentialSecret) < 12 {
			errors = append(errors, model.FieldError{
				Field:   "security.credentialSecret",
				Message: "CredentialSecret must be at least 12 characters when set",
			})
		}
		if len(cfg.Security.CredentialSecret) > 256 {
			errors = append(errors, model.FieldError{
				Field:   "security.credentialSecret",
				Message: "CredentialSecret must not exceed 256 characters",
			})
		}
	}

	if cfg.Security.SessionExpiryTime != 0 {
		if cfg.Security.SessionExpiryTime < 300 || cfg.Security.SessionExpiryTime > 2592000 {
			errors = append(errors, model.FieldError{
				Field:   "security.sessionExpiryTime",
				Message: "SessionExpiryTime must be between 300 and 2592000 seconds",
			})
		}
	}

	if cfg.Security.AdminAuth != nil {
		if cfg.Security.AdminAuth.Type != "credentials" && cfg.Security.AdminAuth.Type != "strategy" {
			errors = append(errors, model.FieldError{
				Field:   "security.adminAuth.type",
				Message: "AdminAuth type must be \"credentials\" or \"strategy\"",
			})
		}

		if cfg.Security.AdminAuth.Type == "credentials" {
			if len(cfg.Security.AdminAuth.Users) == 0 {
				errors = append(errors, model.FieldError{
					Field:   "security.adminAuth.users",
					Message: "AdminAuth must have at least 1 user when type is \"credentials\"",
				})
			}

			for i, user := range cfg.Security.AdminAuth.Users {
				if user.Username == "" {
					errors = append(errors, model.FieldError{
						Field:   fmt.Sprintf("security.adminAuth.users[%d].username", i),
						Message: "Username must not be empty",
					})
				}
				if len(user.Username) > 64 {
					errors = append(errors, model.FieldError{
						Field:   fmt.Sprintf("security.adminAuth.users[%d].username", i),
						Message: "Username must not exceed 64 characters",
					})
				}
				if strings.Contains(user.Username, " ") {
					errors = append(errors, model.FieldError{
						Field:   fmt.Sprintf("security.adminAuth.users[%d].username", i),
						Message: "Username must not contain spaces",
					})
				}
				if user.Permissions != "*" && user.Permissions != "read" {
					errors = append(errors, model.FieldError{
						Field:   fmt.Sprintf("security.adminAuth.users[%d].permissions", i),
						Message: "Permissions must be \"*\" or \"read\"",
					})
				}
			}
		}
	}

	if cfg.Security.HTTPNodeAuth != nil {
		if cfg.Security.HTTPNodeAuth.User == "" {
			errors = append(errors, model.FieldError{
				Field:   "security.httpNodeAuth.user",
				Message: "HTTPNodeAuth user must not be empty",
			})
		}
		if cfg.Security.HTTPNodeAuth.Pass == "" {
			errors = append(errors, model.FieldError{
				Field:   "security.httpNodeAuth.pass",
				Message: "HTTPNodeAuth pass must not be empty",
			})
		}
	}

	// Flows Section
	if cfg.Flows.FlowFile == "" {
		errors = append(errors, model.FieldError{
			Field:   "flows.flowFile",
			Message: "FlowFile must not be empty",
		})
	}

	if !strings.HasSuffix(cfg.Flows.FlowFile, ".json") {
		errors = append(errors, model.FieldError{
			Field:   "flows.flowFile",
			Message: "FlowFile must end with .json",
		})
	}

	if strings.Contains(cfg.Flows.FlowFile, "/") || strings.Contains(cfg.Flows.FlowFile, `\`) {
		errors = append(errors, model.FieldError{
			Field:   "flows.flowFile",
			Message: "FlowFile must be a file name, not a path",
		})
	}

	if cfg.Flows.UserDir != "" {
		if !strings.HasPrefix(cfg.Flows.UserDir, "/") {
			errors = append(errors, model.FieldError{
				Field:   "flows.userDir",
				Message: "UserDir must start with / when set",
			})
		}
		if strings.Contains(cfg.Flows.UserDir, "../") {
			errors = append(errors, model.FieldError{
				Field:   "flows.userDir",
				Message: "UserDir must not contain ../",
			})
		}
	}

	if cfg.Flows.NodesDir != "" {
		if !strings.HasPrefix(cfg.Flows.NodesDir, "/") {
			errors = append(errors, model.FieldError{
				Field:   "flows.nodesDir",
				Message: "NodesDir must start with / when set",
			})
		}
		if strings.Contains(cfg.Flows.NodesDir, "../") {
			errors = append(errors, model.FieldError{
				Field:   "flows.nodesDir",
				Message: "NodesDir must not contain ../",
			})
		}
	}

	// Context Storage Section
	if cfg.ContextStorage.Default == "" {
		errors = append(errors, model.FieldError{
			Field:   "contextStorage.default",
			Message: "ContextStorage default must not be empty",
		})
	}

	if len(cfg.ContextStorage.Stores) == 0 {
		errors = append(errors, model.FieldError{
			Field:   "contextStorage.stores",
			Message: "ContextStorage stores must have at least 1 entry",
		})
	}

	if cfg.ContextStorage.Default != "" {
		if _, exists := cfg.ContextStorage.Stores[cfg.ContextStorage.Default]; !exists {
			errors = append(errors, model.FieldError{
				Field:   "contextStorage.default",
				Message: fmt.Sprintf("ContextStorage default \"%s\" must be a key in stores", cfg.ContextStorage.Default),
			})
		}
	}

	for name, store := range cfg.ContextStorage.Stores {
		if store.Module != "memory" && store.Module != "localfilesystem" {
			errors = append(errors, model.FieldError{
				Field:   fmt.Sprintf("contextStorage.stores[%s].module", name),
				Message: "Store module must be \"memory\" or \"localfilesystem\"",
			})
		}
	}

	// Logging Section
	validLogLevels := map[string]bool{
		"fatal": true, "error": true, "warn": true, "info": true, "debug": true, "trace": true,
	}
	if !validLogLevels[cfg.Logging.Console.Level] {
		errors = append(errors, model.FieldError{
			Field:   "logging.console.level",
			Message: "Logging console level must be one of: fatal, error, warn, info, debug, trace",
		})
	}

	// Runtime Section
	if cfg.Runtime.FunctionTimeout < 0 || cfg.Runtime.FunctionTimeout > 3600 {
		errors = append(errors, model.FieldError{
			Field:   "runtime.functionTimeout",
			Message: "FunctionTimeout must be between 0 and 3600",
		})
	}

	if cfg.Runtime.DebugMaxLength < 100 || cfg.Runtime.DebugMaxLength > 100000 {
		errors = append(errors, model.FieldError{
			Field:   "runtime.debugMaxLength",
			Message: "DebugMaxLength must be between 100 and 100000",
		})
	}

	if cfg.Runtime.NodeMessageBufferMaxLength < 0 || cfg.Runtime.NodeMessageBufferMaxLength > 10000 {
		errors = append(errors, model.FieldError{
			Field:   "runtime.nodeMessageBufferMaxLength",
			Message: "NodeMessageBufferMaxLength must be between 0 and 10000",
		})
	}

	if cfg.Runtime.ExternalModules != nil {
		if cfg.Runtime.ExternalModules.AutoInstallRetry < 5 || cfg.Runtime.ExternalModules.AutoInstallRetry > 3600 {
			errors = append(errors, model.FieldError{
				Field:   "runtime.externalModules.autoInstallRetry",
				Message: "ExternalModules autoInstallRetry must be between 5 and 3600",
			})
		}
	}

	// HTTPS Section
	if cfg.HTTPS.Enabled {
		if cfg.HTTPS.KeyFile == "" {
			errors = append(errors, model.FieldError{
				Field:   "https.keyFile",
				Message: "HTTPS keyFile must not be empty when HTTPS is enabled",
			})
		}
		if cfg.HTTPS.CertFile == "" {
			errors = append(errors, model.FieldError{
				Field:   "https.certFile",
				Message: "HTTPS certFile must not be empty when HTTPS is enabled",
			})
		}
	}

	// Node Reconnect Section
	if cfg.NodeReconnect.MQTTReconnectTime < 100 || cfg.NodeReconnect.MQTTReconnectTime > 300000 {
		errors = append(errors, model.FieldError{
			Field:   "nodeReconnect.mqttReconnectTime",
			Message: "MQTTReconnectTime must be between 100 and 300000",
		})
	}

	if cfg.NodeReconnect.SerialReconnectTime < 100 || cfg.NodeReconnect.SerialReconnectTime > 300000 {
		errors = append(errors, model.FieldError{
			Field:   "nodeReconnect.serialReconnectTime",
			Message: "SerialReconnectTime must be between 100 and 300000",
		})
	}

	if cfg.NodeReconnect.SocketReconnectTime < 100 || cfg.NodeReconnect.SocketReconnectTime > 300000 {
		errors = append(errors, model.FieldError{
			Field:   "nodeReconnect.socketReconnectTime",
			Message: "SocketReconnectTime must be between 100 and 300000",
		})
	}

	if cfg.NodeReconnect.SocketTimeout < 1000 || cfg.NodeReconnect.SocketTimeout > 600000 {
		errors = append(errors, model.FieldError{
			Field:   "nodeReconnect.socketTimeout",
			Message: "SocketTimeout must be between 1000 and 600000",
		})
	}

	// EditorTheme Section
	if len(cfg.EditorTheme.Theme) > 128 {
		errors = append(errors, model.FieldError{
			Field:   "editorTheme.theme",
			Message: "Theme must not exceed 128 characters",
		})
	}

	if cfg.EditorTheme.CodeEditor != nil {
		if cfg.EditorTheme.CodeEditor.Lib != "ace" && cfg.EditorTheme.CodeEditor.Lib != "monaco" {
			errors = append(errors, model.FieldError{
				Field:   "editorTheme.codeEditor.lib",
				Message: "CodeEditor lib must be \"ace\" or \"monaco\"",
			})
		}
	}

	if cfg.EditorTheme.Page != nil {
		if len(cfg.EditorTheme.Page.Title) > 128 {
			errors = append(errors, model.FieldError{
				Field:   "editorTheme.page.title",
				Message: "Page title must not exceed 128 characters",
			})
		}
	}

	if cfg.EditorTheme.Header != nil {
		if len(cfg.EditorTheme.Header.Title) > 128 {
			errors = append(errors, model.FieldError{
				Field:   "editorTheme.header.title",
				Message: "Header title must not exceed 128 characters",
			})
		}
		if cfg.EditorTheme.Header.URL != "" {
			if !strings.HasPrefix(cfg.EditorTheme.Header.URL, "http://") && !strings.HasPrefix(cfg.EditorTheme.Header.URL, "https://") {
				errors = append(errors, model.FieldError{
					Field:   "editorTheme.header.url",
					Message: "Header URL must start with http:// or https:// when set",
				})
			}
		}
	}

	if cfg.EditorTheme.DeployButton != nil {
		if cfg.EditorTheme.DeployButton.Type != "simple" && cfg.EditorTheme.DeployButton.Type != "confirm" {
			errors = append(errors, model.FieldError{
				Field:   "editorTheme.deployButton.type",
				Message: "DeployButton type must be \"simple\" or \"confirm\"",
			})
		}
		if len(cfg.EditorTheme.DeployButton.Label) > 64 {
			errors = append(errors, model.FieldError{
				Field:   "editorTheme.deployButton.label",
				Message: "DeployButton label must not exceed 64 characters",
			})
		}
	}

	// Palette Section
	seenCategories := make(map[string]bool)
	for i, category := range cfg.Palette.Categories {
		if category == "" {
			errors = append(errors, model.FieldError{
				Field:   fmt.Sprintf("palette.categories[%d]", i),
				Message: "Category must not be empty",
			})
		}
		if len(category) > 64 {
			errors = append(errors, model.FieldError{
				Field:   fmt.Sprintf("palette.categories[%d]", i),
				Message: "Category must not exceed 64 characters",
			})
		}
		if seenCategories[category] {
			errors = append(errors, model.FieldError{
				Field:   fmt.Sprintf("palette.categories[%d]", i),
				Message: fmt.Sprintf("Category \"%s\" is duplicated", category),
			})
		}
		seenCategories[category] = true
	}

	return errors
}

// ComputeConfigDiff flattens both configs to dot-notation paths and returns entries where values differ.
func ComputeConfigDiff(current, proposed model.FullAppConfig) []model.ConfigDiffEntry {
	var diff []model.ConfigDiffEntry

	// Marshal both configs to JSON, then flatten to compare
	currentJSON, _ := json.Marshal(current)
	proposedJSON, _ := json.Marshal(proposed)

	var currentMap, proposedMap map[string]interface{}
	json.Unmarshal(currentJSON, &currentMap)
	json.Unmarshal(proposedJSON, &proposedMap)

	// Flatten both maps and compare
	currentFlat := flattenMap(currentMap, "")
	proposedFlat := flattenMap(proposedMap, "")

	// Find all keys and compare values
	allKeys := make(map[string]bool)
	for k := range currentFlat {
		allKeys[k] = true
	}
	for k := range proposedFlat {
		allKeys[k] = true
	}

	for key := range allKeys {
		oldVal, oldExists := currentFlat[key]
		newVal, newExists := proposedFlat[key]

		// Use empty string as default for missing values
		if !oldExists {
			oldVal = ""
		}
		if !newExists {
			newVal = ""
		}

		// Compare values
		oldStr := fmt.Sprintf("%v", oldVal)
		newStr := fmt.Sprintf("%v", newVal)
		if oldStr != newStr {
			diff = append(diff, model.ConfigDiffEntry{
				Field:    key,
				OldValue: oldStr,
				NewValue: newStr,
			})
		}
	}

	return diff
}

// flattenMap flattens a nested map into dot-notation keys
func flattenMap(data map[string]interface{}, prefix string) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range data {
		var key string
		if prefix == "" {
			key = k
		} else {
			key = prefix + "." + k
		}

		switch val := v.(type) {
		case map[string]interface{}:
			// Recursively flatten nested maps
			nested := flattenMap(val, key)
			for nk, nv := range nested {
				result[nk] = nv
			}
		case []interface{}:
			// For arrays, include each element with index
			for i, item := range val {
				itemKey := fmt.Sprintf("%s[%d]", key, i)
				if m, ok := item.(map[string]interface{}); ok {
					nested := flattenMap(m, itemKey)
					for nk, nv := range nested {
						result[nk] = nv
					}
				} else {
					result[itemKey] = item
				}
			}
		default:
			result[key] = v
		}
	}

	return result
}

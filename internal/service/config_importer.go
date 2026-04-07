package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"nrcc/internal/model"
)

// ImportSettingsJS parses a Node-RED settings.js string and returns the equivalent FullAppConfig.
// unrecognized is a list of JS keys that were found but not mapped to any known config field.
// Returns an error only if the content cannot be parsed at all (e.g., no module.exports found).
func ImportSettingsJS(content string) (cfg model.FullAppConfig, unrecognized []string, err error) {
	// Start with defaults
	cfg = model.DefaultFullAppConfig()

	// Strip comments from content
	content = stripJSComments(content)

	// Extract the module.exports = { ... } block
	body, err := extractExportsBlock(content)
	if err != nil {
		return cfg, nil, err
	}

	// Extract top-level key-value pairs
	pairs := extractTopLevelPairs(body)

	// Track recognized and unrecognized keys
	recognized := make(map[string]bool)
	var unrec []string

	// Define all known keys
	knownKeys := map[string]bool{
		// Server
		"uiPort":        true,
		"uiHost":        true,
		"httpAdminRoot": true,
		"httpNodeRoot":  true,
		"httpStatic":    true,
		"disableEditor": true,
		// Security
		"credentialSecret":  true,
		"sessionExpiryTime": true,
		"adminAuth":         true,
		"httpNodeAuth":      true,
		// Flows
		"flowFile":       true,
		"flowFilePretty": true,
		"userDir":        true,
		"nodesDir":       true,
		// Context Storage
		"contextStorage": true,
		// Logging
		"logging": true,
		// Runtime
		"functionExternalModules":    true,
		"functionTimeout":            true,
		"debugMaxLength":             true,
		"diagnosticsEnabled":         true,
		"safeMode":                   true,
		"nodeMessageBufferMaxLength": true,
		"externalModules":            true,
		// HTTPS
		"https": true,
		// Node Reconnect
		"mqttReconnectTime":   true,
		"serialReconnectTime": true,
		"socketReconnectTime": true,
		"socketTimeout":       true,
		// Editor Theme
		"editorTheme": true,
		// Palette
		"paletteCategories": true,
	}

	// Parse known keys
	for key, rawValue := range pairs {
		if !knownKeys[key] {
			unrec = append(unrec, key)
			continue
		}

		recognized[key] = true

		switch key {
		// Server section
		case "uiPort":
			if v, ok := parseJSInt(rawValue); ok {
				cfg.Server.UIPort = v
			}
		case "uiHost":
			if v, ok := parseJSString(rawValue); ok {
				cfg.Server.UIHost = v
			}
		case "httpAdminRoot":
			if v, ok := parseJSString(rawValue); ok {
				cfg.Server.HTTPAdminRoot = v
			}
		case "httpNodeRoot":
			// Special case: if raw value is the JS literal "false", store as string "false"
			if strings.TrimSpace(rawValue) == "false" {
				cfg.Server.HTTPNodeRoot = "false"
			} else if v, ok := parseJSString(rawValue); ok {
				cfg.Server.HTTPNodeRoot = v
			}
		case "httpStatic":
			if v, ok := parseJSString(rawValue); ok {
				cfg.Server.HTTPStatic = v
			}
		case "disableEditor":
			if v, ok := parseJSBool(rawValue); ok {
				cfg.Server.DisableEditor = v
			}

		// Security section
		case "credentialSecret":
			// Special case: if JS value is `false`, store as empty string
			if strings.TrimSpace(rawValue) == "false" {
				cfg.Security.CredentialSecret = ""
			} else if v, ok := parseJSString(rawValue); ok {
				cfg.Security.CredentialSecret = v
			}
		case "sessionExpiryTime":
			if v, ok := parseJSInt(rawValue); ok {
				cfg.Security.SessionExpiryTime = int64(v)
			}
		case "adminAuth":
			if adminAuth, ok := parseAdminAuth(rawValue); ok {
				cfg.Security.AdminAuth = adminAuth
			}
		case "httpNodeAuth":
			if httpNodeAuth, ok := parseHTTPNodeAuth(rawValue); ok {
				cfg.Security.HTTPNodeAuth = httpNodeAuth
			}

		// Flows section
		case "flowFile":
			if v, ok := parseJSString(rawValue); ok {
				cfg.Flows.FlowFile = v
			}
		case "flowFilePretty":
			if v, ok := parseJSBool(rawValue); ok {
				cfg.Flows.FlowFilePretty = v
			}
		case "userDir":
			if v, ok := parseJSString(rawValue); ok {
				cfg.Flows.UserDir = v
			}
		case "nodesDir":
			if v, ok := parseJSString(rawValue); ok {
				cfg.Flows.NodesDir = v
			}

		// Context Storage section
		case "contextStorage":
			if ctxStorage, ok := parseContextStorage(rawValue); ok {
				cfg.ContextStorage = ctxStorage
			}

		// Logging section
		case "logging":
			if logging, ok := parseLogging(rawValue); ok {
				cfg.Logging = logging
			}

		// Runtime section
		case "functionExternalModules":
			if v, ok := parseJSBool(rawValue); ok {
				cfg.Runtime.FunctionExternalModules = v
			}
		case "functionTimeout":
			if v, ok := parseJSInt(rawValue); ok {
				cfg.Runtime.FunctionTimeout = v
			}
		case "debugMaxLength":
			if v, ok := parseJSInt(rawValue); ok {
				cfg.Runtime.DebugMaxLength = v
			}
		case "diagnosticsEnabled":
			if v, ok := parseJSBool(rawValue); ok {
				cfg.Runtime.DiagnosticsEnabled = v
			}
		case "safeMode":
			if v, ok := parseJSBool(rawValue); ok {
				cfg.Runtime.SafeMode = v
			}
		case "nodeMessageBufferMaxLength":
			if v, ok := parseJSInt(rawValue); ok {
				cfg.Runtime.NodeMessageBufferMaxLength = v
			}
		case "externalModules":
			if extMods, ok := parseExternalModules(rawValue); ok {
				cfg.Runtime.ExternalModules = extMods
			}

		// HTTPS section
		case "https":
			if https, ok := parseHTTPS(rawValue); ok {
				cfg.HTTPS = https
			}

		// Node Reconnect section
		case "mqttReconnectTime":
			if v, ok := parseJSInt(rawValue); ok {
				cfg.NodeReconnect.MQTTReconnectTime = v
			}
		case "serialReconnectTime":
			if v, ok := parseJSInt(rawValue); ok {
				cfg.NodeReconnect.SerialReconnectTime = v
			}
		case "socketReconnectTime":
			if v, ok := parseJSInt(rawValue); ok {
				cfg.NodeReconnect.SocketReconnectTime = v
			}
		case "socketTimeout":
			if v, ok := parseJSInt(rawValue); ok {
				cfg.NodeReconnect.SocketTimeout = v
			}

		// Editor Theme section
		case "editorTheme":
			if editorTheme, ok := parseEditorTheme(rawValue); ok {
				cfg.EditorTheme = editorTheme
			}

		// Palette section
		case "paletteCategories":
			if cats := parseJSStringArray(rawValue); len(cats) > 0 {
				cfg.Palette.Categories = cats
			}
		}
	}

	// Apply defaults to fill in any zero-value fields
	cfg = model.MergeWithDefaults(cfg)

	return cfg, unrec, nil
}

// stripJSComments removes // and /* */ comments from JS content
func stripJSComments(content string) string {
	// Remove block comments /* ... */
	blockCommentRe := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	content = blockCommentRe.ReplaceAllString(content, "")

	// Remove line comments //...
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if idx := strings.Index(line, "//"); idx != -1 {
			lines[i] = line[:idx]
		}
	}
	content = strings.Join(lines, "\n")

	return content
}

// extractExportsBlock finds the module.exports = { ... } object and returns its contents (without outer braces)
func extractExportsBlock(content string) (string, error) {
	// Find "module.exports = {"
	exportsRe := regexp.MustCompile(`module\.exports\s*=\s*\{`)
	loc := exportsRe.FindStringIndex(content)
	if loc == nil {
		return "", fmt.Errorf("invalid settings.js: no module.exports found")
	}

	// Start after the opening brace
	start := loc[1]

	// Find matching closing brace
	braceCount := 0
	inString := false
	stringChar := byte(0)
	escaped := false

	for i := start; i < len(content); i++ {
		ch := content[i]

		if escaped {
			escaped = false
			continue
		}

		if ch == '\\' {
			escaped = true
			continue
		}

		if inString {
			if ch == stringChar {
				inString = false
			}
			continue
		}

		if ch == '"' || ch == '\'' {
			inString = true
			stringChar = ch
			continue
		}

		if ch == '{' {
			braceCount++
		} else if ch == '}' {
			if braceCount == 0 {
				return content[start:i], nil
			}
			braceCount--
		}
	}

	return "", fmt.Errorf("invalid settings.js: could not extract module.exports object")
}

// extractTopLevelPairs extracts top-level key: value pairs from a JS object body
// Returns a map of key -> raw_value_string
func extractTopLevelPairs(body string) map[string]string {
	pairs := make(map[string]string)

	// Use manual parsing instead of regex since Go regexp doesn't support lookahead
	pairs = parseTopLevelPairsManual(body)

	return pairs
}

// parseTopLevelPairsManual manually extracts key: value pairs from a JS object body
func parseTopLevelPairsManual(body string) map[string]string {
	pairs := make(map[string]string)
	depth := 0
	inString := false
	stringChar := byte(0)
	escaped := false

	var currentKey strings.Builder
	var currentValue strings.Builder
	parsingKey := true

	for i := 0; i < len(body); i++ {
		ch := body[i]

		if escaped {
			if !parsingKey {
				currentValue.WriteByte(ch)
			}
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			if !parsingKey {
				currentValue.WriteByte(ch)
			}
			escaped = true
			continue
		}

		if inString {
			if !parsingKey {
				currentValue.WriteByte(ch)
			}
			if ch == stringChar {
				inString = false
			}
			continue
		}

		if ch == '"' || ch == '\'' {
			inString = true
			stringChar = ch
			if !parsingKey {
				currentValue.WriteByte(ch)
			}
			continue
		}

		if parsingKey {
			if ch == ':' {
				parsingKey = false
				continue
			}
			if !strings.ContainsAny(string(ch), " \t\n\r") {
				currentKey.WriteByte(ch)
			}
			continue
		}

		// Parsing value
		if ch == '{' || ch == '[' {
			depth++
			currentValue.WriteByte(ch)
		} else if ch == '}' || ch == ']' {
			depth--
			currentValue.WriteByte(ch)
		} else if ch == ',' && depth == 0 {
			// End of this key-value pair
			key := strings.TrimSpace(currentKey.String())
			value := strings.TrimSpace(currentValue.String())
			if key != "" && value != "" {
				pairs[key] = value
			}
			currentKey.Reset()
			currentValue.Reset()
			parsingKey = true
		} else {
			currentValue.WriteByte(ch)
		}
	}

	// Don't forget the last pair
	key := strings.TrimSpace(currentKey.String())
	value := strings.TrimSpace(currentValue.String())
	if key != "" && value != "" {
		pairs[key] = value
	}

	return pairs
}

// parseJSString parses a JS string literal: "value" or 'value' → Go string
func parseJSString(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)

	// Check if it's a quoted string
	if (strings.HasPrefix(raw, "\"") && strings.HasSuffix(raw, "\"")) ||
		(strings.HasPrefix(raw, "'") && strings.HasSuffix(raw, "'")) {
		// Remove quotes
		inner := raw[1 : len(raw)-1]

		// Unescape common JS escapes
		inner = strings.ReplaceAll(inner, "\\n", "\n")
		inner = strings.ReplaceAll(inner, "\\t", "\t")
		inner = strings.ReplaceAll(inner, "\\r", "\r")
		inner = strings.ReplaceAll(inner, "\\'", "'")
		inner = strings.ReplaceAll(inner, "\\\"", "\"")
		inner = strings.ReplaceAll(inner, "\\\\", "\\")

		return inner, true
	}

	return "", false
}

// parseJSBool parses a JS boolean: true/false → bool
func parseJSBool(raw string) (bool, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "true" {
		return true, true
	}
	if raw == "false" {
		return false, true
	}
	return false, false
}

// parseJSInt parses a JS number: 1880 → int
func parseJSInt(raw string) (int, bool) {
	raw = strings.TrimSpace(raw)
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return v, true
}

// parseJSStringArray parses a JS array of strings: ["a", "b"] → []string
func parseJSStringArray(raw string) []string {
	raw = strings.TrimSpace(raw)

	if !strings.HasPrefix(raw, "[") || !strings.HasSuffix(raw, "]") {
		return nil
	}

	// Remove array brackets
	inner := raw[1 : len(raw)-1]

	var result []string
	var current strings.Builder
	inString := false
	stringChar := byte(0)
	escaped := false

	for i := 0; i < len(inner); i++ {
		ch := inner[i]

		if escaped {
			current.WriteByte(ch)
			escaped = false
			continue
		}

		if ch == '\\' && inString {
			current.WriteByte(ch)
			escaped = true
			continue
		}

		if inString {
			current.WriteByte(ch)
			if ch == stringChar {
				inString = false
			}
			continue
		}

		if ch == '"' || ch == '\'' {
			inString = true
			stringChar = ch
			current.WriteByte(ch)
			continue
		}

		if ch == ',' && !inString {
			str := strings.TrimSpace(current.String())
			if str != "" {
				if parsed, ok := parseJSString(str); ok {
					result = append(result, parsed)
				}
			}
			current.Reset()
			continue
		}

		current.WriteByte(ch)
	}

	// Don't forget the last element
	str := strings.TrimSpace(current.String())
	if str != "" {
		if parsed, ok := parseJSString(str); ok {
			result = append(result, parsed)
		}
	}

	return result
}

// extractNestedObject extracts the {...} content from a raw value string
func extractNestedObject(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)

	if !strings.HasPrefix(raw, "{") || !strings.HasSuffix(raw, "}") {
		return "", false
	}

	// Remove outer braces
	return raw[1 : len(raw)-1], true
}

// parseAdminAuth parses the adminAuth object
func parseAdminAuth(raw string) (*model.AdminAuthConfig, bool) {
	body, ok := extractNestedObject(raw)
	if !ok {
		return nil, false
	}

	pairs := extractTopLevelPairs(body)

	result := &model.AdminAuthConfig{}

	// Extract type
	if typeVal, ok := pairs["type"]; ok {
		if parsed, ok := parseJSString(typeVal); ok {
			result.Type = parsed
		}
	}

	// Extract users array
	if usersVal, ok := pairs["users"]; ok {
		result.Users = parseAdminAuthUsers(usersVal)
	}

	// Extract default
	if defaultVal, ok := pairs["default"]; ok {
		if defaultBody, ok := extractNestedObject(defaultVal); ok {
			defaultPairs := extractTopLevelPairs(defaultBody)
			if permVal, ok := defaultPairs["permissions"]; ok {
				if parsed, ok := parseJSString(permVal); ok {
					result.Default = &model.AdminAuthDefault{
						Permissions: parsed,
					}
				}
			}
		}
	}

	return result, true
}

// parseAdminAuthUsers parses an array of user objects
func parseAdminAuthUsers(raw string) []model.AdminAuthUser {
	raw = strings.TrimSpace(raw)

	if !strings.HasPrefix(raw, "[") || !strings.HasSuffix(raw, "]") {
		return nil
	}

	inner := raw[1 : len(raw)-1]

	var users []model.AdminAuthUser
	var currentObj strings.Builder
	depth := 0
	inString := false
	stringChar := byte(0)

	for i := 0; i < len(inner); i++ {
		ch := inner[i]

		if inString {
			currentObj.WriteByte(ch)
			if ch == stringChar {
				inString = false
			}
			continue
		}

		if ch == '"' || ch == '\'' {
			inString = true
			stringChar = ch
			currentObj.WriteByte(ch)
			continue
		}

		if ch == '{' {
			depth++
			currentObj.WriteByte(ch)
		} else if ch == '}' {
			depth--
			currentObj.WriteByte(ch)
			if depth == 0 {
				// Parse this user object
				objStr := strings.TrimSpace(currentObj.String())
				if objStr != "" {
					if user, ok := parseAdminAuthUser(objStr); ok {
						users = append(users, user)
					}
				}
				currentObj.Reset()
			}
		} else if ch == ',' && depth == 0 {
			// Skip commas between objects at depth 0
			continue
		} else {
			currentObj.WriteByte(ch)
		}
	}

	return users
}

// parseAdminAuthUser parses a single user object
func parseAdminAuthUser(raw string) (model.AdminAuthUser, bool) {
	body, ok := extractNestedObject(raw)
	if !ok {
		return model.AdminAuthUser{}, false
	}

	pairs := extractTopLevelPairs(body)
	user := model.AdminAuthUser{}

	if usernameVal, ok := pairs["username"]; ok {
		if parsed, ok := parseJSString(usernameVal); ok {
			user.Username = parsed
		}
	}

	if passwordVal, ok := pairs["password"]; ok {
		if parsed, ok := parseJSString(passwordVal); ok {
			user.Password = parsed
		}
	}

	if permissionsVal, ok := pairs["permissions"]; ok {
		if parsed, ok := parseJSString(permissionsVal); ok {
			user.Permissions = parsed
		}
	}

	return user, true
}

// parseHTTPNodeAuth parses the httpNodeAuth object
func parseHTTPNodeAuth(raw string) (*model.HTTPNodeAuthConfig, bool) {
	body, ok := extractNestedObject(raw)
	if !ok {
		return nil, false
	}

	pairs := extractTopLevelPairs(body)
	result := &model.HTTPNodeAuthConfig{}

	if userVal, ok := pairs["user"]; ok {
		if parsed, ok := parseJSString(userVal); ok {
			result.User = parsed
		}
	}

	if passVal, ok := pairs["pass"]; ok {
		if parsed, ok := parseJSString(passVal); ok {
			result.Pass = parsed
		}
	}

	return result, true
}

// parseContextStorage parses the contextStorage object
func parseContextStorage(raw string) (model.ContextStorageConfig, bool) {
	body, ok := extractNestedObject(raw)
	if !ok {
		return model.ContextStorageConfig{}, false
	}

	pairs := extractTopLevelPairs(body)
	result := model.ContextStorageConfig{
		Stores: make(map[string]model.ContextStoreEntry),
	}

	// Extract default
	if defaultVal, ok := pairs["default"]; ok {
		if parsed, ok := parseJSString(defaultVal); ok {
			result.Default = parsed
		}
	}

	// Extract stores
	if storesVal, ok := pairs["stores"]; ok {
		if storesBody, ok := extractNestedObject(storesVal); ok {
			storesPairs := extractTopLevelPairs(storesBody)
			for storeName, storeRaw := range storesPairs {
				if storeBody, ok := extractNestedObject(storeRaw); ok {
					storePairs := extractTopLevelPairs(storeBody)
					entry := model.ContextStoreEntry{
						Config: make(map[string]any),
					}

					if moduleVal, ok := storePairs["module"]; ok {
						if parsed, ok := parseJSString(moduleVal); ok {
							entry.Module = parsed
						}
					}

					if configVal, ok := storePairs["config"]; ok {
						if configBody, ok := extractNestedObject(configVal); ok {
							configPairs := extractTopLevelPairs(configBody)
							for cfgKey, cfgVal := range configPairs {
								// Try to parse as string first, then as other types
								if parsed, ok := parseJSString(cfgVal); ok {
									entry.Config[cfgKey] = parsed
								} else if boolVal, ok := parseJSBool(cfgVal); ok {
									entry.Config[cfgKey] = boolVal
								} else if intVal, ok := parseJSInt(cfgVal); ok {
									entry.Config[cfgKey] = intVal
								}
							}
						}
					}

					result.Stores[storeName] = entry
				}
			}
		}
	}

	return result, true
}

// parseLogging parses the logging object
func parseLogging(raw string) (model.LoggingConfig, bool) {
	body, ok := extractNestedObject(raw)
	if !ok {
		return model.LoggingConfig{}, false
	}

	pairs := extractTopLevelPairs(body)
	result := model.LoggingConfig{
		Console: model.ConsoleLogConfig{},
	}

	if consoleVal, ok := pairs["console"]; ok {
		if consoleBody, ok := extractNestedObject(consoleVal); ok {
			consolePairs := extractTopLevelPairs(consoleBody)

			if levelVal, ok := consolePairs["level"]; ok {
				if parsed, ok := parseJSString(levelVal); ok {
					result.Console.Level = parsed
				}
			}

			if metricsVal, ok := consolePairs["metrics"]; ok {
				if parsed, ok := parseJSBool(metricsVal); ok {
					result.Console.Metrics = parsed
				}
			}

			if auditVal, ok := consolePairs["audit"]; ok {
				if parsed, ok := parseJSBool(auditVal); ok {
					result.Console.Audit = parsed
				}
			}
		}
	}

	return result, true
}

// parseExternalModules parses the externalModules object
func parseExternalModules(raw string) (*model.ExternalModulesConfig, bool) {
	body, ok := extractNestedObject(raw)
	if !ok {
		return nil, false
	}

	pairs := extractTopLevelPairs(body)
	result := &model.ExternalModulesConfig{}

	if autoInstallVal, ok := pairs["autoInstall"]; ok {
		if parsed, ok := parseJSBool(autoInstallVal); ok {
			result.AutoInstall = parsed
		}
	}

	if retryVal, ok := pairs["autoInstallRetry"]; ok {
		if parsed, ok := parseJSInt(retryVal); ok {
			result.AutoInstallRetry = parsed
		}
	}

	// Parse palette
	if paletteVal, ok := pairs["palette"]; ok {
		if paletteBody, ok := extractNestedObject(paletteVal); ok {
			palettePairs := extractTopLevelPairs(paletteBody)

			palette := model.ExternalModulesPaletteConfig{
				AllowInstall: true,
				AllowUpload:  false,
				AllowList:    []string{},
				DenyList:     []string{},
			}

			if allowInstallVal, ok := palettePairs["allowInstall"]; ok {
				if parsed, ok := parseJSBool(allowInstallVal); ok {
					palette.AllowInstall = parsed
				}
			}

			if allowUploadVal, ok := palettePairs["allowUpload"]; ok {
				if parsed, ok := parseJSBool(allowUploadVal); ok {
					palette.AllowUpload = parsed
				}
			}

			if allowListVal, ok := palettePairs["allowList"]; ok {
				palette.AllowList = parseJSStringArray(allowListVal)
			}

			if denyListVal, ok := palettePairs["denyList"]; ok {
				palette.DenyList = parseJSStringArray(denyListVal)
			}

			result.Palette = palette
		}
	}

	// Parse modules
	if modulesVal, ok := pairs["modules"]; ok {
		if modulesBody, ok := extractNestedObject(modulesVal); ok {
			modulesPairs := extractTopLevelPairs(modulesBody)

			modules := model.ExternalModulesModuleConfig{
				AllowInstall: true,
				AllowList:    []string{},
				DenyList:     []string{},
			}

			if allowInstallVal, ok := modulesPairs["allowInstall"]; ok {
				if parsed, ok := parseJSBool(allowInstallVal); ok {
					modules.AllowInstall = parsed
				}
			}

			if allowListVal, ok := modulesPairs["allowList"]; ok {
				modules.AllowList = parseJSStringArray(allowListVal)
			}

			if denyListVal, ok := modulesPairs["denyList"]; ok {
				modules.DenyList = parseJSStringArray(denyListVal)
			}

			result.Modules = modules
		}
	}

	return result, true
}

// parseHTTPS parses the https object
func parseHTTPS(raw string) (model.HTTPSConfig, bool) {
	body, ok := extractNestedObject(raw)
	if !ok {
		return model.HTTPSConfig{}, false
	}

	pairs := extractTopLevelPairs(body)
	result := model.HTTPSConfig{Enabled: true}

	// Extract key
	if keyVal, ok := pairs["key"]; ok {
		if filePath := extractFilePath(keyVal); filePath != "" {
			result.KeyFile = filePath
		}
	}

	// Extract cert
	if certVal, ok := pairs["cert"]; ok {
		if filePath := extractFilePath(certVal); filePath != "" {
			result.CertFile = filePath
		}
	}

	// Extract ca
	if caVal, ok := pairs["ca"]; ok {
		if filePath := extractFilePath(caVal); filePath != "" {
			result.CAFile = filePath
		}
	}

	return result, true
}

// extractFilePath extracts the file path from fs.readFileSync("path") pattern
func extractFilePath(raw string) string {
	raw = strings.TrimSpace(raw)

	// Look for fs.readFileSync("...")
	fsRe := regexp.MustCompile(`fs\.readFileSync\(\s*['"](.*?)['"]\s*\)`)
	matches := fsRe.FindStringSubmatch(raw)
	if len(matches) > 1 {
		return matches[1]
	}

	// Also try direct string extraction if it's just a quoted path
	if parsed, ok := parseJSString(raw); ok {
		return parsed
	}

	return ""
}

// parseEditorTheme parses the editorTheme object
func parseEditorTheme(raw string) (model.EditorThemeConfig, bool) {
	body, ok := extractNestedObject(raw)
	if !ok {
		return model.EditorThemeConfig{}, false
	}

	pairs := extractTopLevelPairs(body)
	result := model.EditorThemeConfig{
		Tours:    true,
		UserMenu: true,
		Projects: model.EditorProjectsConfig{Enabled: false},
	}

	// Extract theme
	if themeVal, ok := pairs["theme"]; ok {
		if parsed, ok := parseJSString(themeVal); ok {
			result.Theme = parsed
		}
	}

	// Extract tours
	if toursVal, ok := pairs["tours"]; ok {
		if parsed, ok := parseJSBool(toursVal); ok {
			result.Tours = parsed
		}
	}

	// Extract userMenu
	if userMenuVal, ok := pairs["userMenu"]; ok {
		if parsed, ok := parseJSBool(userMenuVal); ok {
			result.UserMenu = parsed
		}
	}

	// Extract projects
	if projectsVal, ok := pairs["projects"]; ok {
		if projectsBody, ok := extractNestedObject(projectsVal); ok {
			projectsPairs := extractTopLevelPairs(projectsBody)
			if enabledVal, ok := projectsPairs["enabled"]; ok {
				if parsed, ok := parseJSBool(enabledVal); ok {
					result.Projects.Enabled = parsed
				}
			}
		}
	}

	// Extract page
	if pageVal, ok := pairs["page"]; ok {
		if pageBody, ok := extractNestedObject(pageVal); ok {
			pagePairs := extractTopLevelPairs(pageBody)
			page := &model.EditorPageConfig{}

			if titleVal, ok := pagePairs["title"]; ok {
				if parsed, ok := parseJSString(titleVal); ok {
					page.Title = parsed
				}
			}

			if faviconVal, ok := pagePairs["favicon"]; ok {
				if parsed, ok := parseJSString(faviconVal); ok {
					page.Favicon = parsed
				}
			}

			if cssVal, ok := pagePairs["css"]; ok {
				if parsed, ok := parseJSString(cssVal); ok {
					page.CSS = parsed
				}
			}

			if page.Title != "" || page.Favicon != "" || page.CSS != "" {
				result.Page = page
			}
		}
	}

	// Extract header
	if headerVal, ok := pairs["header"]; ok {
		if headerBody, ok := extractNestedObject(headerVal); ok {
			headerPairs := extractTopLevelPairs(headerBody)
			header := &model.EditorHeaderConfig{}

			if titleVal, ok := headerPairs["title"]; ok {
				if parsed, ok := parseJSString(titleVal); ok {
					header.Title = parsed
				}
			}

			if imageVal, ok := headerPairs["image"]; ok {
				if parsed, ok := parseJSString(imageVal); ok {
					header.Image = parsed
				}
			}

			if urlVal, ok := headerPairs["url"]; ok {
				if parsed, ok := parseJSString(urlVal); ok {
					header.URL = parsed
				}
			}

			if header.Title != "" || header.Image != "" || header.URL != "" {
				result.Header = header
			}
		}
	}

	// Extract deployButton
	if deployButtonVal, ok := pairs["deployButton"]; ok {
		if deployButtonBody, ok := extractNestedObject(deployButtonVal); ok {
			deployButtonPairs := extractTopLevelPairs(deployButtonBody)
			deployButton := &model.EditorDeployButtonConfig{}

			if typeVal, ok := deployButtonPairs["type"]; ok {
				if parsed, ok := parseJSString(typeVal); ok {
					deployButton.Type = parsed
				}
			}

			if labelVal, ok := deployButtonPairs["label"]; ok {
				if parsed, ok := parseJSString(labelVal); ok {
					deployButton.Label = parsed
				}
			}

			if deployButton.Type != "" || deployButton.Label != "" {
				result.DeployButton = deployButton
			}
		}
	}

	// Extract codeEditor
	if codeEditorVal, ok := pairs["codeEditor"]; ok {
		if codeEditorBody, ok := extractNestedObject(codeEditorVal); ok {
			codeEditorPairs := extractTopLevelPairs(codeEditorBody)
			codeEditor := &model.EditorCodeConfig{
				Options: make(map[string]string),
			}

			if libVal, ok := codeEditorPairs["lib"]; ok {
				if parsed, ok := parseJSString(libVal); ok {
					codeEditor.Lib = parsed
				}
			}

			if optionsVal, ok := codeEditorPairs["options"]; ok {
				if optionsBody, ok := extractNestedObject(optionsVal); ok {
					optionsPairs := extractTopLevelPairs(optionsBody)
					for optKey, optVal := range optionsPairs {
						if parsed, ok := parseJSString(optVal); ok {
							codeEditor.Options[optKey] = parsed
						}
					}
				}
			}

			if codeEditor.Lib != "" || len(codeEditor.Options) > 0 {
				result.CodeEditor = codeEditor
			}
		}
	}

	return result, true
}

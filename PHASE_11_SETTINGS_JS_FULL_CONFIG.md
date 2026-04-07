# Phase 11: Full settings.js Configuration UI

## Overview

Phase 11 extends the Node-RED Control Center to provide a comprehensive web UI for managing ALL major sections of Node-RED's `settings.js` configuration file. Currently only 5 settings are exposed; this phase expands to ~80 configurable keys across 10 sections.

## Goals

- Expose all 10 major Node-RED settings.js sections via tabbed UI
- Provide live preview of generated settings.js
- Support JSON schema validation with per-field error messages
- Enable config backup/restore via snapshots
- Allow import of existing settings.js files
- Maintain full backward compatibility with existing 5-field config

## Architecture

### Backend
- **ConfigService**: Extended with `db *sql.DB` for snapshot storage
- **FullAppConfig**: 10 nested section structs (~80 fields total)
- **Programmatic JS Builder**: `jsWriter` for safe settings.js generation (handles JS expressions like `fs.readFileSync()`)
- **Regex Importer**: Best-effort settings.js parser (no JS execution)
- **Config Snapshots**: SQLite table with 50-entry retention limit

### Frontend
- **ConfigPage**: Extracted from monolithic App.tsx
- **SettingsPanel**: Tabbed interface with 10 sections
- **LivePreviewPanel**: Debounced (800ms) settings.js preview
- **AdvancedJSONEditor**: Raw JSON editing mode
- **SnapshotPanel**: Backup/restore management
- **ImportDialog**: Paste/upload settings.js for migration

## Configurable Sections

### 1. Server & Port
| Field | Type | Default | UI Widget |
|-------|------|---------|-----------|
| `uiPort` | int | 1880 | Number input (1-65535) |
| `uiHost` | string | "0.0.0.0" | Text input |
| `httpAdminRoot` | string | "/" | Text input with "/" prefix |
| `httpNodeRoot` | string | "/" | Text input |
| `httpStatic` | string | "" | Text input (file path) |
| `disableEditor` | bool | false | Toggle with warning |

### 2. Security & Auth
| Field | Type | Default | UI Widget |
|-------|------|---------|-----------|
| `credentialSecret` | string | "" | Password input + generator |
| `sessionExpiryTime` | int64 | 86400 | Number + presets (1h/8h/24h/7d) |
| `adminAuth.type` | string | — | Select (credentials/strategy) |
| `adminAuth.users[]` | array | — | Dynamic user list (add/remove) |
| `httpNodeAuth.user` | string | — | Text input |
| `httpNodeAuth.pass` | string | — | Password input + hash generator |

### 3. Editor & Theme
| Field | Type | Default | UI Widget |
|-------|------|---------|-----------|
| `editorTheme.theme` | string | "" | Text input |
| `editorTheme.tours` | bool | true | Toggle |
| `editorTheme.userMenu` | bool | true | Toggle |
| `editorTheme.projects.enabled` | bool | false | Toggle |
| `editorTheme.codeEditor.lib` | string | "ace" | Radio (ace/monaco) |
| `editorTheme.page.title` | string | "Node-RED" | Text input |
| `editorTheme.header.title` | string | "Node-RED" | Text input |
| `editorTheme.header.url` | string | "" | URL input |
| `editorTheme.deployButton.type` | string | "simple" | Select (simple/confirm) |
| `editorTheme.deployButton.label` | string | "Deploy" | Text input |

### 4. Flows & Storage
| Field | Type | Default | UI Widget |
|-------|------|---------|-----------|
| `flowFile` | string | "flows.json" | Text input |
| `flowFilePretty` | bool | false | Toggle |
| `userDir` | string | "" | Text input (absolute path) |
| `nodesDir` | string | "" | Text input (absolute path) |

### 5. Context Storage
| Field | Type | Default | UI Widget |
|-------|------|---------|-----------|
| `contextStorage.default` | string | "default" | Select from stores |
| `contextStorage.stores` | map | memory default | Dynamic list (add/remove) |

### 6. Logging
| Field | Type | Default | UI Widget |
|-------|------|---------|-----------|
| `logging.console.level` | string | "info" | Select (6 levels) |
| `logging.console.metrics` | bool | false | Toggle |
| `logging.console.audit` | bool | false | Toggle |

### 7. Runtime
| Field | Type | Default | UI Widget |
|-------|------|---------|-----------|
| `functionExternalModules` | bool | false | Toggle |
| `functionTimeout` | int | 0 | Number input |
| `debugMaxLength` | int | 1000 | Number input |
| `diagnosticsEnabled` | bool | true | Toggle |
| `safeMode` | bool | false | Toggle with warning |
| `nodeMessageBufferMaxLength` | int | 0 | Number input |
| `externalModules.autoInstall` | bool | false | Toggle |
| `externalModules.palette.*` | various | — | Nested section |

### 8. HTTPS/SSL
| Field | Type | Default | UI Widget |
|-------|------|---------|-----------|
| `https.enabled` | bool | false | Toggle (shows/hides fields) |
| `https.keyFile` | string | "" | Path input |
| `https.certFile` | string | "" | Path input |
| `https.caFile` | string | "" | Path input (optional) |

### 9. Node Reconnect
| Field | Type | Default | UI Widget |
|-------|------|---------|-----------|
| `mqttReconnectTime` | int | 5000 | Number (ms) |
| `serialReconnectTime` | int | 5000 | Number (ms) |
| `socketReconnectTime` | int | 10000 | Number (ms) |
| `socketTimeout` | int | 120000 | Number (ms) |

### 10. Palette
| Field | Type | Default | UI Widget |
|-------|------|---------|-----------|
| `palette.categories` | []string | default order | Drag-and-drop list |

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/config` | Get full config (masked sensitive fields) |
| POST | `/api/config/validate` | Validate config with per-field errors |
| POST | `/api/config/apply` | Apply config (auto-backup + atomic write) |
| GET | `/api/config/schema` | Get JSON Schema for all fields |
| GET/POST | `/api/config/preview` | Preview rendered settings.js |
| POST | `/api/config/import` | Parse raw settings.js into config |
| POST | `/api/config/backup` | Create named config snapshot |
| GET | `/api/config/backups` | List config snapshots |
| POST | `/api/config/backups/{id}/restore` | Restore a snapshot |

## Key Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| settings.js renderer | Programmatic Go builder (`jsWriter`) | Go templates can't safely produce JS expressions |
| settings.js importer | Regex extractor (no sandboxed JS) | Best-effort, safe, no heavy dependencies |
| Backward compat | `FullAppConfig.UnmarshalJSON` dual-format | Transparent legacy support |
| ConfigService.db | Added `*sql.DB` parameter | For config_snapshots table |
| Sensitive masking | Server-layer `maskSensitiveFields()` | API never returns real secrets |
| App.tsx refactor | Extract ConfigPage only | Minimal blast radius |
| Frontend state | React Query + useState | Matches existing patterns |
| Tab routing | URL query param `?section=` | Linkable, browser history friendly |
| JSON Schema | Static hand-crafted map | Per-field metadata support |

## Implementation Phases

| Phase | Name | Tasks | Effort |
|-------|------|-------|--------|
| A | Backend Models & DB | 6 | ~12h |
| B | Validation & Defaults | 7 | ~14h |
| C | settings.js Renderer | 7 | ~13h |
| D | settings.js Importer | 5 | ~9h |
| E | ConfigService Methods | 7 | ~11h |
| F | API Routes | 9 | ~12h |
| G | DB Initialization | 1 | ~1h |
| H | Frontend Types | 2 | ~4h |
| I | Frontend Layout | 4 | ~7h |
| J | Section Components | 10 | ~15h |
| K | Preview & Editor | 3 | ~4h |
| L | Snapshots & Import | 4 | ~5h |
| M | Validation & Polish | 4 | ~6h |
| N | E2E Testing | 6 | ~6h |
| O | Unit Tests | 4 | ~5h |
| P | Docs & Cleanup | 3 | ~3h |
| **TOTAL** | | **92** | **~120h** |

## File Changes

### New Files (20)
- `internal/model/config.go` (extend with FullAppConfig)
- `internal/service/config_renderer.go`
- `internal/service/config_importer.go`
- `internal/service/config_snapshot.go`
- `internal/service/config_schema.go`
- `internal/service/config_renderer_test.go`
- `internal/service/config_importer_test.go`
- `internal/service/config_snapshot_test.go`
- `frontend/src/types/config.ts`
- `frontend/src/pages/ConfigPage.tsx`
- `frontend/src/components/settings/SettingsPanel.tsx`
- `frontend/src/components/settings/TabBar.tsx`
- `frontend/src/components/settings/sections/ServerSection.tsx`
- `frontend/src/components/settings/sections/SecuritySection.tsx`
- `frontend/src/components/settings/sections/EditorThemeSection.tsx`
- `frontend/src/components/settings/sections/FlowsSection.tsx`
- `frontend/src/components/settings/sections/ContextStorageSection.tsx`
- `frontend/src/components/settings/sections/LoggingSection.tsx`
- `frontend/src/components/settings/sections/RuntimeSection.tsx`
- `frontend/src/components/settings/sections/HTTPSSection.tsx`
- `frontend/src/components/settings/sections/NodeReconnectSection.tsx`
- `frontend/src/components/settings/sections/PaletteSection.tsx`
- `frontend/src/components/settings/LivePreviewPanel.tsx`
- `frontend/src/components/settings/AdvancedJSONEditor.tsx`
- `frontend/src/components/settings/SnapshotPanel.tsx`
- `frontend/src/components/settings/ImportDialog.tsx`

### Modified Files (7)
- `internal/model/config.go`
- `internal/service/config.go`
- `internal/service/backup.go`
- `internal/server/server.go`
- `cmd/run.go`
- `frontend/src/api.ts`
- `frontend/src/App.tsx`

## Security Considerations

- All config endpoints require authentication + CSRF
- Sensitive fields (passwords, secrets) masked with "****" in GET responses
- Sentinel detection in Apply prevents "****" from being written to disk
- File path validation rejects path traversal (`../`)
- Import uses regex only — no code execution
- Config snapshots stored in plaintext (encryption deferred to future phase)

## Success Criteria

- [ ] All 10 config sections exposed in tabbed UI
- [ ] Per-field validation with inline error messages
- [ ] Live settings.js preview (debounced 800ms)
- [ ] Config backup/restore via snapshots
- [ ] Import from existing settings.js
- [ ] Advanced JSON editor mode
- [ ] Backward compatibility with existing 5-field config
- [ ] 20+ backend unit tests passing
- [ ] Frontend builds without errors
- [ ] Go build passes

## Status

| Task | Status |
|------|--------|
| Specification | ✅ Complete |
| Design | ✅ Complete |
| Task Breakdown | ✅ Complete |
| Implementation | ⬜ Not Started |
| Testing | ⬜ Not Started |
| Documentation | ⬜ Not Started |

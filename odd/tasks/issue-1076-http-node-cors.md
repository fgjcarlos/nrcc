# Issue #1076 — Node-RED httpNodeCors as a managed setting
Source: https://github.com/fgjcarlos/nrcc/issues/1076
Base: origin/main. Local implementation only; no PR, no merge. Branch: feat/1076-http-node-cors.

## Specs

S1 — Catalog and ownership.
- Add to `nodeRED5Catalog` (`internal/service/nodered_compatibility.go`), keeping slice order = doc order:
  `{Key: "httpNodeCors", Shape: "http-node-cors", Default: "undefined", Validation: "http-node-cors", Secret: false, RestartRequired: true, UIEditable: true}`
- Mirror the row in `docs/handbook/configuration/setting-catalog.md` (enforced by `catalog_doc_sync_test.go`), plus the narrative/prose entry the neighbouring keys have.
- Add `"httpNodeCors"` to `managedSettingKeys` (`internal/service/source_patch.go`).

S2 — Model (the JSON contract both workers code against; do not deviate).
In `internal/model/config.go`, next to `HttpsConfig`:

```go
// HTTPNodeCorsConfig describes the Node-RED `httpNodeCors` block, rendered as
// an object literal in settings.js. A nil *HTTPNodeCorsConfig means the key is
// absent (CORS disabled for http-in endpoints).
type HTTPNodeCorsConfig struct {
	Origin         []string `json:"origin"`
	Methods        []string `json:"methods,omitempty"`
	Credentials    bool     `json:"credentials,omitempty"`
	AllowedHeaders []string `json:"allowedHeaders,omitempty"`
	// Unsupported marks a hand-written block whose shape NRCC cannot model
	// (function origin, regexp, unknown keys). Raw keeps its verbatim source
	// text so the block is shown read-only and never rewritten.
	Unsupported bool   `json:"unsupported,omitempty"`
	Raw         string `json:"raw,omitempty"`
}
```
and on `NodeRedConfig`, after `Https`:
```go
	// CORS for http-in endpoints (issue #1076). Absent key = disabled.
	HTTPNodeCors *HTTPNodeCorsConfig `json:"httpNodeCors,omitempty"`
```

S3 — Parse and render (`internal/service/config.go`), modeled on `parseHttpsBlockFromJS` / `renderHttpsBlock`.
- `parseHTTPNodeCorsBlockFromJS(content string) *model.HTTPNodeCorsConfig`: nil when the top-level key is absent (reuse `findTopLevelBlock`). Accept only `origin`, `methods`, `credentials`, `allowedHeaders`; `origin`/`methods`/`allowedHeaders` as a single- or double-quoted string or an array of those; `credentials` as `true`/`false`. Accepted scalars follow Node-RED: `methods` is a comma-separated string or an array; `origin` is `"*"`, one origin string, or an array.
- Any other shape (function, arrow function, `new RegExp`, regex literal, nested object, unknown key, non-literal value) ⇒ return `&model.HTTPNodeCorsConfig{Unsupported: true, Raw: <verbatim block body>}`, never nil, so the UI can show it read-only.
- `renderHTTPNodeCorsBlock(cfg model.NodeRedConfig) string`: empty string when `cfg.HTTPNodeCors == nil` **or** `Unsupported` (an unsupported block is left byte-for-byte untouched in settings.js). Otherwise emit `  httpNodeCors: {` + `\n` + one `    key: value,` line per present field + `\n  },` with `%q` for strings and `["a", "b"]` for arrays, exactly like `renderHttpsBlock` (final comma per field included; `origin` always present; `methods` only when non-empty; `credentials` only when true; `allowedHeaders` only when non-empty).
- Wire into `(*ConfigService).parseConfigFromContent` next to `cfg.Https = ...`, and into `patchSettingsJS` next to the https branch: non-empty render ⇒ `replaceBlockKey`; nil config ⇒ `removeObjectKey`; unsupported ⇒ neither (leave the source untouched).
- Keep `generateSettingsJS` unchanged (it emits no `https` either).

S4 — Validation.
`validateHTTPNodeCors(cors *model.HTTPNodeCorsConfig) error` called from `(*ConfigService).Validate`, next to `validateAdminAuth` / `validateHTTPBasicAuth` (nil or `Unsupported` ⇒ nil error). Rules, with messages in this exact shape:
- `httpNodeCors origin must not be empty`
- `httpNodeCors origin %q must be "*" or an http(s) URL without a path`
- `httpNodeCors origin "*" cannot be combined with credentials`
- `httpNodeCors method %q is not one of GET, HEAD, PUT, PATCH, POST, DELETE`
- `httpNodeCors allowedHeaders must not contain an empty header name`
An origin is valid when it is `*`, or parses with `url.Parse` where Scheme is `http`/`https`, Host is non-empty, and Path, RawQuery and Fragment are all empty. Methods must be a non-empty subset of `GET,HEAD,PUT,PATCH,POST,DELETE` (upper case, deduplicated on render). Empty `methods` means Node-RED's default, and is allowed.

S5 — Security card (frontend).
In `SecuritySettings.tsx`, a CORS card after the TLS card, driven by new form fields, with i18n keys in `frontend/src/locales/{en,es}/configuration.json`:
- enable toggle; when off, the key is absent from the payload.
- origin list editor: one origin per line (a textarea is fine), `*` allowed.
- method checkboxes for GET/HEAD/PUT/PATCH/POST/DELETE, defaulting to `GET,PUT,POST,DELETE`.
- credentials toggle.
- optional allowed-headers field (comma separated).
- a visible warning when `*` is used, and a distinct warning/disable when credentials and `*` are both set (the save is rejected by S4 either way).
- an unsupported hand-written block renders read-only: no editable controls, the raw source shown, and the value round-trips unchanged.
- Copy is English in the en catalog and Spanish in the es catalog; `catalog_parity.test.ts` must pass.

S6 — Frontend contract (`frontend/src/shared/types/index.ts`), mirroring S2:
```ts
export interface HTTPNodeCorsConfig {
  origin: string[];
  methods?: string[];
  credentials?: boolean;
  allowedHeaders?: string[];
  unsupported?: boolean;
  raw?: string;
}
```
`NodeRedConfig.httpNodeCors?: HTTPNodeCorsConfig`, plus flat form fields `httpNodeCorsEnabled?: boolean`, `httpNodeCorsOrigins?: string` (newline separated), `httpNodeCorsMethods?: string` (comma separated, default `GET,PUT,POST,DELETE`), `httpNodeCorsCredentials?: boolean`, `httpNodeCorsAllowedHeaders?: string` (comma separated), `httpNodeCorsUnsupported?: boolean`, `httpNodeCorsRaw?: string`.
`configToFormData` / `formDataToConfig` map both ways: disabled ⇒ `httpNodeCors` omitted; unsupported ⇒ `{unsupported: true, raw}` emitted unchanged; enabled ⇒ arrays back from the separated strings, `methods` omitted when it equals the default set. `formDataToConfig(configToFormData(x))` must preserve `x.httpNodeCors`.

S7 — Apply path. No new pipeline: the block flows through the existing structured apply + restart, since it is a managed key with `RestartRequired: true`. `httpNodeCors` is not a secret: do not add it to `apply_diff.go` `secretSettingKeys`.

## Non-goals
- NRCC's own API CORS (`NRCC_CORS_ORIGINS`, `NRCC_CORS_UNSAFE_WILDCARD`) stays env-only.
- No generic raw-settings.js editor.
- No new Node-RED preset for CORS.

## Tasks

- T1 [pending] S1, S2, S3, S4, S7 — backend model, catalog, ownership, parse/render, validation, tests; worker; commit.
- T2 [pending] S5, S6 — frontend types, form mapping, Security card, i18n, tests; worker; commit.
- T3 [pending] S1 — `docs/openapi.yaml` NodeRedConfig property + regenerated `frontend/src/shared/api/schema.ts`; parent; commit.
- T4 [pending] S1–S7 — verification: Go focused + suite, frontend unit + typecheck + i18n parity, independent verifier; parent.

## Log

L1 — Resuelve la issue #1076
L2 — Issue is `status:approved` (labels: enhancement, area: backend, area: frontend). Local worktree /tmp/nrcc-1076 on feat/1076-http-node-cors, branched from origin/main (c828db5).
L3 — Prior session: two backend worker handoffs stopped before any edits on an Allowed-edit-surfaces mismatch. Worktree is clean apart from this document; nothing to recover. Handoffs now list explicit repository-relative file paths.

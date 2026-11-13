# Security forms and editor theme fixes

Admin auth new users cannot be edited, HTTP auth demands a bcrypt hash instead of a
password, and editor theme images never reach Node-RED.

## Specs

- S1: "En el formulario admin auth no dme deja escribir al añadir un nuevo user en el de http auth si" — a newly added adminAuth user row accepts typing in username, password and permissions, and the values are saved.
- S2: "poner un pass hash... lo normal es poner una contraseña y nrcc que lo hashee" — httpNodeAuth / httpStaticAuth accept a plain password and NRCC stores a bcrypt hash; an existing bcrypt hash is kept as is. UI labels stop asking for a hash (adminAuth too).
- S3: "la configuracion de nodered tmb fallo desde cambiar el nombre al header, icono, etc" — editorTheme images (header image, favicon, login image, deploy button icon) uploaded through NRCC resolve in Node-RED (settings.js gets a filesystem path Node-RED can stat); the UI preview keeps working.
- S4: header title failure — root cause unknown; needs the server error from the user.

## Tasks

- T1 (S1) worker — AdminAuthBoundaryCard new-row editing + test. Commit: "fix(security): make new admin auth users editable".
- T2 (S2) worker — plain → bcrypt for HTTP basic auth (backend), labels/warnings/openapi. Commit: pending.
- T3 (S3) worker — editorTheme image paths. Commit: pending.
- T4 (S4) blocked — waiting for the error message / audit failure stage.

## Log

- L1: "En el formulario admin auth no dme deja escribir al añadir un nuevo user en el de http auth si, y poner un pass hash... lo normal es poner una contraseña y nrcc que lo hashee, la configuracion de nodered tmb fallo desde cambiar el nombre al header, icono, etc"
- L2: Exploration (gentle-ai-explore): S1 cause AdminAuthBoundaryCard.tsx:103 vs :118-143; S2 validateHTTPBasicAuth config.go:862-873 rejects plaintext, admin already hashes (label wrong es/security.json:24); S3 toEditorImageUrl strips to "/<name>" but Node-RED theme.js serveFile fs.statSync's the value.

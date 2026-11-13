# Library install activates the package in settings.js

Installing a library from the Libraries view only ran `npm install`; settings.js
was never touched, so plain npm libraries were unusable from function nodes.

## Specs

- S1: "lo de instalar librerias no funciona, nrcc no activa o no modifica el setting.js con la libreria instalada" — a successful install must update settings.js.
- S2: Activation mode chosen by the user: "Ambos" — set `functionExternalModules: true` AND expose the library with its alias in `functionGlobalContext` (`alias: require('pkg')`).
- S3: Uninstall removes the NRCC-managed `functionGlobalContext` entry of that package; operator-authored entries stay byte-stable.
- S4: The `alias` sent by the UI reaches the service (it was silently dropped); invalid aliases are rejected before npm runs.

## Tasks

- T1 (S1-S4) inline — settings transform + LibraryService hook + handler alias + server wiring + openapi. Commit: this branch, "fix(libraries): activate installed libraries in settings.js".

## Log

- L1: "Estoy probando, pero lo de instalar librerias no funciona, nrcc no activa o no modifica el setting.js con la libreria instalada"
- L2: User chose activation mode "Ambos" (functionExternalModules + functionGlobalContext with alias).
- L3: Out of scope: npm installs into dataDir while Node-RED may run with a different NODE_RED_USER_DIR (default deployments set both to /data).
- L4: Independent verify found B1 (one-line `functionGlobalContext: {}` rendered invalid JS: the entry's line comment swallowed the closing brace) and B2 (invalid package name reported INVALID_LIBRARY_ALIAS instead of INVALID_PACKAGE_NAME). Both fixed test-first (TestActivateLibraryInSettings_OneLineGlobalContext, TestLibraryServiceInstallInvalidPackageBeatsAlias).
- L5: Regenerating frontend/src/shared/api/schema.ts also picks up pre-existing openapi drift (#878 expectedRevision, https schema); it is a generated file and must match docs/openapi.yaml.

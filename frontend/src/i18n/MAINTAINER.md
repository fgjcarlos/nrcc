# i18n catalog maintenance

Catalogs in `../locales/{en,es}` are capability-owned JSON files. English is the canonical key set; every English key needs a non-empty Spanish counterpart.

## Namespace ownership

- `auth`: login, setup, users, roles, session, and access-denied copy.
- `backups`: snapshots, restore, schedules, retention, and recovery.
- `configuration`: Node-RED settings, validation, and security configuration.
- `dashboard`: overview, runtime, health, and status cards.
- `env-vars`, `files`, `flows`, `libraries`, and `updates`: their matching feature capabilities.
- `common`: only brief, generic actions and states shared by unrelated capabilities.

Never put feature headings, feature forms, validation, API errors, or complete sentences in `common`. Put cross-feature generic errors in `errors` only when they are genuinely global; otherwise use a feature path such as `backups.errors.restoreFailed`.

## Accessibility labels

Write `aria-label` values as concise action or purpose phrases: `Open user menu`, `Switch to ES`, or `Close dialog`. They must describe the control, not its implementation or visual position. Catalog them in the namespace that owns the control; use `common` only for a truly shared control.

## Plurals

i18next resolves plural variants with `_one` and `_other`. Add both variants to every supported locale and pass `count` when translating.

```json
{
  "backup_one": "{{count}} backup",
  "backup_other": "{{count}} backups"
}
```

## Scanner allowlist

The hardcoded-copy scanner permits exact product brands and stable technical identifiers, plus code-like snippets. Add an allowlist entry only for a stable identifier that must remain verbatim, such as a Node-RED setting name or filename. Do not add UI sentences, labels, or feature names to silence a finding: put those in the catalog.

## Adding a third locale

1. Copy all ten English namespace files into `../locales/<locale>/`.
2. Translate every value, retaining keys, interpolation tokens, and plural suffixes.
3. Register the locale in the i18n constants and resource configuration.
4. Extend parity tests and locale-switcher E2E coverage.
5. Run `pnpm i18n:check`, `pnpm i18n:coverage`, typecheck, and tests from `frontend/`.

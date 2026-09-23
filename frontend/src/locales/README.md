# Locale catalogs

Each locale directory contains one JSON catalog per capability namespace. English (`en`) is the canonical key set and Spanish (`es`) is maintained in lockstep; application code reads keys through i18next rather than embedding product copy in JSX.

Catalog files are named `namespace.json`, use lowercase names, and use dash-separated words for future multi-word namespaces.

To add a key, edit the matching file in both `en` and `es`, preserve interpolation tokens and plural suffixes, then run from `frontend/`:

```bash
pnpm i18n:check
pnpm i18n:coverage
```

See [`../i18n/MAINTAINER.md`](../i18n/MAINTAINER.md) for ownership and translation rules.

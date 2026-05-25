# TanStack Query keys and cache defaults

NRCC keeps TanStack Query keys in one central factory:

- Source: `frontend/src/shared/lib/queryKeys.ts`
- Import: `import { queryKeys } from '@/shared/lib/queryKeys'`

## Policy

Default query client behavior is also exported from the same file as `queryClientConfig`:

- `staleTime`: 30 seconds
- `gcTime`: 5 minutes
- `refetchOnWindowFocus`: enabled
- `retry`: 3 attempts

Hooks may override these defaults for high-frequency resources such as update progress polling.

## Invalidation guidance

Use the narrowest key that matches the mutation scope:

- Mutations that alter one feature should invalidate that feature root key.
- Dashboard views may consume shared keys such as `queryKeys.docker.status` or `queryKeys.system.info` instead of creating duplicate literals.
- Paginated backups use `queryKeys.backups.listRoot` for broad invalidation and `queryKeys.backups.list(...)` for reads.

Avoid inline array literals in hooks. Add a named key to `queryKeys` first, then use it from queries and invalidations.

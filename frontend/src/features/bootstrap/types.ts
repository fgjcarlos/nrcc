import type { HostStatus } from '@/shared/types';

// shortcut: local extension until shared types / schema.ts are regenerated from
// docs/openapi.yaml; move recommendationCodes into HostStatus then.
export type HostStatusWithCodes = HostStatus & {
  /** Stable code for each entry in `recommendations`, same order and length. */
  recommendationCodes?: string[];
};

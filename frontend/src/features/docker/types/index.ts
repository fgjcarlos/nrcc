// Re-export docker-specific types from shared
// These were originally in shared/types but are only used by docker
export type { ContainerStatus, PortMapping, ContainerInfo, ContainerState, DockerInfo } from '@/shared/types';

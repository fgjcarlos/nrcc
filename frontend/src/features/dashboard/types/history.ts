// Types for the system history and runtime history endpoints

export interface MetricsSnapshot {
  timestamp: string;
  cpuPercent: number;
  memoryPercent: number;
  diskPercent: number;
}

export interface RestartEvent {
  timestamp: string;
  exitCode: number;
  attempt: number;
  maxAttempts: number;
}

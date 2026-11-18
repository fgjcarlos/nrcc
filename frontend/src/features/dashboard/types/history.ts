// Types for the runtime history endpoint

export interface RestartEvent {
  timestamp: string;
  exitCode: number;
  attempt: number;
  maxAttempts: number;
}

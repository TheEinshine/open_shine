export interface User {
  id: number;
  name: string;
  email: string;
  createdAt: string;
}

export interface Usage {
  used: number;
  total: number;
  percent: number;
}

export interface Metrics {
  time: string;
  host: string;
  hostAvailable: boolean;
  uptimeSeconds: number;
  cpu: number;
  mem: Usage;
  disk: Usage;
  load: [number, number, number];
  go: { version: string; goroutines: number; heapBytes: number };
}

export interface MetricPoint {
  ts: string;
  cpu: number;
  mem: number;
  disk: number;
  load1: number;
}

export interface MailSettings {
  recipient: string;
  intervalMins: number;
  subject: string;
  enabled: boolean;
}

export type Metric = "cpu" | "mem" | "disk" | "load1";
export type Op = "gt" | "gte" | "lt" | "lte";

export interface Threshold {
  id: number;
  metric: Metric;
  op: Op;
  value: number;
  enabled: boolean;
}

export type TargetKind = "http" | "tcp";

export interface Target {
  id: number;
  name: string;
  kind: TargetKind;
  address: string;
  enabled: boolean;
}

export interface LogEntry {
  sentAt: string;
  status: string;
  error: string;
}

export interface Alert {
  ts: string;
  source: string;
  state: string;
  message: string;
}

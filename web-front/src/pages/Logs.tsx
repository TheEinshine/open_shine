import { useEffect, useState } from "react";
import { api } from "../api";
import type { Alert, LogEntry } from "../types";
import { fmtTime } from "../util";

export function Logs() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [err, setErr] = useState("");

  useEffect(() => {
    Promise.all([
      api.get<LogEntry[]>("/logs?limit=100"),
      api.get<Alert[]>("/alerts?limit=100"),
    ])
      .then(([l, a]) => {
        setLogs(l || []);
        setAlerts(a || []);
      })
      .catch((e) => setErr((e as Error).message));
  }, []);

  return (
    <>
      <h1 className="page-title">Logs</h1>
      <p className="page-sub">heartbeat send history · monitoring alerts</p>
      {err && <div className="error-banner">{err}</div>}

      <div className="section-head">
        <h2>Mail log</h2>
      </div>
      <div className="card" style={{ padding: 0 }}>
        <table>
          <thead>
            <tr>
              <th>When</th>
              <th>Status</th>
              <th>Detail</th>
            </tr>
          </thead>
          <tbody>
            {logs.length === 0 && (
              <tr><td colSpan={3} className="muted">No sends recorded.</td></tr>
            )}
            {logs.map((l, i) => (
              <tr key={i}>
                <td className="mono muted" style={{ whiteSpace: "nowrap" }}>{fmtTime(l.sentAt)}</td>
                <td className={`pill ${l.status}`}>{l.status}</td>
                <td className="mono muted">{l.error || "—"}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="section-head">
        <h2>Alert log</h2>
      </div>
      <div className="card" style={{ padding: 0 }}>
        <table>
          <thead>
            <tr>
              <th>When</th>
              <th>Source</th>
              <th>State</th>
              <th>Detail</th>
            </tr>
          </thead>
          <tbody>
            {alerts.length === 0 && (
              <tr><td colSpan={4} className="muted">No alerts recorded.</td></tr>
            )}
            {alerts.map((a, i) => (
              <tr key={i}>
                <td className="mono muted" style={{ whiteSpace: "nowrap" }}>{fmtTime(a.ts)}</td>
                <td className="mono">{a.source}</td>
                <td className={`pill ${a.state}`}>{a.state}</td>
                <td className="mono muted">{a.message}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}

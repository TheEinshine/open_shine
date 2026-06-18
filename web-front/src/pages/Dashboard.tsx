import { useEffect, useState } from "react";
import { api } from "../api";
import { Sparkline } from "../components/Sparkline";
import type { Alert, Metrics, MetricPoint } from "../types";
import { fmtTime, humanBytes, humanDuration, pct, usageColor } from "../util";

export function Dashboard() {
  const [m, setM] = useState<Metrics | null>(null);
  const [hist, setHist] = useState<MetricPoint[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [err, setErr] = useState("");

  useEffect(() => {
    let active = true;
    const load = async () => {
      try {
        const [mm, hh, aa] = await Promise.all([
          api.get<Metrics>("/metrics"),
          api.get<MetricPoint[]>("/metrics/history?limit=120"),
          api.get<Alert[]>("/alerts?limit=10"),
        ]);
        if (!active) return;
        setM(mm);
        setHist(hh || []);
        setAlerts(aa || []);
        setErr("");
      } catch (e) {
        if (active) setErr((e as Error).message);
      }
    };
    load();
    const id = setInterval(load, 5000);
    return () => {
      active = false;
      clearInterval(id);
    };
  }, []);

  return (
    <>
      <h1 className="page-title">Dashboard</h1>
      <p className="page-sub">{m ? `${m.host} · updated ${fmtTime(m.time)}` : "loading…"}</p>

      {err && <div className="error-banner">{err}</div>}

      {m && !m.hostAvailable && (
        <div className="ok-banner">Host metrics unavailable on this platform (non-Linux). Runtime stats only.</div>
      )}

      {m && (
        <>
          <div className="grid cols-3">
            <MetricCard
              name="CPU"
              value={pct(m.cpu)}
              percent={m.cpu}
              series={hist.map((p) => p.cpu)}
            />
            <MetricCard
              name="Memory"
              value={pct(m.mem.percent)}
              sub={`${humanBytes(m.mem.used)} / ${humanBytes(m.mem.total)}`}
              percent={m.mem.percent}
              series={hist.map((p) => p.mem)}
            />
            <MetricCard
              name="Storage"
              value={pct(m.disk.percent)}
              sub={`${humanBytes(m.disk.used)} / ${humanBytes(m.disk.total)}`}
              percent={m.disk.percent}
              series={hist.map((p) => p.disk)}
            />
          </div>

          <div className="grid cols-2" style={{ marginTop: 16 }}>
            <div className="card">
              <div className="label" style={{ marginBottom: 14 }}>System</div>
              <KV k="Load avg" v={m.load.map((l) => l.toFixed(2)).join(" · ")} />
              <KV k="Uptime" v={humanDuration(m.uptimeSeconds)} />
              <KV k="Host" v={m.host || "unknown"} />
            </div>
            <div className="card">
              <div className="label" style={{ marginBottom: 14 }}>Runtime</div>
              <KV k="Go" v={m.go.version} />
              <KV k="Goroutines" v={String(m.go.goroutines)} />
              <KV k="Heap in use" v={humanBytes(m.go.heapBytes)} />
            </div>
          </div>
        </>
      )}

      <div className="section-head">
        <h2>Recent alerts</h2>
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
              <tr>
                <td colSpan={4} className="muted">No alerts recorded.</td>
              </tr>
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

function MetricCard({
  name,
  value,
  sub,
  percent,
  series,
}: {
  name: string;
  value: string;
  sub?: string;
  percent: number;
  series: number[];
}) {
  const color = usageColor(percent);
  return (
    <div className="card metric">
      <div className="top">
        <span className="name">{name}</span>
        <span className="val">{value}</span>
      </div>
      {sub && <div className="sub">{sub}</div>}
      <div className="bar">
        <i style={{ width: `${Math.max(0, Math.min(100, percent))}%`, background: color }} />
      </div>
      <div style={{ marginTop: 12 }}>
        <Sparkline data={series} color={color} />
      </div>
    </div>
  );
}

function KV({ k, v }: { k: string; v: string }) {
  return (
    <div style={{ display: "flex", justifyContent: "space-between", padding: "7px 0" }}>
      <span className="muted">{k}</span>
      <span className="mono">{v}</span>
    </div>
  );
}

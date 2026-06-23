import { useEffect, useState, type FormEvent } from "react";
import { api } from "../api";
import type { MailSettings, Metric, Op, Target, TargetKind, Threshold } from "../types";

export function Settings() {
  const [err, setErr] = useState("");
  const [ok, setOk] = useState("");

  const flash = (msg: string) => {
    setOk(msg);
    setTimeout(() => setOk(""), 2500);
  };
  const fail = (e: unknown) => setErr((e as Error).message || "request failed");

  return (
    <>
      <h1 className="page-title">Settings</h1>
      <p className="page-sub">heartbeat email · alert thresholds · monitored targets</p>
      {err && <div className="error-banner">{err}</div>}
      {ok && <div className="ok-banner">{ok}</div>}

      <MailSection onError={fail} onOk={flash} clearError={() => setErr("")} />
      <ThresholdSection onError={fail} onOk={flash} clearError={() => setErr("")} />
      <TargetSection onError={fail} onOk={flash} clearError={() => setErr("")} />
    </>
  );
}

interface SectionProps {
  onError: (e: unknown) => void;
  onOk: (msg: string) => void;
  clearError: () => void;
}

function MailSection({ onError, onOk, clearError }: SectionProps) {
  const [s, setS] = useState<MailSettings | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    api.get<MailSettings>("/settings/mail").then(setS).catch(onError);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function save(e: FormEvent) {
    e.preventDefault();
    if (!s) return;
    clearError();
    setBusy(true);
    try {
      const saved = await api.put<MailSettings>("/settings/mail", s);
      setS(saved);
      onOk("Mail settings saved.");
    } catch (e) {
      onError(e);
    } finally {
      setBusy(false);
    }
  }

  if (!s) return null;

  return (
    <>
      <div className="section-head">
        <h2>Heartbeat email</h2>
      </div>
      <form className="card" onSubmit={save} style={{ maxWidth: 520 }}>
        <label className="field">
          <span>Recipient</span>
          <input type="email" value={s.recipient} onChange={(e) => setS({ ...s, recipient: e.target.value })} />
        </label>
        <label className="field">
          <span>Subject</span>
          <input value={s.subject} onChange={(e) => setS({ ...s, subject: e.target.value })} />
        </label>
        <label className="field">
          <span>Sender Name</span>
          <input value={s.senderName} onChange={(e) => setS({ ...s, senderName: e.target.value })} />
        </label>
        <label className="field">
          <span>Interval (minutes)</span>
          <input
            type="number"
            min={1}
            value={s.intervalMins}
            onChange={(e) => setS({ ...s, intervalMins: Number(e.target.value) })}
          />
        </label>
        <label style={{ display: "flex", gap: 8, alignItems: "center", marginBottom: 16 }}>
          <input
            type="checkbox"
            style={{ width: "auto" }}
            checked={s.enabled}
            onChange={(e) => setS({ ...s, enabled: e.target.checked })}
          />
          <span>Enabled</span>
        </label>
        <button className="primary" disabled={busy}>
          {busy ? "Saving…" : "Save"}
        </button>
      </form>
    </>
  );
}

const METRICS: Metric[] = ["cpu", "mem", "disk", "load1"];
const OPS: Op[] = ["gt", "gte", "lt", "lte"];

function ThresholdSection({ onError, onOk, clearError }: SectionProps) {
  const [list, setList] = useState<Threshold[]>([]);
  const [draft, setDraft] = useState({ metric: "cpu" as Metric, op: "gte" as Op, value: "90" });

  const reload = () => api.get<Threshold[]>("/thresholds").then((r) => setList(r || [])).catch(onError);
  useEffect(() => {
    reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function add(e: FormEvent) {
    e.preventDefault();
    clearError();
    try {
      await api.post<Threshold>("/thresholds", {
        metric: draft.metric,
        op: draft.op,
        value: Number(draft.value),
        enabled: true,
      });
      setDraft({ ...draft, value: "" });
      onOk("Threshold added.");
      reload();
    } catch (e) {
      onError(e);
    }
  }

  async function toggle(t: Threshold) {
    clearError();
    try {
      await api.put(`/thresholds/${t.id}`, { ...t, enabled: !t.enabled });
      reload();
    } catch (e) {
      onError(e);
    }
  }

  async function remove(id: number) {
    clearError();
    try {
      await api.del(`/thresholds/${id}`);
      reload();
    } catch (e) {
      onError(e);
    }
  }

  return (
    <>
      <div className="section-head">
        <h2>Alert thresholds</h2>
      </div>
      <div className="card">
        <form className="inline-form" onSubmit={add} style={{ marginBottom: list.length ? 18 : 0 }}>
          <div>
            <label className="field" style={{ margin: 0 }}>
              <span>Metric</span>
              <select value={draft.metric} onChange={(e) => setDraft({ ...draft, metric: e.target.value as Metric })}>
                {METRICS.map((m) => (
                  <option key={m} value={m}>{m}</option>
                ))}
              </select>
            </label>
          </div>
          <div>
            <label className="field" style={{ margin: 0 }}>
              <span>Operator</span>
              <select value={draft.op} onChange={(e) => setDraft({ ...draft, op: e.target.value as Op })}>
                {OPS.map((o) => (
                  <option key={o} value={o}>{o}</option>
                ))}
              </select>
            </label>
          </div>
          <div>
            <label className="field" style={{ margin: 0 }}>
              <span>Value</span>
              <input type="number" step="any" value={draft.value} onChange={(e) => setDraft({ ...draft, value: e.target.value })} required />
            </label>
          </div>
          <button className="primary">Add</button>
        </form>

        {list.length > 0 && (
          <table>
            <thead>
              <tr>
                <th>Rule</th>
                <th>Enabled</th>
                <th style={{ textAlign: "right" }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {list.map((t) => (
                <tr key={t.id}>
                  <td className="mono">{t.metric} {t.op} {t.value}</td>
                  <td className={`pill ${t.enabled ? "ok" : ""}`}>{t.enabled ? "on" : "off"}</td>
                  <td>
                    <div className="row-actions" style={{ justifyContent: "flex-end" }}>
                      <button className="ghost" onClick={() => toggle(t)}>{t.enabled ? "Disable" : "Enable"}</button>
                      <button className="ghost danger" onClick={() => remove(t.id)}>Delete</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </>
  );
}

const KINDS: TargetKind[] = ["http", "tcp"];

function TargetSection({ onError, onOk, clearError }: SectionProps) {
  const [list, setList] = useState<Target[]>([]);
  const [draft, setDraft] = useState({ name: "", kind: "http" as TargetKind, address: "" });

  const reload = () => api.get<Target[]>("/targets").then((r) => setList(r || [])).catch(onError);
  useEffect(() => {
    reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function add(e: FormEvent) {
    e.preventDefault();
    clearError();
    try {
      await api.post<Target>("/targets", { ...draft, enabled: true });
      setDraft({ name: "", kind: "http", address: "" });
      onOk("Target added.");
      reload();
    } catch (e) {
      onError(e);
    }
  }

  async function toggle(t: Target) {
    clearError();
    try {
      await api.put(`/targets/${t.id}`, { ...t, enabled: !t.enabled });
      reload();
    } catch (e) {
      onError(e);
    }
  }

  async function remove(id: number) {
    clearError();
    try {
      await api.del(`/targets/${id}`);
      reload();
    } catch (e) {
      onError(e);
    }
  }

  return (
    <>
      <div className="section-head">
        <h2>Monitored targets</h2>
      </div>
      <div className="card">
        <form className="inline-form" onSubmit={add} style={{ marginBottom: list.length ? 18 : 0 }}>
          <div>
            <label className="field" style={{ margin: 0 }}>
              <span>Name</span>
              <input value={draft.name} onChange={(e) => setDraft({ ...draft, name: e.target.value })} required />
            </label>
          </div>
          <div>
            <label className="field" style={{ margin: 0 }}>
              <span>Kind</span>
              <select value={draft.kind} onChange={(e) => setDraft({ ...draft, kind: e.target.value as TargetKind })}>
                {KINDS.map((k) => (
                  <option key={k} value={k}>{k}</option>
                ))}
              </select>
            </label>
          </div>
          <div style={{ flex: 2 }}>
            <label className="field" style={{ margin: 0 }}>
              <span>Address ({draft.kind === "http" ? "URL" : "host:port"})</span>
              <input value={draft.address} onChange={(e) => setDraft({ ...draft, address: e.target.value })} required />
            </label>
          </div>
          <button className="primary">Add</button>
        </form>

        {list.length > 0 && (
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Check</th>
                <th>Enabled</th>
                <th style={{ textAlign: "right" }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {list.map((t) => (
                <tr key={t.id}>
                  <td>{t.name}</td>
                  <td className="mono muted">{t.kind} {t.address}</td>
                  <td className={`pill ${t.enabled ? "ok" : ""}`}>{t.enabled ? "on" : "off"}</td>
                  <td>
                    <div className="row-actions" style={{ justifyContent: "flex-end" }}>
                      <button className="ghost" onClick={() => toggle(t)}>{t.enabled ? "Disable" : "Enable"}</button>
                      <button className="ghost danger" onClick={() => remove(t.id)}>Delete</button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </>
  );
}

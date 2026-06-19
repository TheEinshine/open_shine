import { useEffect, useState, type FormEvent } from "react";
import { api } from "../api";

export interface Subscriber {
  id: number;
  email: string;
  active: boolean;
  createdAt: string;
}

export function Subscribers() {
  const [list, setList] = useState<Subscriber[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");
  const [ok, setOk] = useState("");
  const [busy, setBusy] = useState(false);
  const [newEmail, setNewEmail] = useState("");
  const [confirmDelete, setConfirmDelete] = useState<number | null>(null);

  const flash = (msg: string) => {
    setOk(msg);
    setTimeout(() => setOk(""), 2500);
  };
  const fail = (e: unknown) => setErr((e as Error).message || "request failed");

  const reload = async () => {
    try {
      const data = await api.get<Subscriber[]>("/subscribers");
      setList(data || []);
    } catch (e) {
      fail(e);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function handleAdd(e: FormEvent) {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await api.post("/subscribers", { email: newEmail });
      flash("Subscriber added seamlessly.");
      setNewEmail("");
      reload();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function handleDelete(id: number) {
    setErr("");
    try {
      await api.del(`/subscribers/${id}`);
      flash("Subscriber gracefully removed.");
      reload();
    } catch (e) {
      fail(e);
    } finally {
      setConfirmDelete(null);
    }
  }

  const activeCount = list.filter(s => s.active).length;

  return (
    <>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-end", marginBottom: 32 }}>
        <div>
          <h1 className="page-title">Subscribers</h1>
          <p className="page-sub" style={{ margin: 0 }}>Grow and manage your audience.</p>
        </div>
        {!loading && (
          <div style={{ textAlign: "right", color: "var(--muted)", fontSize: 13 }}>
            <strong>{activeCount}</strong> active of {list.length} total
          </div>
        )}
      </div>

      {err && <div className="error-banner">{err}</div>}
      {ok && <div className="ok-banner">{ok}</div>}

      <div style={{ 
        background: "var(--panel-2)", 
        padding: "24px", 
        borderRadius: "12px", 
        marginBottom: "32px",
        border: "1px solid var(--border)",
        display: "flex",
        flexDirection: "column",
        gap: "16px"
      }}>
        <h3 style={{ margin: 0, fontSize: "15px", color: "var(--text)", fontWeight: 500 }}>Invite new subscriber</h3>
        <form onSubmit={handleAdd} style={{ display: "flex", gap: "12px" }}>
          <input
            type="email"
            value={newEmail}
            onChange={(e) => setNewEmail(e.target.value)}
            required
            placeholder="email@example.com"
            style={{ flex: 1, padding: "10px 16px", borderRadius: "8px", border: "1px solid var(--border)", background: "var(--panel)" }}
          />
          <button type="submit" className="primary" disabled={busy || !newEmail} style={{ padding: "0 24px", borderRadius: "8px" }}>
            {busy ? <span className="spinner"></span> : "Add to list"}
          </button>
        </form>
      </div>

      <div className="card" style={{ padding: 0, overflow: "hidden" }}>
        {loading ? (
          <div style={{ textAlign: "center", padding: "60px" }}>
            <div className="spinner"></div>
          </div>
        ) : list.length === 0 ? (
          <div className="nl-empty" style={{ padding: "60px 20px" }}>
            <div className="nl-empty-icon" style={{ fontSize: "40px", marginBottom: "16px" }}>🌱</div>
            <p className="nl-empty-title">Your list is empty</p>
            <p className="nl-empty-sub">Add your first subscriber to start growing your community.</p>
          </div>
        ) : (
          <table style={{ width: "100%", borderCollapse: "collapse" }}>
            <thead>
              <tr style={{ background: "var(--panel-2)", borderBottom: "1px solid var(--border)" }}>
                <th style={{ padding: "16px 24px", textAlign: "left", fontSize: "12px", color: "var(--muted)", fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.5px" }}>Subscriber</th>
                <th style={{ padding: "16px 24px", textAlign: "left", fontSize: "12px", color: "var(--muted)", fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.5px" }}>Status</th>
                <th style={{ padding: "16px 24px", textAlign: "left", fontSize: "12px", color: "var(--muted)", fontWeight: 600, textTransform: "uppercase", letterSpacing: "0.5px" }}>Joined</th>
                <th style={{ padding: "16px 24px", textAlign: "right" }}></th>
              </tr>
            </thead>
            <tbody>
              {list.map((sub) => (
                <tr key={sub.id} style={{ borderBottom: "1px solid var(--border)" }}>
                  <td style={{ padding: "16px 24px" }}>
                    <div style={{ display: "flex", alignItems: "center", gap: "12px" }}>
                      <div style={{ 
                        width: "32px", height: "32px", borderRadius: "50%", 
                        background: "var(--track)", display: "flex", alignItems: "center", justifyContent: "center",
                        color: "var(--muted)", fontSize: "12px", fontWeight: 600
                      }}>
                        {sub.email.charAt(0).toUpperCase()}
                      </div>
                      <span style={{ fontWeight: 500, color: "var(--text)" }}>{sub.email}</span>
                    </div>
                  </td>
                  <td style={{ padding: "16px 24px" }}>
                    <span className={`nl-status ${sub.active ? "sent" : "draft"}`}>
                      {sub.active ? "Active" : "Unsubscribed"}
                    </span>
                  </td>
                  <td style={{ padding: "16px 24px", color: "var(--muted)", fontSize: 13 }}>
                    {new Date(sub.createdAt).toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })}
                  </td>
                  <td style={{ padding: "16px 24px", textAlign: "right" }}>
                    <button
                      className="ghost danger"
                      onClick={() => setConfirmDelete(sub.id)}
                      style={{ padding: "6px 12px", fontSize: "13px" }}
                    >
                      Remove
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {confirmDelete !== null && (
        <div className="nl-confirm-overlay">
          <div className="card nl-confirm-card" style={{ padding: "32px", textAlign: "center", maxWidth: "360px" }}>
            <h3 style={{ margin: "0 0 12px 0", fontSize: "18px", color: "var(--text)" }}>Remove Subscriber?</h3>
            <p style={{ margin: "0 0 24px 0", fontSize: "14px", color: "var(--muted)", lineHeight: 1.5 }}>
              This subscriber will no longer receive your newsletters. This action cannot be undone.
            </p>
            <div style={{ display: "flex", gap: "12px", justifyContent: "center" }}>
              <button className="ghost" onClick={() => setConfirmDelete(null)} style={{ flex: 1 }}>Cancel</button>
              <button className="danger" onClick={() => handleDelete(confirmDelete)} style={{ flex: 1 }}>Remove</button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}

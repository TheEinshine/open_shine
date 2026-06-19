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
      flash("Subscriber added.");
      setNewEmail("");
      reload();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function handleDelete(id: number) {
    if (!window.confirm("Delete this subscriber?")) return;
    setErr("");
    try {
      await api.del(`/subscribers/${id}`);
      flash("Subscriber deleted.");
      reload();
    } catch (e) {
      fail(e);
    }
  }

  return (
    <>
      <h1 className="page-title">Subscribers</h1>
      <p className="page-sub">manage your newsletter audience</p>
      {err && <div className="error-banner">{err}</div>}
      {ok && <div className="ok-banner">{ok}</div>}

      <div className="card" style={{ marginBottom: 24 }}>
        <form onSubmit={handleAdd} style={{ display: "flex", gap: "12px", alignItems: "flex-end" }}>
          <label className="field" style={{ flex: 1, marginBottom: 0 }}>
            <span>Add Subscriber Email</span>
            <input
              type="email"
              value={newEmail}
              onChange={(e) => setNewEmail(e.target.value)}
              required
              placeholder="user@example.com"
            />
          </label>
          <button type="submit" className="primary" disabled={busy || !newEmail}>
            {busy ? "Adding..." : "+ Add"}
          </button>
        </form>
      </div>

      <div className="card">
        {loading ? (
          <div style={{ textAlign: "center", padding: "40px" }}>
            <div className="spinner"></div>
          </div>
        ) : list.length === 0 ? (
          <div className="nl-empty">
            <div className="nl-empty-icon">👥</div>
            <p className="nl-empty-title">No subscribers yet</p>
            <p className="nl-empty-sub">Add your first subscriber above.</p>
          </div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Email</th>
                <th>Status</th>
                <th>Joined</th>
                <th style={{ textAlign: "right" }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {list.map((sub) => (
                <tr key={sub.id}>
                  <td style={{ fontWeight: 500 }}>{sub.email}</td>
                  <td>
                    <span className={`nl-status ${sub.active ? "sent" : "draft"}`}>
                      {sub.active ? "Active" : "Inactive"}
                    </span>
                  </td>
                  <td className="muted" style={{ fontSize: 12 }}>
                    {new Date(sub.createdAt).toLocaleDateString()}
                  </td>
                  <td>
                    <div className="row-actions" style={{ justifyContent: "flex-end" }}>
                      <button
                        className="ghost danger"
                        onClick={() => handleDelete(sub.id)}
                      >
                        Delete
                      </button>
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

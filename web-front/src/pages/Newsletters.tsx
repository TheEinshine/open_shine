import { useEffect, useState, type FormEvent } from "react";
import ReactQuill from "react-quill";
import "react-quill/dist/quill.snow.css";
import { api } from "../api";
import { fmtTime } from "../util";
import type { Newsletter } from "../types";

interface FormData {
  title: string;
  subject: string;
  bodyHtml: string;
  bodyText: string;
  scheduledAt: string;
}

const EMPTY_FORM: FormData = {
  title: "",
  subject: "",
  bodyHtml: "",
  bodyText: "",
  scheduledAt: "",
};

const STATUS_CLASS: Record<Newsletter["status"], string> = {
  draft: "draft",
  scheduled: "scheduled",
  sent: "sent",
  failed: "failed",
};

const STATUS_LABEL: Record<Newsletter["status"], string> = {
  draft: "Draft",
  scheduled: "Scheduled",
  sent: "Sent",
  failed: "Failed",
};

export function Newsletters() {
  const [list, setList] = useState<Newsletter[]>([]);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState("");
  const [ok, setOk] = useState("");
  const [formOpen, setFormOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [form, setForm] = useState<FormData>({ ...EMPTY_FORM });
  const [busy, setBusy] = useState(false);
  const [confirm, setConfirm] = useState<{ type: "delete" | "send"; id: number } | null>(null);

  const flash = (msg: string) => {
    setOk(msg);
    setTimeout(() => setOk(""), 2500);
  };
  const fail = (e: unknown) => setErr((e as Error).message || "request failed");

  const reload = async () => {
    try {
      const data = await api.get<Newsletter[]>("/newsletters");
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

  function openCreate() {
    setEditingId(null);
    setForm({ ...EMPTY_FORM });
    setFormOpen(true);
    setErr("");
  }

  function openEdit(nl: Newsletter) {
    setEditingId(nl.id);
    setForm({
      title: nl.title,
      subject: nl.subject,
      bodyHtml: nl.bodyHtml,
      bodyText: nl.bodyText,
      scheduledAt: nl.scheduledAt ? toLocalInput(nl.scheduledAt) : "",
    });
    setFormOpen(true);
    setErr("");
  }

  function closeForm() {
    setFormOpen(false);
    setEditingId(null);
    setForm({ ...EMPTY_FORM });
  }

  async function handleSave(e: FormEvent, schedule: boolean) {
    e.preventDefault();
    setErr("");
    setBusy(true);
    const body = {
      title: form.title,
      subject: form.subject,
      bodyHtml: form.bodyHtml,
      bodyText: form.bodyText,
      scheduledAt: schedule && form.scheduledAt ? new Date(form.scheduledAt).toISOString() : null,
    };
    try {
      if (editingId) {
        await api.put<Newsletter>(`/newsletters/${editingId}`, body);
        flash("Newsletter updated.");
      } else {
        await api.post<Newsletter>("/newsletters", body);
        flash(schedule ? "Newsletter scheduled." : "Draft saved.");
      }
      closeForm();
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
      await api.del(`/newsletters/${id}`);
      flash("Newsletter deleted.");
      reload();
    } catch (e) {
      fail(e);
    }
    setConfirm(null);
  }

  async function handleSendNow(id: number) {
    setErr("");
    try {
      await api.post(`/newsletters/${id}/send`);
      flash("Newsletter sent!");
      reload();
    } catch (e) {
      fail(e);
    }
    setConfirm(null);
  }

  return (
    <>
      <h1 className="page-title">Newsletters</h1>
      <p className="page-sub">create · schedule · send newsletters</p>
      {err && <div className="error-banner">{err}</div>}
      {ok && <div className="ok-banner">{ok}</div>}

      <div className="section-head">
        <h2>All newsletters</h2>
        <button id="nl-new-btn" className="primary" onClick={openCreate}>
          + New Newsletter
        </button>
      </div>

      {/* Inline form */}
      <div className={`nl-form ${formOpen ? "open" : ""}`}>
        <div className="card">
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 16 }}>
            <h3 style={{ margin: 0, fontSize: 15, fontWeight: 600 }}>
              {editingId ? "Edit Newsletter" : "New Newsletter"}
            </h3>
            <button id="nl-form-close" className="ghost" onClick={closeForm} style={{ padding: "4px 10px" }}>
              ✕
            </button>
          </div>
          <form
            onSubmit={(e) => handleSave(e, false)}
            id="nl-form"
          >
            <div className="nl-form-grid">
              <label className="field">
                <span>Title</span>
                <input
                  id="nl-title"
                  value={form.title}
                  onChange={(e) => setForm({ ...form, title: e.target.value })}
                  required
                  placeholder="Monthly Update"
                />
              </label>
              <label className="field">
                <span>Subject</span>
                <input
                  id="nl-subject"
                  value={form.subject}
                  onChange={(e) => setForm({ ...form, subject: e.target.value })}
                  required
                  placeholder="Your monthly newsletter"
                />
              </label>
              <label className="field">
                <span>Schedule at</span>
                <input
                  id="nl-scheduled-at"
                  type="datetime-local"
                  className="nl-datetime"
                  value={form.scheduledAt}
                  onChange={(e) => setForm({ ...form, scheduledAt: e.target.value })}
                />
              </label>
            </div>
            <label className="field" style={{ paddingBottom: "50px" }}>
              <span>Body HTML</span>
              <ReactQuill
                theme="snow"
                value={form.bodyHtml}
                onChange={(content) => setForm({ ...form, bodyHtml: content })}
                placeholder="Write your newsletter content here..."
                style={{ height: "300px", marginBottom: "40px" }}
              />
            </label>

            <div style={{ padding: "16px", background: "var(--panel-2)", border: "1px solid var(--border)", borderRadius: "8px", marginBottom: "24px" }}>
              <h3 style={{ margin: "0 0 12px 0", fontSize: "14px", color: "var(--accent)" }}>✨ Article Curator</h3>
              <p style={{ margin: "0 0 12px 0", fontSize: "12px", color: "var(--muted)" }}>Quickly generate beautifully formatted article blocks and append them to your newsletter.</p>
              <div style={{ display: "flex", gap: "12px", marginBottom: "12px" }}>
                <input id="curate-title" placeholder="Article Title" style={{ flex: 1 }} />
                <input id="curate-url" placeholder="https://..." style={{ flex: 1 }} />
              </div>
              <textarea id="curate-desc" placeholder="Brief description..." rows={2} style={{ width: "100%", marginBottom: "12px" }} />
              <button
                type="button"
                className="secondary"
                onClick={() => {
                  const titleInput = document.getElementById("curate-title") as HTMLInputElement;
                  const urlInput = document.getElementById("curate-url") as HTMLInputElement;
                  const descInput = document.getElementById("curate-desc") as HTMLTextAreaElement;
                  const title = titleInput.value.trim();
                  const url = urlInput.value.trim();
                  const desc = descInput.value.trim();
                  
                  if (!title || !url) {
                    alert("Title and URL are required to curate an article.");
                    return;
                  }
                  
                  const block = `
                    <div style="margin-bottom: 24px; padding-bottom: 24px; border-bottom: 1px solid #34343a;">
                      <h2 style="margin: 0 0 8px 0; font-size: 18px;">
                        <a href="${url}" target="_blank" style="color: #60a5fa; text-decoration: none;">${title} &rarr;</a>
                      </h2>
                      <p style="margin: 0; color: #9a9aa0; line-height: 1.6; font-size: 14px;">${desc}</p>
                    </div>
                  `;
                  
                  setForm(prev => ({ ...prev, bodyHtml: prev.bodyHtml + block }));
                  
                  titleInput.value = "";
                  urlInput.value = "";
                  descInput.value = "";
                }}
              >
                + Append to Newsletter
              </button>
            </div>

            <label className="field">
              <span>Body Text <span className="muted">(optional plain-text fallback)</span></span>
              <textarea
                id="nl-body-text"
                className="nl-editor nl-editor-sm"
                value={form.bodyText}
                onChange={(e) => setForm({ ...form, bodyText: e.target.value })}
                placeholder="Plain text version of your newsletter..."
                rows={4}
              />
            </label>
            <div className="nl-actions">
              <button
                id="nl-save-draft"
                type="submit"
                className="primary"
                disabled={busy}
              >
                {busy ? "Saving…" : editingId ? "Update" : "Save as Draft"}
              </button>
              {form.scheduledAt && (
                <button
                  id="nl-schedule"
                  type="button"
                  className="primary nl-schedule-btn"
                  disabled={busy}
                  onClick={(e) => handleSave(e as unknown as FormEvent, true)}
                >
                  {busy ? "Scheduling…" : "Schedule"}
                </button>
              )}
              <button
                id="nl-cancel"
                type="button"
                className="ghost"
                onClick={closeForm}
              >
                Cancel
              </button>
            </div>
          </form>
        </div>
      </div>

      {/* Confirmation dialog */}
      {confirm && (
        <div className="nl-confirm-overlay">
          <div className="card nl-confirm-card">
            <p style={{ margin: "0 0 16px", fontSize: 14 }}>
              {confirm.type === "delete"
                ? "Are you sure you want to delete this newsletter? This action cannot be undone."
                : "Send this newsletter now? This will deliver it immediately."}
            </p>
            <div className="nl-actions">
              <button
                id="nl-confirm-yes"
                className={confirm.type === "delete" ? "danger" : "primary"}
                onClick={() =>
                  confirm.type === "delete"
                    ? handleDelete(confirm.id)
                    : handleSendNow(confirm.id)
                }
              >
                {confirm.type === "delete" ? "Delete" : "Send Now"}
              </button>
              <button
                id="nl-confirm-cancel"
                className="ghost"
                onClick={() => setConfirm(null)}
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Newsletter list */}
      <div className="card">
        {loading ? (
          <div style={{ textAlign: "center", padding: "40px" }}>
            <div className="spinner"></div>
          </div>
        ) : list.length === 0 ? (
          <div className="nl-empty">
            <div className="nl-empty-icon">📨</div>
            <p className="nl-empty-title">No newsletters yet</p>
            <p className="nl-empty-sub">
              Create your first newsletter to get started
            </p>
            <button id="nl-empty-create" className="primary" onClick={openCreate}>
              + New Newsletter
            </button>
          </div>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Title</th>
                <th>Subject</th>
                <th>Status</th>
                <th>Scheduled</th>
                <th>Sent</th>
                <th style={{ textAlign: "right" }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {list.map((nl) => (
                <tr key={nl.id}>
                  <td style={{ fontWeight: 500 }}>{nl.title}</td>
                  <td className="muted" style={{ maxWidth: 200, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>{nl.subject}</td>
                  <td>
                    <span className={`nl-status ${STATUS_CLASS[nl.status]}`} id={`nl-status-${nl.id}`}>
                      {STATUS_LABEL[nl.status]}
                    </span>
                  </td>
                  <td className="muted" style={{ fontSize: 12 }}>
                    {nl.scheduledAt ? fmtTime(nl.scheduledAt) : "—"}
                  </td>
                  <td className="muted" style={{ fontSize: 12 }}>
                    {nl.sentAt ? fmtTime(nl.sentAt) : "—"}
                  </td>
                  <td>
                    <div className="row-actions" style={{ justifyContent: "flex-end" }}>
                      {(nl.status === "draft" || nl.status === "scheduled") && (
                        <button
                          id={`nl-edit-${nl.id}`}
                          className="ghost"
                          onClick={() => openEdit(nl)}
                        >
                          Edit
                        </button>
                      )}
                      {(nl.status === "draft" || nl.status === "scheduled") && (
                        <button
                          id={`nl-send-${nl.id}`}
                          className="ghost"
                          style={{ color: "var(--accent)" }}
                          onClick={() => setConfirm({ type: "send", id: nl.id })}
                        >
                          Send
                        </button>
                      )}
                      <button
                        id={`nl-delete-${nl.id}`}
                        className="ghost danger"
                        onClick={() => setConfirm({ type: "delete", id: nl.id })}
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

/** Convert an ISO string to datetime-local input value */
function toLocalInput(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

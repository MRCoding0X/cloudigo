"use client";

import { useEffect, useState, useCallback } from "react";

import { apiFetchJSON, ApiError } from "@/lib/api";
import { Card, PageHeader, StatusBadge, inputClass, primaryButtonClass, secondaryButtonClass, LoadingRow } from "@/components/admin-ui";

type EmailTemplate = {
  id: string;
  type: string;
  lang: string;
  subject: string;
  body: string;
  enabled: boolean;
};

const TEMPLATE_TYPES = ["sender", "receiver", "downloaded", "destroyed", "email_verify", "password_reset"];

const PLACEHOLDER_HINTS: Record<string, string> = {
  sender: "{download_url}, {file_names}, {size}, {site_name}",
  receiver: "{email_from}, {file_names}, {size}, {message}, {download_url}, {site_name}",
  downloaded: "{file_names}, {by_email}, {site_name}",
  destroyed: "{file_names}, {site_name}",
  email_verify: "{code}, {site_name}",
  password_reset: "{reset_url}, {site_name}",
};

const emptyForm = { type: "sender", lang: "en", subject: "", body: "", enabled: true };

export default function AdminEmailTemplatesPage() {
  const [templates, setTemplates] = useState<EmailTemplate[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [form, setForm] = useState(emptyForm);

  const load = useCallback(async () => {
    try {
      const data = await apiFetchJSON<{ templates: EmailTemplate[] }>("/api/admin/email-templates");
      setTemplates(data.templates ?? []);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  function startCreate() {
    setEditingKey("new");
    setForm(emptyForm);
    setError(null);
  }

  function startEdit(t: EmailTemplate) {
    setEditingKey(`${t.type}/${t.lang}`);
    setForm({ type: t.type, lang: t.lang, subject: t.subject, body: t.body, enabled: t.enabled });
    setError(null);
  }

  async function save() {
    setError(null);
    try {
      await apiFetchJSON("/api/admin/email-templates", { method: "PUT", body: JSON.stringify(form) });
      setEditingKey(null);
      load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not save the email template");
    }
  }

  async function sendTest(t: EmailTemplate) {
    const to = prompt(`Send a test "${t.type}" (${t.lang}) email to:`);
    if (!to) return;
    try {
      const result = await apiFetchJSON<{ delivered: boolean }>("/api/admin/email-templates/test", {
        method: "POST",
        body: JSON.stringify({ type: t.type, lang: t.lang, to }),
      });
      alert(
        result.delivered
          ? `Test email sent to ${to}.`
          : `No SMTP server is configured, so the email was only logged on the server instead of actually being sent.`,
      );
    } catch (err) {
      alert(err instanceof ApiError ? err.message : "Could not send test email");
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Email templates"
        action={
          <button onClick={startCreate} className={primaryButtonClass}>
            New translation
          </button>
        }
      />

      {editingKey && (
        <Card className="flex flex-col gap-3 p-4">
          <div className="flex flex-wrap items-center gap-2">
            <select
              value={form.type}
              onChange={(e) => setForm({ ...form, type: e.target.value })}
              disabled={editingKey !== "new"}
              className={`disabled:opacity-60 ${inputClass}`}
            >
              {TEMPLATE_TYPES.map((type) => (
                <option key={type} value={type}>
                  {type}
                </option>
              ))}
            </select>
            <input
              value={form.lang}
              onChange={(e) => setForm({ ...form, lang: e.target.value })}
              placeholder="Language (e.g. en)"
              disabled={editingKey !== "new"}
              className={`w-32 disabled:opacity-60 ${inputClass}`}
            />
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={form.enabled}
                onChange={(e) => setForm({ ...form, enabled: e.target.checked })}
              />
              Enabled
            </label>
          </div>

          <p className="text-xs text-muted">Available placeholders: {PLACEHOLDER_HINTS[form.type]}</p>

          <input
            value={form.subject}
            onChange={(e) => setForm({ ...form, subject: e.target.value })}
            placeholder="Subject"
            className={inputClass}
          />
          <textarea
            value={form.body}
            onChange={(e) => setForm({ ...form, body: e.target.value })}
            placeholder="Body"
            rows={10}
            className={`font-mono ${inputClass}`}
          />

          {error && <p className="text-sm text-red-500">{error}</p>}

          <div className="flex gap-2">
            <button onClick={save} className={primaryButtonClass}>
              Save
            </button>
            <button onClick={() => setEditingKey(null)} className={secondaryButtonClass}>
              Cancel
            </button>
          </div>
        </Card>
      )}

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-surface-muted text-xs uppercase tracking-wide text-muted">
              <tr>
                <th className="p-3 font-medium">Type</th>
                <th className="p-3 font-medium">Language</th>
                <th className="p-3 font-medium">Subject</th>
                <th className="p-3 font-medium">Status</th>
                <th className="p-3"></th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <LoadingRow colSpan={5} />
              ) : templates.length === 0 ? (
                <tr>
                  <td colSpan={5} className="p-6 text-center text-muted">
                    No templates.
                  </td>
                </tr>
              ) : (
                templates.map((t) => (
                  <tr key={`${t.type}/${t.lang}`} className="border-t transition-colors hover:bg-surface-muted">
                    <td className="p-3 font-mono text-xs">{t.type}</td>
                    <td className="p-3">{t.lang}</td>
                    <td className="p-3">{t.subject}</td>
                    <td className="p-3">
                      <StatusBadge value={t.enabled ? "enabled" : "disabled"} />
                    </td>
                    <td className="p-3 flex gap-3">
                      <button onClick={() => startEdit(t)} className="hover:underline">
                        Edit
                      </button>
                      <button onClick={() => sendTest(t)} className="hover:underline">
                        Send test
                      </button>
                    </td>
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
}

"use client";

import { useEffect, useState, useCallback } from "react";

import { apiFetch, apiFetchJSON, ApiError } from "@/lib/api";
import { Card, PageHeader, inputClass, primaryButtonClass, secondaryButtonClass, LoadingRow } from "@/components/admin-ui";

type Page = { id: string; type: string; lang: string; title: string; content: string; sortOrder: number };

const emptyForm = { type: "page", lang: "en", title: "", content: "", sortOrder: 0 };

export default function AdminPagesPage() {
  const [pages, setPages] = useState<Page[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [form, setForm] = useState(emptyForm);

  const load = useCallback(async () => {
    try {
      const data = await apiFetchJSON<{ pages: Page[] }>("/api/admin/pages");
      setPages(data.pages ?? []);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  function startCreate() {
    setEditingId("new");
    setForm(emptyForm);
  }

  function startEdit(p: Page) {
    setEditingId(p.id);
    setForm({ type: p.type, lang: p.lang, title: p.title, content: p.content, sortOrder: p.sortOrder });
  }

  async function save() {
    setError(null);
    try {
      if (editingId === "new") {
        await apiFetchJSON("/api/admin/pages", { method: "POST", body: JSON.stringify(form) });
      } else if (editingId) {
        await apiFetchJSON(`/api/admin/pages/${editingId}`, { method: "PUT", body: JSON.stringify(form) });
      }
      setEditingId(null);
      load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not save the page");
    }
  }

  async function remove(id: string) {
    if (!confirm("Delete this page?")) return;
    await apiFetch(`/api/admin/pages/${id}`, { method: "DELETE" });
    load();
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader
        title="Pages"
        action={
          <button onClick={startCreate} className={primaryButtonClass}>
            New page
          </button>
        }
      />

      {editingId && (
        <Card className="flex flex-col gap-3 p-4">
          <div className="flex gap-2">
            <select value={form.type} onChange={(e) => setForm({ ...form, type: e.target.value })} className={inputClass}>
              <option value="page">page</option>
              <option value="terms_page">terms_page</option>
            </select>
            <input value={form.lang} onChange={(e) => setForm({ ...form, lang: e.target.value })} placeholder="lang (e.g. en)" className={`w-24 ${inputClass}`} />
            <input type="number" value={form.sortOrder} onChange={(e) => setForm({ ...form, sortOrder: Number(e.target.value) })} placeholder="Sort order" className={`w-28 ${inputClass}`} />
          </div>
          <input value={form.title} onChange={(e) => setForm({ ...form, title: e.target.value })} placeholder="Title" className={inputClass} />
          <textarea value={form.content} onChange={(e) => setForm({ ...form, content: e.target.value })} placeholder="Content (HTML)" rows={6} className={inputClass} />
          {error && <p className="text-sm text-red-500">{error}</p>}
          <div className="flex gap-2">
            <button onClick={save} className={primaryButtonClass}>Save</button>
            <button onClick={() => setEditingId(null)} className={secondaryButtonClass}>Cancel</button>
          </div>
        </Card>
      )}

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-surface-muted text-xs uppercase tracking-wide text-muted">
              <tr>
                <th className="p-3 font-medium">Title</th>
                <th className="p-3 font-medium">Type</th>
                <th className="p-3 font-medium">Language</th>
                <th className="p-3"></th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <LoadingRow colSpan={4} />
              ) : pages.length === 0 ? (
                <tr><td colSpan={4} className="p-6 text-center text-muted">No pages.</td></tr>
              ) : (
                pages.map((p) => (
                  <tr key={p.id} className="border-t transition-colors hover:bg-surface-muted">
                    <td className="p-3">{p.title}</td>
                    <td className="p-3">{p.type}</td>
                    <td className="p-3">{p.lang}</td>
                    <td className="p-3 flex gap-3">
                      <button onClick={() => startEdit(p)} className="hover:underline">Edit</button>
                      <button onClick={() => remove(p.id)} className="text-red-500 hover:underline">Delete</button>
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

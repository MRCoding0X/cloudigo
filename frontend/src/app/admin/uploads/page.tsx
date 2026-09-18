"use client";

import { Fragment, useEffect, useState, useCallback } from "react";

import { apiFetch, apiFetchJSON } from "@/lib/api";
import { Card, PageHeader, Pagination, StatusBadge, inputClass, secondaryButtonClass, LoadingRow } from "@/components/admin-ui";

type Upload = {
  id: string;
  uploadId: string;
  emailFrom: string;
  shareType: string;
  status: string;
  fileCount: number;
  totalSizeBytes: number;
  ip: string;
  createdAt: string;
  expiresAt: string | null;
  destruct: boolean;
  passwordProtected: boolean;
};

type Receiver = { id: string; email: string; createdAt: string };

const PAGE_SIZE = 20;

export default function AdminUploadsPage() {
  const [uploads, setUploads] = useState<Upload[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [expanded, setExpanded] = useState<string | null>(null);
  const [receivers, setReceivers] = useState<Record<string, Receiver[]>>({});
  const [receiversLoading, setReceiversLoading] = useState<string | null>(null);
  const [resendingId, setResendingId] = useState<string | null>(null);

  const load = useCallback(async () => {
    const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE), status, search });
    try {
      const data = await apiFetchJSON<{ uploads: Upload[]; total: number }>(`/api/admin/uploads?${params}`);
      setUploads(data.uploads ?? []);
      setTotal(data.total);
      setSelected(new Set());
    } finally {
      setLoading(false);
    }
  }, [page, status, search]);

  useEffect(() => {
    load();
  }, [load]);

  async function destroy(id: string) {
    if (!confirm("Permanently delete this upload?")) return;
    await apiFetch(`/api/admin/uploads/${id}/destroy`, { method: "POST" });
    load();
  }

  function toggleSelected(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function toggleSelectAll() {
    setSelected((prev) =>
      prev.size === uploads.length ? new Set() : new Set(uploads.map((u) => u.id)),
    );
  }

  async function destroySelected() {
    if (selected.size === 0) return;
    if (!confirm(`Permanently delete ${selected.size} selected upload(s)?`)) return;
    await Promise.all(
      Array.from(selected).map((id) => apiFetch(`/api/admin/uploads/${id}/destroy`, { method: "POST" })),
    );
    setSelected(new Set());
    load();
  }

  async function toggleReceivers(upload: Upload) {
    if (expanded === upload.id) {
      setExpanded(null);
      return;
    }
    setExpanded(upload.id);
    if (receivers[upload.id]) return;
    setReceiversLoading(upload.id);
    try {
      const data = await apiFetchJSON<{ receivers: Receiver[] }>(`/api/admin/uploads/${upload.id}/receivers`);
      setReceivers((prev) => ({ ...prev, [upload.id]: data.receivers ?? [] }));
    } finally {
      setReceiversLoading(null);
    }
  }

  async function resend(uploadId: string, receiverId: string) {
    setResendingId(receiverId);
    try {
      await apiFetch(`/api/admin/uploads/${uploadId}/receivers/${receiverId}/resend`, { method: "POST" });
      alert("Email resent.");
    } catch {
      alert("Could not resend the email.");
    } finally {
      setResendingId(null);
    }
  }

  // A plain <a href> can't carry the Authorization header this protected
  // endpoint needs, so fetch it via apiFetch and trigger the save manually.
  async function exportCsv() {
    const params = new URLSearchParams({ status, search });
    const res = await apiFetch(`/api/admin/uploads/export?${params}`);
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "uploads.csv";
    a.click();
    URL.revokeObjectURL(url);
  }

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Uploads"
        action={
          <div className="flex gap-2">
            {selected.size > 0 && (
              <button onClick={destroySelected} className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm font-medium text-red-500 transition-colors hover:bg-red-500/20">
                Delete selected ({selected.size})
              </button>
            )}
            <button onClick={exportCsv} className={secondaryButtonClass}>
              Export CSV
            </button>
          </div>
        }
      />

      <div className="flex gap-2">
        <select
          value={status}
          onChange={(e) => {
            setPage(1);
            setStatus(e.target.value);
          }}
          className={inputClass}
        >
          <option value="">All statuses</option>
          <option value="processing">Processing</option>
          <option value="ready">Ready</option>
          <option value="destroyed">Destroyed</option>
        </select>
        <input
          value={search}
          onChange={(e) => {
            setPage(1);
            setSearch(e.target.value);
          }}
          placeholder="Search upload ID or email…"
          className={`flex-1 ${inputClass}`}
        />
      </div>

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-surface-muted text-xs uppercase tracking-wide text-muted">
              <tr>
                <th className="w-8 p-3">
                  <input
                    type="checkbox"
                    checked={uploads.length > 0 && selected.size === uploads.length}
                    onChange={toggleSelectAll}
                    className="accent-brand"
                  />
                </th>
                <th className="p-3 font-medium">Upload ID</th>
                <th className="p-3 font-medium">Sender</th>
                <th className="p-3 font-medium">Type</th>
                <th className="p-3 font-medium">Status</th>
                <th className="p-3 font-medium">Files</th>
                <th className="p-3 font-medium">Size</th>
                <th className="p-3 font-medium">Created</th>
                <th className="p-3"></th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <LoadingRow colSpan={9} />
              ) : uploads.length === 0 ? (
                <tr>
                  <td colSpan={9} className="p-6 text-center text-muted">
                    No uploads.
                  </td>
                </tr>
              ) : (
                uploads.map((u) => (
                  <Fragment key={u.id}>
                    <tr className="border-t transition-colors hover:bg-surface-muted">
                      <td className="p-3">
                        <input
                          type="checkbox"
                          checked={selected.has(u.id)}
                          onChange={() => toggleSelected(u.id)}
                          className="accent-brand"
                        />
                      </td>
                      <td className="p-3 font-mono text-xs">{u.uploadId}</td>
                      <td className="p-3">{u.emailFrom || "—"}</td>
                      <td className="p-3">{u.shareType}</td>
                      <td className="p-3">
                        <StatusBadge value={u.status} />
                      </td>
                      <td className="p-3">{u.fileCount}</td>
                      <td className="p-3">{(u.totalSizeBytes / 1024).toFixed(1)} KB</td>
                      <td className="p-3 text-muted">{new Date(u.createdAt).toLocaleString("en-US")}</td>
                      <td className="p-3 flex gap-3">
                        {u.shareType === "mail" && (
                          <button onClick={() => toggleReceivers(u)} className="hover:underline">
                            {expanded === u.id ? "Hide receivers" : "Receivers"}
                          </button>
                        )}
                        {u.status !== "destroyed" && (
                          <button onClick={() => destroy(u.id)} className="text-red-500 hover:underline">
                            Delete
                          </button>
                        )}
                      </td>
                    </tr>
                    {expanded === u.id && (
                      <tr className="border-t bg-surface-muted/50">
                        <td colSpan={9} className="p-3">
                          {receiversLoading === u.id ? (
                            <p className="text-xs text-muted">Loading receivers…</p>
                          ) : (receivers[u.id]?.length ?? 0) === 0 ? (
                            <p className="text-xs text-muted">No receivers.</p>
                          ) : (
                            <ul className="flex flex-col gap-1.5">
                              {receivers[u.id].map((r) => (
                                <li key={r.id} className="flex items-center justify-between gap-3 text-xs">
                                  <span>{r.email}</span>
                                  <button
                                    onClick={() => resend(u.id, r.id)}
                                    disabled={resendingId === r.id}
                                    className="font-medium text-brand hover:underline disabled:opacity-50"
                                  >
                                    {resendingId === r.id ? "Sending…" : "Resend"}
                                  </button>
                                </li>
                              ))}
                            </ul>
                          )}
                        </td>
                      </tr>
                    )}
                  </Fragment>
                ))
              )}
            </tbody>
          </table>
        </div>
      </Card>

      <Pagination page={page} totalPages={totalPages} total={total} onChange={setPage} />
    </div>
  );
}

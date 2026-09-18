"use client";

import { useEffect, useState } from "react";

import { apiFetch, apiFetchJSON } from "@/lib/api";
import { Card, PageHeader, Pagination, LoadingRow, inputClass, secondaryButtonClass } from "@/components/admin-ui";

type DownloadRow = { uploadId: string; email: string; ip: string; downloadedAt: string };

const PAGE_SIZE = 30;

export default function AdminDownloadsPage() {
  const [rows, setRows] = useState<DownloadRow[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE), search });
    setLoading(true);
    apiFetchJSON<{ downloads: DownloadRow[]; total: number }>(`/api/admin/downloads?${params}`)
      .then((data) => {
        setRows(data.downloads ?? []);
        setTotal(data.total);
      })
      .finally(() => setLoading(false));
  }, [page, search]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  // A plain <a href> can't carry the Authorization header this protected
  // endpoint needs, so fetch it via apiFetch and trigger the save manually.
  async function exportCsv() {
    const params = new URLSearchParams({ search });
    const res = await apiFetch(`/api/admin/downloads/export?${params}`);
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "downloads.csv";
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Downloads"
        action={
          <button onClick={exportCsv} className={secondaryButtonClass}>
            Export CSV
          </button>
        }
      />

      <input
        value={search}
        onChange={(e) => {
          setPage(1);
          setSearch(e.target.value);
        }}
        placeholder="Search upload ID, email, or IP…"
        className={inputClass}
      />

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-surface-muted text-xs uppercase tracking-wide text-muted">
              <tr>
                <th className="p-3 font-medium">Upload ID</th>
                <th className="p-3 font-medium">Email</th>
                <th className="p-3 font-medium">IP</th>
                <th className="p-3 font-medium">Downloaded</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <LoadingRow colSpan={4} />
              ) : rows.length === 0 ? (
                <tr>
                  <td colSpan={4} className="p-6 text-center text-muted">No downloads.</td>
                </tr>
              ) : (
                rows.map((r, i) => (
                  <tr key={i} className="border-t transition-colors hover:bg-surface-muted">
                    <td className="p-3 font-mono text-xs">{r.uploadId}</td>
                    <td className="p-3">{r.email || "—"}</td>
                    <td className="p-3">{r.ip || "—"}</td>
                    <td className="p-3 text-muted">{new Date(r.downloadedAt).toLocaleString("en-US")}</td>
                  </tr>
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

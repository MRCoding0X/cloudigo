"use client";

import { useEffect, useState } from "react";

import { apiFetch, apiFetchJSON } from "@/lib/api";
import { Card, PageHeader, Pagination, LoadingRow, inputClass, secondaryButtonClass } from "@/components/admin-ui";

type AuditEntry = { id: string; actorEmail: string; eventType: string; details: string; ip: string; createdAt: string };

const PAGE_SIZE = 30;

export default function AdminAuditPage() {
  const [entries, setEntries] = useState<AuditEntry[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const params = new URLSearchParams({ page: String(page), limit: String(PAGE_SIZE), search });
    setLoading(true);
    apiFetchJSON<{ entries: AuditEntry[]; total: number }>(`/api/admin/audit?${params}`)
      .then((data) => {
        setEntries(data.entries ?? []);
        setTotal(data.total);
      })
      .finally(() => setLoading(false));
  }, [page, search]);

  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  // A plain <a href> can't carry the Authorization header this protected
  // endpoint needs, so fetch it via apiFetch and trigger the save manually.
  async function exportCsv() {
    const params = new URLSearchParams({ search });
    const res = await apiFetch(`/api/admin/audit/export?${params}`);
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "audit-log.csv";
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title="Audit log"
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
        placeholder="Search actor, event, IP, or details…"
        className={inputClass}
      />

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-surface-muted text-xs uppercase tracking-wide text-muted">
              <tr>
                <th className="p-3 font-medium">Time</th>
                <th className="p-3 font-medium">Event</th>
                <th className="p-3 font-medium">Actor</th>
                <th className="p-3 font-medium">IP</th>
                <th className="p-3 font-medium">Details</th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <LoadingRow colSpan={5} />
              ) : entries.length === 0 ? (
                <tr><td colSpan={5} className="p-6 text-center text-muted">No entries.</td></tr>
              ) : (
                entries.map((e) => (
                  <tr key={e.id} className="border-t transition-colors hover:bg-surface-muted">
                    <td className="p-3 whitespace-nowrap text-muted">{new Date(e.createdAt).toLocaleString("en-US")}</td>
                    <td className="p-3 font-mono text-xs">{e.eventType}</td>
                    <td className="p-3">{e.actorEmail || "—"}</td>
                    <td className="p-3">{e.ip || "—"}</td>
                    <td className="p-3 text-muted">{e.details || "—"}</td>
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

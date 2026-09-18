"use client";

import { useEffect, useState, useCallback } from "react";

import { apiFetch, apiFetchJSON, ApiError } from "@/lib/api";
import { Card, PageHeader, inputClass, primaryButtonClass, LoadingRow } from "@/components/admin-ui";

type BlockedIP = { id: string; ip: string; reason: string; createdAt: string };

export default function AdminBlockedIPsPage() {
  const [rows, setRows] = useState<BlockedIP[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [ip, setIp] = useState("");
  const [reason, setReason] = useState("");
  const [adding, setAdding] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await apiFetchJSON<{ blockedIps: BlockedIP[] }>("/api/admin/blocked-ips");
      setRows(data.blockedIps ?? []);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function addBlock() {
    setError(null);
    if (!ip.trim()) {
      setError("An IP address is required");
      return;
    }
    setAdding(true);
    try {
      await apiFetchJSON("/api/admin/blocked-ips", {
        method: "POST",
        body: JSON.stringify({ ip: ip.trim(), reason: reason.trim() }),
      });
      setIp("");
      setReason("");
      load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not block IP");
    } finally {
      setAdding(false);
    }
  }

  async function unblock(id: string) {
    if (!confirm("Unblock this IP address?")) return;
    await apiFetch(`/api/admin/blocked-ips/${id}`, { method: "DELETE" });
    load();
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="Blocked IPs" description="Addresses in this list can't start new uploads." />

      <Card className="flex flex-wrap items-end gap-3 p-4">
        <label className="flex flex-col gap-1 text-xs text-muted">
          IP address
          <input
            value={ip}
            onChange={(e) => setIp(e.target.value)}
            placeholder="203.0.113.42"
            className={inputClass}
          />
        </label>
        <label className="flex flex-1 flex-col gap-1 text-xs text-muted">
          Reason <span className="font-normal">(optional)</span>
          <input value={reason} onChange={(e) => setReason(e.target.value)} className={inputClass} />
        </label>
        <button onClick={addBlock} disabled={adding} className={primaryButtonClass}>
          {adding ? "Blocking…" : "Block IP"}
        </button>
      </Card>
      {error && <p className="text-sm text-red-500">{error}</p>}

      <Card className="overflow-hidden">
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead className="bg-surface-muted text-xs uppercase tracking-wide text-muted">
              <tr>
                <th className="p-3 font-medium">IP address</th>
                <th className="p-3 font-medium">Reason</th>
                <th className="p-3 font-medium">Blocked</th>
                <th className="p-3"></th>
              </tr>
            </thead>
            <tbody>
              {loading ? (
                <LoadingRow colSpan={4} />
              ) : rows.length === 0 ? (
                <tr>
                  <td colSpan={4} className="p-6 text-center text-muted">No blocked IPs.</td>
                </tr>
              ) : (
                rows.map((r) => (
                  <tr key={r.id} className="border-t transition-colors hover:bg-surface-muted">
                    <td className="p-3 font-mono text-xs">{r.ip}</td>
                    <td className="p-3 text-muted">{r.reason || "—"}</td>
                    <td className="p-3 text-muted">{new Date(r.createdAt).toLocaleString("en-US")}</td>
                    <td className="p-3">
                      <button onClick={() => unblock(r.id)} className="text-red-500 hover:underline">
                        Unblock
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

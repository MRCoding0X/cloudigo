"use client";

import { useEffect, useState } from "react";

import { apiFetchJSON } from "@/lib/api";
import { Card, PageHeader } from "@/components/admin-ui";
import { SystemIcon, DownloadsIcon, AuditIcon, DiskIcon } from "@/components/icons";

type SystemStats = {
  appVersion: string;
  goVersion: string;
  os: string;
  arch: string;
  numCpu: number;
  numGoroutine: number;
  uptimeSeconds: number;
  db: { connected: boolean; version: string; latencyMs: number; error: string };
  storage: { bytes: number; files: number };
  disk: { freeBytes: number; totalBytes: number };
};

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function formatUptime(seconds: number): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  const parts = [];
  if (d) parts.push(`${d}d`);
  if (h) parts.push(`${h}h`);
  if (m) parts.push(`${m}m`);
  parts.push(`${s}s`);
  return parts.join(" ");
}

function Panel({ icon: Icon, title, children }: { icon: React.ComponentType<{ className?: string }>; title: string; children: React.ReactNode }) {
  return (
    <Card interactive className="animate-fade-up p-5">
      <div className="mb-3 flex items-center gap-2.5">
        <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-brand text-white shadow-sm">
          <Icon />
        </div>
        <p className="font-semibold">{title}</p>
      </div>
      <dl className="flex flex-col gap-1.5 text-sm">{children}</dl>
    </Card>
  );
}

function Row({ label, value, valueClassName = "" }: { label: string; value: React.ReactNode; valueClassName?: string }) {
  return (
    <div className="flex justify-between gap-4 border-b border-dashed py-1 last:border-0">
      <dt className="text-muted">{label}</dt>
      <dd className={`text-right font-medium ${valueClassName}`}>{value}</dd>
    </div>
  );
}

export default function AdminSystemPage() {
  const [stats, setStats] = useState<SystemStats | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiFetchJSON<SystemStats>("/api/admin/system")
      .then(setStats)
      .catch(() => setError("Could not load system information"));
  }, []);

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="System info" />
      {error && <p className="text-sm text-red-500">{error}</p>}

      {stats && (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <Panel icon={SystemIcon} title="Server">
            <Row label="Cloudigo" value={stats.appVersion} />
            <Row label="Go" value={stats.goVersion} />
            <Row label="OS/Arch" value={`${stats.os}/${stats.arch}`} />
            <Row label="CPU cores" value={stats.numCpu} />
            <Row label="Goroutines" value={stats.numGoroutine} />
            <Row label="Uptime" value={formatUptime(stats.uptimeSeconds)} />
          </Panel>

          <Panel icon={AuditIcon} title="Database">
            <Row
              label="Status"
              value={stats.db.connected ? "Connected" : "Disconnected"}
              valueClassName={stats.db.connected ? "text-green-600 dark:text-green-400" : "text-red-500"}
            />
            {stats.db.connected ? (
              <>
                <Row label="Latency" value={`${stats.db.latencyMs} ms`} />
                <Row label="Version" value={stats.db.version} />
              </>
            ) : (
              <Row label="Error" value={stats.db.error} valueClassName="text-red-500" />
            )}
          </Panel>

          <Panel icon={DownloadsIcon} title="Storage">
            <Row label="Used space" value={formatBytes(stats.storage.bytes)} />
            <Row label="File count" value={stats.storage.files} />
          </Panel>

          {stats.disk.totalBytes > 0 && (() => {
            const usedBytes = stats.disk.totalBytes - stats.disk.freeBytes;
            const usedPct = (usedBytes / stats.disk.totalBytes) * 100;
            const freePct = 100 - usedPct;
            const low = freePct < 10;
            const critical = freePct < 5;
            return (
              <Panel icon={DiskIcon} title="Disk">
                <Row label="Free" value={formatBytes(stats.disk.freeBytes)} valueClassName={low ? (critical ? "text-red-500" : "text-amber-600 dark:text-amber-400") : ""} />
                <Row label="Total" value={formatBytes(stats.disk.totalBytes)} />
                <div className="mt-2">
                  <div className="h-1.5 w-full overflow-hidden rounded-full bg-surface-muted">
                    <div
                      className={`h-full transition-all ${critical ? "bg-red-500" : low ? "bg-amber-500" : "bg-brand"}`}
                      style={{ width: `${usedPct}%` }}
                    />
                  </div>
                  {low && (
                    <p className={`mt-1.5 text-xs ${critical ? "text-red-500" : "text-amber-600 dark:text-amber-400"}`}>
                      Only {freePct.toFixed(1)}% free — consider freeing up disk space.
                    </p>
                  )}
                </div>
              </Panel>
            );
          })()}
        </div>
      )}
    </div>
  );
}

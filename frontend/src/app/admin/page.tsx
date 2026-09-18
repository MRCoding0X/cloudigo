"use client";

import { useEffect, useState } from "react";

import { apiFetchJSON } from "@/lib/api";
import { Card } from "@/components/admin-ui";
import { UploadsIcon, DownloadsIcon, AuditIcon, SystemIcon } from "@/components/icons";

type Stats = {
  totalUploads: number;
  activeUploads: number;
  destroyedUploads: number;
  totalDownloads: number;
  totalStorageBytes: number;
};

type DailyStat = { day: string; uploads: number; downloads: number; storageBytes: number };

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`;
}

function dayLabel(iso: string): string {
  const d = new Date(iso + "T00:00:00");
  return d.toLocaleDateString("en-US", { month: "short", day: "numeric" });
}

function ActivityChart({ data }: { data: DailyStat[] }) {
  if (data.length === 0) return null;

  const max = Math.max(1, ...data.map((d) => Math.max(d.uploads, d.downloads)));
  const barWidth = 8;
  const barGap = 2;
  const groupGap = 6;
  const groupWidth = barWidth * 2 + barGap;
  const chartHeight = 120;
  const width = data.length * (groupWidth + groupGap);
  const labelEvery = Math.max(1, Math.ceil(data.length / 8));

  return (
    <Card className="animate-fade-up p-4" style={{ animationDelay: "300ms" }}>
      <div className="mb-4 flex items-center justify-between">
        <h2 className="text-sm font-medium">Activity — last {data.length} days</h2>
        <div className="flex items-center gap-4 text-xs text-muted">
          <span className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-brand" /> Uploads
          </span>
          <span className="flex items-center gap-1.5">
            <span className="h-2 w-2 rounded-full bg-brand/30" /> Downloads
          </span>
        </div>
      </div>
      <div className="overflow-x-auto">
        <svg width={Math.max(width, 480)} height={chartHeight + 24} role="img" aria-label="Uploads and downloads per day">
          {data.map((d, i) => {
            const x = i * (groupWidth + groupGap);
            const uploadHeight = (d.uploads / max) * chartHeight;
            const downloadHeight = (d.downloads / max) * chartHeight;
            return (
              <g key={d.day}>
                <title>{`${d.day}: ${d.uploads} uploads, ${d.downloads} downloads, ${formatBytes(d.storageBytes)} added`}</title>
                <rect x={x} y={chartHeight - uploadHeight} width={barWidth} height={Math.max(uploadHeight, d.uploads > 0 ? 2 : 0)} rx={1.5} className="fill-brand" />
                <rect x={x + barWidth + barGap} y={chartHeight - downloadHeight} width={barWidth} height={Math.max(downloadHeight, d.downloads > 0 ? 2 : 0)} rx={1.5} className="fill-brand/30" />
                {i % labelEvery === 0 && (
                  <text x={x + groupWidth / 2} y={chartHeight + 16} textAnchor="middle" className="fill-muted text-[10px]">
                    {dayLabel(d.day)}
                  </text>
                )}
              </g>
            );
          })}
        </svg>
      </div>
    </Card>
  );
}

export default function AdminDashboardPage() {
  const [stats, setStats] = useState<Stats | null>(null);
  const [dailyStats, setDailyStats] = useState<DailyStat[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    apiFetchJSON<Stats>("/api/admin/dashboard")
      .then(setStats)
      .catch(() => setError("Could not load statistics"));
    apiFetchJSON<{ days: DailyStat[] }>("/api/admin/dashboard/timeseries?days=30")
      .then((data) => setDailyStats(data.days ?? []))
      .catch(() => {});
  }, []);

  const cards = stats
    ? [
        { label: "Total uploads", value: stats.totalUploads, icon: UploadsIcon },
        { label: "Active uploads", value: stats.activeUploads, icon: UploadsIcon },
        { label: "Deleted uploads", value: stats.destroyedUploads, icon: AuditIcon },
        { label: "Total downloads", value: stats.totalDownloads, icon: DownloadsIcon },
        { label: "Total storage (active)", value: formatBytes(stats.totalStorageBytes), icon: SystemIcon },
      ]
    : [];

  return (
    <div className="flex flex-col gap-6">
      <div className="animate-fade-up">
        <h1 className="text-2xl font-semibold">Dashboard</h1>
        <p className="mt-1 text-sm text-muted">An overview of activity across your Cloudigo instance.</p>
      </div>
      {error && <p className="text-sm text-red-500">{error}</p>}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
        {cards.map((c, i) => (
          <Card
            key={c.label}
            interactive
            className="animate-fade-up p-4"
            style={{ animationDelay: `${i * 60}ms` }}
          >
            <div className="mb-3 flex h-9 w-9 items-center justify-center rounded-lg bg-brand text-white shadow-sm">
              <c.icon />
            </div>
            <p className="text-2xl font-semibold tracking-tight">{c.value}</p>
            <p className="text-sm text-muted">{c.label}</p>
          </Card>
        ))}
      </div>
      <ActivityChart data={dailyStats} />
      <p className="text-sm text-muted">New uploads, deletions and downloads appear live at the bottom of the page.</p>
    </div>
  );
}

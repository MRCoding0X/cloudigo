"use client";

import { useEffect, useRef, useState, useCallback } from "react";

import { apiFetch } from "@/lib/api";
import { Card, PageHeader, inputClass, primaryButtonClass, secondaryButtonClass } from "@/components/admin-ui";
import { Spinner } from "@/components/icons";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

type Background = { id: string; url: string; durationSeconds: number | null; fileUrl: string };

export default function AdminBackgroundsPage() {
  const [backgrounds, setBackgrounds] = useState<Background[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [clickUrl, setClickUrl] = useState("");
  const [duration, setDuration] = useState("");
  const [fileName, setFileName] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch("/api/admin/backgrounds");
      const data = await res.json();
      setBackgrounds(data.backgrounds ?? []);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  async function upload() {
    setError(null);
    const file = fileInputRef.current?.files?.[0];
    if (!file) {
      setError("Choose an image file");
      return;
    }
    const formData = new FormData();
    formData.append("file", file);
    formData.append("url", clickUrl);
    if (duration) formData.append("durationSeconds", duration);

    const res = await apiFetch("/api/admin/backgrounds", { method: "POST", body: formData });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      setError(body.error ?? "Could not upload background");
      return;
    }
    setClickUrl("");
    setDuration("");
    setFileName(null);
    if (fileInputRef.current) fileInputRef.current.value = "";
    load();
  }

  async function remove(id: string) {
    if (!confirm("Delete this background image?")) return;
    await apiFetch(`/api/admin/backgrounds/${id}`, { method: "DELETE" });
    load();
  }

  return (
    <div className="flex flex-col gap-6">
      <PageHeader title="Backgrounds" />

      <Card className="flex flex-wrap items-end gap-3 p-4">
        <div className="flex flex-col gap-1 text-xs text-muted">
          Image file
          <div className="flex items-center gap-2">
            <input
              ref={fileInputRef}
              type="file"
              accept="image/*"
              className="hidden"
              onChange={(e) => setFileName(e.target.files?.[0]?.name ?? null)}
            />
            <button type="button" onClick={() => fileInputRef.current?.click()} className={secondaryButtonClass}>
              Choose file
            </button>
            <span className="max-w-40 truncate text-foreground">{fileName ?? "No file chosen"}</span>
          </div>
        </div>
        <label className="flex flex-col gap-1 text-xs text-muted">
          Click URL (optional)
          <input value={clickUrl} onChange={(e) => setClickUrl(e.target.value)} className={inputClass} />
        </label>
        <label className="flex flex-col gap-1 text-xs text-muted">
          Display duration (sec, optional)
          <input value={duration} onChange={(e) => setDuration(e.target.value)} className={`w-32 ${inputClass}`} />
        </label>
        <button onClick={upload} className={primaryButtonClass}>
          Upload
        </button>
      </Card>
      {error && <p className="text-sm text-red-500">{error}</p>}

      {loading ? (
        <div className="flex justify-center p-8 text-muted">
          <Spinner />
        </div>
      ) : backgrounds.length === 0 ? (
        <p className="text-sm text-muted">No background images.</p>
      ) : (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-4">
          {backgrounds.map((bg) => (
            <Card key={bg.id} interactive className="overflow-hidden">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src={`${API_URL}${bg.fileUrl}`} alt="" className="h-32 w-full object-cover transition-transform duration-500 hover:scale-105" />
              <div className="flex items-center justify-between p-2.5 text-xs">
                <span className="truncate text-muted">{bg.url || "no URL"}</span>
                <button onClick={() => remove(bg.id)} className="shrink-0 text-red-500 hover:underline">Delete</button>
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}

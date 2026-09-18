"use client";

import { useEffect, useState } from "react";
import QRCode from "qrcode";

import { useUploadSocket, type UploadEvent } from "@/lib/ws";
import LockGate from "@/components/lock-gate";
import { useSettings } from "@/lib/settings-context";
import { FileTypeIcon } from "@/components/file-type-icon";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

type FileInfo = { id: string; fileName: string; sizeBytes: number; hasThumbnail: boolean };

type DownloadInfo = {
  status: string;
  shareType: "link" | "mail";
  fileCount: number;
  totalSizeBytes: number;
  files: FileInfo[];
  requiresPassword: boolean;
  isOwner: boolean;
  emailFrom: string;
  message: string;
  expiresAt: string | null;
  shareCode: string;
};

const EXPIRE_OPTIONS = [
  { label: "Keep current expiry", value: "" },
  { label: "1 hour from now", value: String(60 * 60) },
  { label: "1 day from now", value: String(60 * 60 * 24) },
  { label: "1 week from now", value: String(60 * 60 * 24 * 7) },
  { label: "Never", value: "0" },
];

type ViewState = "loading" | "not-found" | "gone" | "ready" | "destroyed";

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function TrashIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 6h18" />
      <path d="M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2m3 0-1 14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2L4 6" />
      <path d="M10 11v6M14 11v6" />
    </svg>
  );
}

function CopyIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="9" y="9" width="13" height="13" rx="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </svg>
  );
}

function QrIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="3" y="3" width="7" height="7" rx="1" />
      <rect x="14" y="3" width="7" height="7" rx="1" />
      <rect x="3" y="14" width="7" height="7" rx="1" />
      <path d="M14 14h3v3h-3zM20 14v3M14 20h3M20 20v.01" />
    </svg>
  );
}

function PencilIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
    </svg>
  );
}

function formatExpiry(expiresAt: string | null): string {
  if (!expiresAt) return "Never";
  return new Date(expiresAt).toLocaleString("en-US");
}

function DownloadView({ uploadId, code }: { uploadId: string; code: string }) {
  const settings = useSettings();
  const [view, setView] = useState<ViewState>("loading");
  const [info, setInfo] = useState<DownloadInfo | null>(null);
  const [password, setPassword] = useState("");
  const [passwordError, setPasswordError] = useState<string | null>(null);
  const [ticket, setTicket] = useState<string | null>(null);
  const [activity, setActivity] = useState<string[]>([]);
  const [confirmingDelete, setConfirmingDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const [linkCopied, setLinkCopied] = useState(false);
  const [showQr, setShowQr] = useState(false);
  const [qrDataUrl, setQrDataUrl] = useState<string | null>(null);

  const [editingSettings, setEditingSettings] = useState(false);
  const [removePassword, setRemovePassword] = useState(false);
  const [newPassword, setNewPassword] = useState("");
  const [newExpireSeconds, setNewExpireSeconds] = useState("");
  const [savingSettings, setSavingSettings] = useState(false);
  const [settingsError, setSettingsError] = useState<string | null>(null);
  const [settingsSaved, setSettingsSaved] = useState(false);

  useEffect(() => {
    fetch(`${API_URL}/api/download/${uploadId}/${code}`)
      .then(async (res) => {
        if (res.status === 404) return setView("not-found");
        if (res.status === 410) return setView("gone");
        if (!res.ok) return setView("not-found");
        const data: DownloadInfo = await res.json();
        setInfo(data);
        setView("ready");
      })
      .catch(() => setView("not-found"));
  }, [uploadId, code]);

  useUploadSocket(view === "ready" ? uploadId : null, (event: UploadEvent) => {
    if (event.type === "download.happened") {
      const time = new Date(event.downloadedAt).toLocaleTimeString("en-US");
      setActivity((prev) => [
        ...prev,
        event.email ? `Downloaded at ${time} by ${event.email}` : `Downloaded at ${time}`,
      ]);
    } else if (event.type === "upload.destroyed") {
      setView("destroyed");
    }
  });

  async function submitPassword() {
    setPasswordError(null);
    const res = await fetch(`${API_URL}/api/download/${uploadId}/${code}/ticket`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ password }),
    });
    if (!res.ok) {
      const body = await res.json().catch(() => ({}));
      setPasswordError(body.error ?? "Incorrect password");
      return;
    }
    const data = await res.json();
    setTicket(data.ticket);
  }

  async function deleteUpload() {
    setDeleting(true);
    setDeleteError(null);
    try {
      const res = await fetch(`${API_URL}/api/download/${uploadId}/${code}`, { method: "DELETE" });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        setDeleteError(body.error ?? "Could not delete this upload");
        setDeleting(false);
        return;
      }
      setView("destroyed");
    } catch {
      setDeleteError("Could not delete this upload");
      setDeleting(false);
    }
  }

  async function saveSettings() {
    setSavingSettings(true);
    setSettingsError(null);
    try {
      const body: { password?: string; expireSeconds?: number } = {};
      if (removePassword) body.password = "";
      else if (newPassword) body.password = newPassword;
      if (newExpireSeconds !== "") body.expireSeconds = Number(newExpireSeconds);

      const res = await fetch(`${API_URL}/api/download/${uploadId}/${code}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const errBody = await res.json().catch(() => ({}));
        setSettingsError(errBody.error ?? "Could not save changes");
        return;
      }
      const updated = await res.json();
      setInfo((prev) => (prev ? { ...prev, requiresPassword: updated.requiresPassword, expiresAt: updated.expiresAt } : prev));
      setEditingSettings(false);
      setRemovePassword(false);
      setNewPassword("");
      setNewExpireSeconds("");
      setSettingsSaved(true);
      setTimeout(() => setSettingsSaved(false), 2500);
    } catch {
      setSettingsError("Could not save changes");
    } finally {
      setSavingSettings(false);
    }
  }

  const shareLink =
    info?.shareCode && typeof window !== "undefined" ? `${window.location.origin}/${uploadId}/${info.shareCode}` : null;

  useEffect(() => {
    if (!showQr || qrDataUrl || !shareLink) return;
    QRCode.toDataURL(shareLink, { margin: 1, width: 200 })
      .then(setQrDataUrl)
      .catch(() => {});
  }, [showQr, qrDataUrl, shareLink]);

  function copyShareLink() {
    if (!shareLink) return;
    navigator.clipboard.writeText(shareLink);
    setLinkCopied(true);
    setTimeout(() => setLinkCopied(false), 1500);
  }

  const needsPassword = info?.requiresPassword && !ticket;
  const fileUrl = (fileId?: string) => {
    const base = fileId
      ? `${API_URL}/api/download/${uploadId}/${code}/thumb/${fileId}`
      : `${API_URL}/api/download/${uploadId}/${code}/file`;
    return ticket ? `${base}?ticket=${encodeURIComponent(ticket)}` : base;
  };

  if (view === "loading") {
    return <main className="flex flex-1 items-center justify-center p-8 text-sm text-muted">Loading…</main>;
  }
  if (view === "not-found") {
    return <main className="flex flex-1 items-center justify-center p-8 text-sm text-muted">Upload not found.</main>;
  }
  if (view === "gone" || view === "destroyed") {
    return <main className="flex flex-1 items-center justify-center p-8 text-sm text-muted">This file is no longer available.</main>;
  }
  if (!info) return null;

  return (
    <main className="relative flex flex-1 items-center justify-center overflow-hidden p-6 sm:p-8">
      <div className="pointer-events-none absolute -top-28 -left-28 h-72 w-72 animate-float-slow rounded-full bg-brand/20 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-28 -right-28 h-72 w-72 animate-float rounded-full bg-brand/12 blur-3xl" />
      <div className="animate-fade-up relative w-full max-w-lg">
        <div className="flex flex-col gap-5 rounded-2xl border bg-surface p-6 shadow-sm sm:p-8">
          <div className="text-center">
            <h1 className="text-xl font-semibold">
              {info.fileCount} {info.fileCount === 1 ? "file" : "files"} · {formatBytes(info.totalSizeBytes)}
            </h1>
            {info.message && <p className="mt-2 text-sm text-muted">{info.message}</p>}
          </div>

          {needsPassword ? (
            <div className="flex flex-col gap-3 border-t pt-5">
              <p className="text-sm">This file is password protected.</p>
              <input
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Password"
                className="rounded-lg border bg-surface px-3 py-2.5 text-sm transition-colors focus:border-brand"
              />
              {passwordError && <p className="text-sm text-red-500">{passwordError}</p>}
              <button
                onClick={submitPassword}
                className="rounded-lg bg-brand px-4 py-2.5 text-sm font-medium text-white shadow-sm shadow-brand/20 transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-brand/30 active:translate-y-0 active:scale-[0.98]"
              >
                Unlock
              </button>
            </div>
          ) : (
            <>
              <ul className="flex flex-col gap-2">
                {info.files.map((f) => (
                  <li key={f.id} className="flex items-center gap-3 rounded-lg border bg-surface-muted px-3 py-2 text-sm transition-colors hover:bg-surface">
                    {f.hasThumbnail ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={fileUrl(f.id)} alt="" className="h-9 w-9 shrink-0 rounded object-cover" />
                    ) : (
                      <FileTypeIcon fileName={f.fileName} />
                    )}
                    <span className="flex-1 truncate">{f.fileName}</span>
                    <span className="shrink-0 text-xs text-muted">{formatBytes(f.sizeBytes)}</span>
                  </li>
                ))}
              </ul>

              <a
                href={fileUrl()}
                className="rounded-lg bg-brand px-4 py-3 text-center text-sm font-medium text-white shadow-sm shadow-brand/20 transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-brand/30 active:translate-y-0 active:scale-[0.98]"
              >
                {info.fileCount > 1 ? "Download (zip)" : "Download"}
              </a>

              {info.isOwner && shareLink && (
                <div className="border-t pt-4">
                  <p className="text-xs font-medium text-muted">Share link</p>
                  <div className="mt-1.5 flex items-center gap-2 rounded-lg border bg-surface-muted p-2.5 pl-3">
                    <code className="flex-1 truncate text-left text-sm">{shareLink}</code>
                    <button
                      onClick={copyShareLink}
                      className="flex shrink-0 items-center gap-1.5 rounded-md border bg-surface px-2.5 py-1.5 text-xs font-medium transition-colors hover:bg-surface-muted"
                    >
                      <CopyIcon />
                      {linkCopied ? "Copied" : "Copy"}
                    </button>
                  </div>
                  <button
                    onClick={() => setShowQr((v) => !v)}
                    className="mt-2 flex items-center gap-1.5 text-xs text-muted transition-colors hover:text-foreground"
                  >
                    <QrIcon />
                    {showQr ? "Hide QR code" : "Show QR code"}
                  </button>
                  {showQr && (
                    <div className="animate-fade-up mt-2 inline-block rounded-xl bg-white p-3 shadow-sm">
                      {qrDataUrl ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img src={qrDataUrl} alt="QR code for the share link" className="h-[180px] w-[180px]" />
                      ) : (
                        <div className="h-[180px] w-[180px]" />
                      )}
                    </div>
                  )}
                  <p className="mt-1.5 text-xs text-muted">
                    Safe to share with anyone — they can view and download, but not manage this upload.
                  </p>
                </div>
              )}

              {info.isOwner && (
                <div className="rounded-lg border bg-surface-muted p-3 text-sm">
                  <p className="font-medium">Live activity</p>
                  {activity.length === 0 ? (
                    <p className="text-muted">No one has downloaded this yet.</p>
                  ) : (
                    <ul className="mt-1 space-y-1 text-muted">
                      {activity.map((a, i) => (
                        <li key={i}>{a}</li>
                      ))}
                    </ul>
                  )}
                </div>
              )}

              {info.isOwner && (
                <div className="border-t pt-4">
                  {editingSettings ? (
                    <div className="flex flex-col gap-3 rounded-lg border bg-surface-muted p-3">
                      <div>
                        <p className="text-sm font-medium">Expiry</p>
                        <p className="text-xs text-muted">Currently: {formatExpiry(info.expiresAt)}</p>
                        <select
                          value={newExpireSeconds}
                          onChange={(e) => setNewExpireSeconds(e.target.value)}
                          className="mt-1.5 w-full rounded-lg border bg-surface px-3 py-2 text-sm transition-colors focus:border-brand"
                        >
                          {EXPIRE_OPTIONS.map((o) => (
                            <option key={o.value} value={o.value}>
                              {o.label}
                            </option>
                          ))}
                        </select>
                      </div>

                      {settings.passwordEnabled && (
                        <div>
                          <p className="text-sm font-medium">Password</p>
                          <input
                            type="password"
                            value={newPassword}
                            disabled={removePassword}
                            onChange={(e) => setNewPassword(e.target.value)}
                            placeholder={info.requiresPassword ? "New password (leave blank to keep current)" : "Set a password (optional)"}
                            className="mt-1.5 w-full rounded-lg border bg-surface px-3 py-2 text-sm transition-colors focus:border-brand disabled:opacity-50"
                          />
                          {info.requiresPassword && (
                            <label className="mt-1.5 flex items-center gap-2 text-xs text-muted">
                              <input
                                type="checkbox"
                                checked={removePassword}
                                onChange={(e) => {
                                  setRemovePassword(e.target.checked);
                                  if (e.target.checked) setNewPassword("");
                                }}
                              />
                              Remove the current password instead
                            </label>
                          )}
                        </div>
                      )}

                      {settingsError && <p className="text-sm text-red-500">{settingsError}</p>}

                      <div className="flex gap-2">
                        <button
                          onClick={saveSettings}
                          disabled={savingSettings}
                          className="rounded-lg bg-brand px-3 py-1.5 text-xs font-medium text-white transition-opacity hover:opacity-90 disabled:opacity-50"
                        >
                          {savingSettings ? "Saving…" : "Save changes"}
                        </button>
                        <button
                          onClick={() => {
                            setEditingSettings(false);
                            setRemovePassword(false);
                            setNewPassword("");
                            setNewExpireSeconds("");
                            setSettingsError(null);
                          }}
                          disabled={savingSettings}
                          className="rounded-lg border bg-surface px-3 py-1.5 text-xs font-medium transition-colors hover:bg-surface-muted disabled:opacity-50"
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  ) : (
                    <div className="flex items-center justify-between">
                      <button
                        onClick={() => setEditingSettings(true)}
                        className="flex items-center gap-1.5 text-xs text-muted transition-colors hover:text-foreground"
                      >
                        <PencilIcon />
                        Edit password / expiry
                      </button>
                      {settingsSaved && <span className="text-xs text-green-600 dark:text-green-400">Saved.</span>}
                    </div>
                  )}
                </div>
              )}

              {info.isOwner && (
                <div className="border-t pt-4">
                  {confirmingDelete ? (
                    <div className="flex flex-col gap-2 rounded-lg border border-red-500/30 bg-red-500/5 p-3">
                      <p className="text-sm">Delete this upload now? This can&apos;t be undone.</p>
                      {deleteError && <p className="text-sm text-red-500">{deleteError}</p>}
                      <div className="flex gap-2">
                        <button
                          onClick={deleteUpload}
                          disabled={deleting}
                          className="rounded-lg bg-red-500 px-3 py-1.5 text-xs font-medium text-white transition-opacity hover:opacity-90 disabled:opacity-50"
                        >
                          {deleting ? "Deleting…" : "Yes, delete it"}
                        </button>
                        <button
                          onClick={() => {
                            setConfirmingDelete(false);
                            setDeleteError(null);
                          }}
                          disabled={deleting}
                          className="rounded-lg border bg-surface px-3 py-1.5 text-xs font-medium transition-colors hover:bg-surface-muted disabled:opacity-50"
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  ) : (
                    <button
                      onClick={() => setConfirmingDelete(true)}
                      className="flex items-center gap-1.5 text-xs text-muted transition-colors hover:text-red-500"
                    >
                      <TrashIcon />
                      Delete this upload now
                    </button>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </main>
  );
}

export default function DownloadPageClient({ uploadId, code }: { uploadId: string; code: string }) {
  const settings = useSettings();
  const locked = settings.lockPage === "both" || settings.lockPage === "download";

  return (
    <LockGate locked={locked}>
      <DownloadView uploadId={uploadId} code={code} />
    </LockGate>
  );
}

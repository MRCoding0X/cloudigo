"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import QRCode from "qrcode";

import { useUploadSocket, type UploadEvent } from "@/lib/ws";
import {
  completeUpload,
  createUploadSession,
  registerUpload,
  requestEmailVerification,
  confirmEmailVerification,
  uploadFileInChunks,
} from "@/lib/upload-client";
import LockGate from "@/components/lock-gate";
import TermsGate from "@/components/terms-gate";
import SiteFooter from "@/components/site-footer";
import { FileTypeIcon } from "@/components/file-type-icon";
import { useSettings } from "@/lib/settings-context";

type Stage = "idle" | "uploading" | "processing" | "done" | "error";

type FileProgress = {
  file: File;
  id: string;
  receivedBytes: number;
  complete: boolean;
};

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

function formatDuration(seconds: number): string {
  if (!isFinite(seconds) || seconds < 0) return "…";
  if (seconds < 60) return `${Math.ceil(seconds)}s`;
  const m = Math.floor(seconds / 60);
  const s = Math.round(seconds % 60);
  return `${m}m ${s}s`;
}

const EXPIRE_OPTIONS = [
  { label: "2 weeks (default)", value: "" },
  { label: "1 hour", value: String(60 * 60) },
  { label: "1 day", value: String(60 * 60 * 24) },
  { label: "1 week", value: String(60 * 60 * 24 * 7) },
  { label: "Never", value: "0" },
];

const inputClass =
  "rounded-lg border bg-surface px-3 py-2.5 text-sm text-foreground placeholder:text-muted transition-colors focus:border-brand";

function UploadIcon() {
  return (
    <svg width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 16V4M12 4l-4 4M12 4l4 4" />
      <path d="M4 16v2a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2v-2" />
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

function FolderIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
      <path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2Z" />
    </svg>
  );
}

function ResultCard({
  resultLink,
  manageLink,
  downloadEvents,
}: {
  resultLink: string;
  manageLink: string;
  downloadEvents: string[];
}) {
  const [copied, setCopied] = useState(false);
  const [showQr, setShowQr] = useState(false);
  const [qrDataUrl, setQrDataUrl] = useState<string | null>(null);

  useEffect(() => {
    if (!showQr || qrDataUrl) return;
    QRCode.toDataURL(resultLink, { margin: 1, width: 220 })
      .then(setQrDataUrl)
      .catch(() => {});
  }, [showQr, qrDataUrl, resultLink]);

  function copyLink() {
    navigator.clipboard.writeText(resultLink);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  return (
    <main className="relative flex flex-1 items-center justify-center overflow-hidden p-6 sm:p-8">
      <div className="pointer-events-none absolute -top-24 -left-24 h-72 w-72 animate-float-slow rounded-full bg-brand/25 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-24 -right-24 h-72 w-72 animate-float rounded-full bg-brand/15 blur-3xl" />
      <div className="animate-fade-up relative flex w-full max-w-md flex-col items-center gap-5 rounded-2xl border bg-surface p-8 text-center shadow-sm">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-brand text-white shadow-sm">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
            <path d="M20 6 9 17l-5-5" />
          </svg>
        </div>
        <div>
          <h1 className="text-xl font-semibold">Done!</h1>
          <p className="mt-1 text-sm text-muted">Your share link is ready.</p>
        </div>
        <div className="flex w-full items-center gap-2 rounded-lg border bg-surface-muted p-2.5 pl-3">
          <code className="flex-1 truncate text-left text-sm">{resultLink}</code>
          <button
            onClick={copyLink}
            className="flex shrink-0 items-center gap-1.5 rounded-md border bg-surface px-2.5 py-1.5 text-xs font-medium transition-colors hover:bg-surface-muted"
          >
            <CopyIcon />
            {copied ? "Copied" : "Copy"}
          </button>
        </div>

        <button
          onClick={() => setShowQr((v) => !v)}
          className="flex items-center gap-1.5 text-xs text-muted transition-colors hover:text-foreground"
        >
          <QrIcon />
          {showQr ? "Hide QR code" : "Show QR code"}
        </button>
        {showQr && (
          <div className="animate-fade-up rounded-xl bg-white p-3 shadow-sm">
            {qrDataUrl ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={qrDataUrl} alt="QR code for the share link" className="h-[220px] w-[220px]" />
            ) : (
              <div className="h-[220px] w-[220px]" />
            )}
          </div>
        )}

        {downloadEvents.length > 0 && (
          <div className="w-full rounded-lg border border-green-600/20 bg-green-600/10 p-3 text-left text-sm">
            <p className="font-medium text-green-700 dark:text-green-400">Live activity</p>
            <ul className="mt-1 space-y-1 text-muted">
              {downloadEvents.map((e, i) => (
                <li key={i}>{e}</li>
              ))}
            </ul>
          </div>
        )}
        <button
          onClick={() => window.location.reload()}
          className="w-full rounded-lg bg-brand px-4 py-2.5 text-sm font-medium text-white shadow-sm shadow-brand/20 transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-brand/30 active:translate-y-0 active:scale-[0.98]"
        >
          Upload another file
        </button>
        <a href={manageLink} className="text-xs text-muted underline decoration-dotted underline-offset-2 hover:text-foreground">
          Manage or delete this upload — keep this link private
        </a>
      </div>
    </main>
  );
}

function UploadForm() {
  const settings = useSettings();
  const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

  const [files, setFiles] = useState<FileProgress[]>([]);
  const [shareType, setShareType] = useState<"link" | "mail">("link");
  const [recipients, setRecipients] = useState("");
  const [emailFrom, setEmailFrom] = useState("");
  const [message, setMessage] = useState("");
  const [password, setPassword] = useState("");
  const [destruct, setDestruct] = useState(false);
  const [expireSeconds, setExpireSeconds] = useState("");

  const [stage, setStage] = useState<Stage>("idle");
  const [error, setError] = useState<string | null>(null);
  const [uploadId, setUploadId] = useState<string | null>(null);
  const [resultLink, setResultLink] = useState<string | null>(null);
  const [manageLink, setManageLink] = useState<string | null>(null);
  const [downloadEvents, setDownloadEvents] = useState<string[]>([]);
  const [speedBps, setSpeedBps] = useState<number | null>(null);
  const [retrying, setRetrying] = useState(false);

  const [verifyStep, setVerifyStep] = useState(false);
  const [verifyCode, setVerifyCode] = useState("");

  const fileInputRef = useRef<HTMLInputElement>(null);
  const folderInputRef = useRef<HTMLInputElement | null>(null);
  const abortControllerRef = useRef<AbortController | null>(null);
  const speedSampleRef = useRef<{ time: number; bytes: number } | null>(null);

  const totalBytes = useMemo(() => files.reduce((sum, f) => sum + f.file.size, 0), [files]);
  const receivedTotal = useMemo(() => files.reduce((sum, f) => sum + f.receivedBytes, 0), [files]);

  useEffect(() => {
    if (stage !== "uploading") {
      speedSampleRef.current = null;
      setSpeedBps(null);
      return;
    }
    const now = Date.now();
    const prev = speedSampleRef.current;
    if (prev) {
      const dtSeconds = (now - prev.time) / 1000;
      const deltaBytes = receivedTotal - prev.bytes;
      if (dtSeconds > 0.4 && deltaBytes >= 0) {
        setSpeedBps(deltaBytes / dtSeconds);
        speedSampleRef.current = { time: now, bytes: receivedTotal };
      }
    } else {
      speedSampleRef.current = { time: now, bytes: receivedTotal };
    }
  }, [receivedTotal, stage]);

  useUploadSocket(uploadId, (event: UploadEvent) => {
    if (event.type === "chunk.progress" || event.type === "file.complete") {
      setRetrying(false);
      setFiles((prev) =>
        prev.map((f) =>
          f.id === event.fileId
            ? {
                ...f,
                receivedBytes: "receivedBytes" in event ? event.receivedBytes : f.file.size,
                complete: event.type === "file.complete",
              }
            : f,
        ),
      );
    } else if (event.type === "upload.ready") {
      setStage("done");
    } else if (event.type === "download.happened") {
      const time = new Date(event.downloadedAt).toLocaleTimeString("en-US");
      setDownloadEvents((prev) => [
        ...prev,
        `Someone downloaded the file at ${time}` + (event.email ? ` (${event.email})` : ""),
      ]);
    }
  });

  const maxUploadSizeBytes = settings.maxUploadSizeMB > 0 ? settings.maxUploadSizeMB * 1024 * 1024 : 0;

  function addFiles(list: FileList | null) {
    if (!list) return;
    const incoming = Array.from(list);

    if (maxUploadSizeBytes > 0) {
      const oversized = incoming.find((f) => f.size > maxUploadSizeBytes);
      if (oversized) {
        setError(`"${oversized.name}" is larger than the ${formatBytes(maxUploadSizeBytes)} limit.`);
        return;
      }
    }

    if (settings.maxFiles > 0 && files.length + incoming.length > settings.maxFiles) {
      setError(`You can upload at most ${settings.maxFiles} files at once.`);
      return;
    }

    setError(null);
    const next: FileProgress[] = incoming.map((file) => ({
      file,
      id: crypto.randomUUID(),
      receivedBytes: 0,
      complete: false,
    }));
    setFiles((prev) => [...prev, ...next]);
  }

  function removeFile(id: string) {
    setFiles((prev) => prev.filter((f) => f.id !== id));
  }

  // Holds the session once files are uploaded, so a retry after email
  // verification only redoes register()+complete() — never re-uploads files.
  const pendingSessionRef = useRef<{ uploadId: string; secretCode: string; shareCode: string } | null>(null);

  const finishRegistration = useCallback(
    async (session: { uploadId: string; secretCode: string; shareCode: string }) => {
      await registerUpload(session.uploadId, {
        emailFrom,
        message,
        recipients: shareType === "mail" ? recipients.split(",").map((r) => r.trim()).filter(Boolean) : undefined,
        password: password || undefined,
        destruct,
        shareType,
        expireSeconds: expireSeconds === "" ? undefined : Number(expireSeconds),
      });

      const result = await completeUpload(session.uploadId);
      // The link shown/copied/QR-coded here must always be the download-only
      // share code, never the owner secret — otherwise anyone the uploader
      // forwards it to (or a mail recipient who reshares it) inherits full
      // owner rights (delete, edit password/expiry). The secret stays behind
      // its own separate "manage" link below.
      setResultLink(`${window.location.origin}/${result.uploadId}/${session.shareCode}`);
      setManageLink(`${window.location.origin}/${result.uploadId}/${session.secretCode}`);
      setStage("done");
    },
    [shareType, recipients, emailFrom, message, password, destruct, expireSeconds],
  );

  const startUpload = useCallback(async () => {
    if (files.length === 0) return;
    setError(null);
    setStage("uploading");

    const controller = new AbortController();
    abortControllerRef.current = controller;

    try {
      const session = await createUploadSession();
      setUploadId(session.uploadId);

      for (const f of files) {
        await uploadFileInChunks(session.uploadId, f.id, f.file, controller.signal, () => setRetrying(true));
        setRetrying(false);
      }
      pendingSessionRef.current = session;

      setStage("processing");
      await finishRegistration(session);
    } catch (err) {
      setRetrying(false);
      if (err instanceof DOMException && err.name === "AbortError") {
        return; // cancelUpload() already reset the UI
      }
      if (err instanceof Error && err.message === "email address not verified") {
        setStage("idle");
        try {
          await requestEmailVerification(emailFrom);
          setVerifyStep(true);
        } catch {
          setError("Could not send verification code");
        }
        return;
      }
      setError(err instanceof Error ? err.message : "Something went wrong");
      setStage("error");
    }
  }, [files, emailFrom, finishRegistration]);

  function cancelUpload() {
    abortControllerRef.current?.abort();
    setStage("idle");
    setUploadId(null);
    setRetrying(false);
    setFiles((prev) => prev.map((f) => ({ ...f, receivedBytes: 0, complete: false })));
  }

  async function handleConfirmVerification() {
    setError(null);
    const session = pendingSessionRef.current;
    if (!session) {
      setError("Session lost, please upload the files again");
      setVerifyStep(false);
      return;
    }
    try {
      await confirmEmailVerification(emailFrom, verifyCode);
      setVerifyStep(false);
      setStage("processing");
      await finishRegistration(session);
    } catch {
      setError("Incorrect verification code");
    }
  }

  if (stage === "done" && resultLink && manageLink) {
    return <ResultCard resultLink={resultLink} manageLink={manageLink} downloadEvents={downloadEvents} />;
  }

  const remainingBytes = Math.max(totalBytes - receivedTotal, 0);
  const etaSeconds = speedBps && speedBps > 0 ? remainingBytes / speedBps : null;

  return (
    <main className="relative flex flex-1 items-center justify-center overflow-hidden p-6 sm:p-8">
      <div className="pointer-events-none absolute -top-32 -left-32 h-80 w-80 animate-float-slow rounded-full bg-brand/20 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-32 -right-32 h-80 w-80 animate-float rounded-full bg-brand/12 blur-3xl" />
      <div className="animate-fade-up relative flex w-full max-w-lg flex-col gap-6">
        <div className="flex flex-col items-center text-center">
          {settings.logoPath ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={`${apiUrl}${settings.logoPath}`} alt={settings.siteName} className="mb-3 h-10 w-auto" />
          ) : (
            <h1 className="bg-gradient-to-br from-foreground to-foreground/70 bg-clip-text text-2xl font-semibold tracking-tight text-transparent">
              {settings.siteName}
            </h1>
          )}
          <p className="mt-1 text-sm text-muted">Share files fast and securely.</p>
        </div>

        <div className="rounded-2xl border bg-surface p-5 shadow-sm sm:p-6">
          {verifyStep ? (
            <div className="flex flex-col gap-3">
              <p className="text-sm">Enter the code sent to {emailFrom}.</p>
              <input
                value={verifyCode}
                onChange={(e) => setVerifyCode(e.target.value)}
                placeholder="4-digit code"
                className={inputClass}
              />
              <button
                onClick={handleConfirmVerification}
                className="rounded-lg bg-brand px-4 py-2.5 text-sm font-medium text-white shadow-sm shadow-brand/20 transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-brand/30 active:translate-y-0 active:scale-[0.98]"
              >
                Confirm
              </button>
            </div>
          ) : (
            <div className="flex flex-col gap-4">
              <div
                onDragOver={(e) => e.preventDefault()}
                onDrop={(e) => {
                  e.preventDefault();
                  addFiles(e.dataTransfer.files);
                }}
                onClick={() => fileInputRef.current?.click()}
                className="group flex cursor-pointer flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed bg-surface-muted p-10 text-center transition-all hover:border-brand hover:bg-brand-soft active:scale-[0.99]"
              >
                <input
                  ref={fileInputRef}
                  type="file"
                  multiple
                  className="hidden"
                  onChange={(e) => addFiles(e.target.files)}
                />
                <input
                  ref={(el) => {
                    folderInputRef.current = el;
                    if (el) el.setAttribute("webkitdirectory", "");
                  }}
                  type="file"
                  multiple
                  className="hidden"
                  onChange={(e) => addFiles(e.target.files)}
                />
                <div className="text-muted transition-transform duration-300 group-hover:-translate-y-1 group-hover:text-brand">
                  <UploadIcon />
                </div>
                <p className="text-sm text-muted">
                  <span className="font-medium text-foreground">Click to upload</span> or drag and drop
                </p>
                {(settings.maxFiles > 0 || maxUploadSizeBytes > 0) && (
                  <p className="text-xs text-muted">
                    {settings.maxFiles > 0 && maxUploadSizeBytes > 0
                      ? `Up to ${settings.maxFiles} files, ${formatBytes(maxUploadSizeBytes)} per upload`
                      : settings.maxFiles > 0
                        ? `Up to ${settings.maxFiles} files`
                        : `Up to ${formatBytes(maxUploadSizeBytes)} per upload`}
                  </p>
                )}
                <button
                  type="button"
                  onClick={(e) => {
                    e.stopPropagation();
                    folderInputRef.current?.click();
                  }}
                  className="mt-1 flex items-center gap-1.5 text-xs text-muted underline-offset-2 hover:text-brand hover:underline"
                >
                  <FolderIcon />
                  or select a folder
                </button>
              </div>

              {files.length > 0 && (
                <ul className="flex flex-col gap-2">
                  {files.map((f) => (
                    <li key={f.id} className="animate-fade-up flex items-center gap-3 rounded-lg border bg-surface-muted px-3 py-2 text-sm transition-colors">
                      <FileTypeIcon fileName={f.file.name} size="sm" />
                      <span className="flex-1 truncate">{f.file.name}</span>
                      <span className="shrink-0 text-xs text-muted">{formatBytes(f.file.size)}</span>
                      {stage === "uploading" || stage === "processing" ? (
                        <div className="h-1.5 w-20 shrink-0 overflow-hidden rounded-full bg-surface">
                          <div
                            className="h-full bg-brand transition-all"
                            style={{ width: `${f.file.size ? (f.receivedBytes / f.file.size) * 100 : 0}%` }}
                          />
                        </div>
                      ) : (
                        <button onClick={() => removeFile(f.id)} className="shrink-0 text-muted hover:text-red-500">
                          ✕
                        </button>
                      )}
                    </li>
                  ))}
                </ul>
              )}

              {stage === "uploading" && totalBytes > 0 && (
                <div className="flex items-center justify-between gap-2 text-xs text-muted">
                  <span className={retrying ? "flex items-center gap-1.5 text-amber-500" : undefined}>
                    {retrying ? (
                      <>
                        <span className="h-1.5 w-1.5 animate-pulse rounded-full bg-amber-500" />
                        Reconnecting…
                      </>
                    ) : (
                      <>
                        Uploading… {formatBytes(receivedTotal)} / {formatBytes(totalBytes)}
                        {speedBps ? ` · ${formatBytes(speedBps)}/s` : ""}
                        {etaSeconds !== null ? ` · ${formatDuration(etaSeconds)} left` : ""}
                      </>
                    )}
                  </span>
                  <button onClick={cancelUpload} className="shrink-0 font-medium text-red-500 hover:underline">
                    Cancel
                  </button>
                </div>
              )}
              {stage === "processing" && (
                <p className="text-center text-xs text-muted">Processing (zipping/encrypting if needed)…</p>
              )}

              <fieldset className="flex flex-col gap-3 border-t pt-4" disabled={stage !== "idle" && stage !== "error"}>
                {settings.shareEnabled && (
                  <div className="flex gap-1 rounded-lg bg-surface-muted p-1 text-sm">
                    <button
                      type="button"
                      onClick={() => setShareType("link")}
                      className={`flex-1 rounded-md py-1.5 font-medium transition-all ${shareType === "link" ? "bg-surface shadow-sm" : "text-muted hover:text-foreground"}`}
                    >
                      Link
                    </button>
                    <button
                      type="button"
                      onClick={() => setShareType("mail")}
                      className={`flex-1 rounded-md py-1.5 font-medium transition-all ${shareType === "mail" ? "bg-surface shadow-sm" : "text-muted hover:text-foreground"}`}
                    >
                      Email
                    </button>
                  </div>
                )}

                <input
                  value={emailFrom}
                  onChange={(e) => setEmailFrom(e.target.value)}
                  placeholder="Your email (optional)"
                  className={inputClass}
                />

                {settings.shareEnabled && shareType === "mail" && (
                  <input
                    value={recipients}
                    onChange={(e) => setRecipients(e.target.value)}
                    placeholder="Recipients, comma-separated email addresses"
                    className={inputClass}
                  />
                )}

                <textarea
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                  placeholder="Message (optional)"
                  rows={2}
                  className={inputClass}
                />

                {settings.passwordEnabled && (
                  <input
                    type="password"
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                    placeholder="Password (optional)"
                    className={inputClass}
                  />
                )}

                {settings.destructEnabled && (
                  <label className="flex items-center gap-2 text-sm text-muted">
                    <input type="checkbox" checked={destruct} onChange={(e) => setDestruct(e.target.checked)} />
                    Delete automatically after download
                  </label>
                )}

                <select
                  value={expireSeconds}
                  onChange={(e) => setExpireSeconds(e.target.value)}
                  className={inputClass}
                >
                  {EXPIRE_OPTIONS.map((o) => (
                    <option key={o.value} value={o.value}>
                      {o.label}
                    </option>
                  ))}
                </select>
              </fieldset>

              {error && <p className="text-center text-sm text-red-500">{error}</p>}

              <button
                onClick={startUpload}
                disabled={files.length === 0 || stage === "uploading" || stage === "processing"}
                className="rounded-lg bg-brand px-4 py-3 text-sm font-medium text-white shadow-sm shadow-brand/20 transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-brand/30 active:translate-y-0 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-40"
              >
                {stage === "uploading" || stage === "processing" ? "Uploading…" : "Upload"}
              </button>
            </div>
          )}
        </div>
      </div>
    </main>
  );
}

export default function UploadPage() {
  const settings = useSettings();
  const locked = settings.lockPage === "both" || settings.lockPage === "upload";

  return (
    <>
      <LockGate locked={locked}>
        <TermsGate required={settings.acceptTerms}>
          <UploadForm />
        </TermsGate>
      </LockGate>
      <SiteFooter />
    </>
  );
}

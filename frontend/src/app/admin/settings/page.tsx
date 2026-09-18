"use client";

import { useEffect, useRef, useState } from "react";

import { apiFetch, apiFetchJSON, ApiError } from "@/lib/api";
import { Card, PageHeader, inputClass as baseInputClass, primaryButtonClass, secondaryButtonClass } from "@/components/admin-ui";
import { Spinner } from "@/components/icons";

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

type Settings = {
  siteName: string; siteUrl: string;
  maxUploadSizeMb: number; maxChunkSizeMb: number; maxFiles: number; maxRecipients: number;
  blockedFileTypes: string; blockedEmails: string; defaultExpireSeconds: number; uploadIdLength: number;
  emailVerify: string; passwordEnabled: boolean; destructEnabled: boolean; shareEnabled: boolean;
  defaultShareType: string; encryptFiles: boolean; ipUploadLimit: number;
  smtpHost: string; smtpPort: number; smtpUsername: string; smtpPassword: string;
  emailFromName: string; emailFromAddress: string;
  contactEnabled: boolean; contactEmail: string;
  themeColor: string; themeColorSecondary: string; logoPath: string; faviconPath: string;
  lockPage: string; acceptTerms: boolean;
};

type Social = { facebook: string; twitter: string; instagram: string; github: string };

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="flex flex-col gap-1.5 text-xs font-medium text-muted">
      {label}
      {children}
    </label>
  );
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <Card className="flex flex-col gap-4 p-5">
      <h2 className="font-semibold">{title}</h2>
      {children}
    </Card>
  );
}

const inputClass = baseInputClass;
const checkboxRow = "flex items-center gap-2 text-sm";

export default function AdminSettingsPage() {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [social, setSocial] = useState<Social | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [brandingError, setBrandingError] = useState<string | null>(null);
  const [importError, setImportError] = useState<string | null>(null);
  const logoInputRef = useRef<HTMLInputElement>(null);
  const faviconInputRef = useRef<HTMLInputElement>(null);
  const importInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    apiFetchJSON<Settings>("/api/admin/settings").then(setSettings);
    apiFetchJSON<Social>("/api/admin/social").then(setSocial);
  }, []);

  function set<K extends keyof Settings>(key: K, value: Settings[K]) {
    setSettings((s) => (s ? { ...s, [key]: value } : s));
  }

  function setSocialField<K extends keyof Social>(key: K, value: Social[K]) {
    setSocial((s) => (s ? { ...s, [key]: value } : s));
  }

  async function uploadBranding(kind: "logo" | "favicon", file: File) {
    setBrandingError(null);
    const formData = new FormData();
    formData.append("file", file);
    try {
      const res = await apiFetch(`/api/admin/branding/${kind}`, { method: "POST", body: formData });
      if (!res.ok) throw new Error();
      const data: { path: string } = await res.json();
      set(kind === "logo" ? "logoPath" : "faviconPath", data.path);
    } catch {
      setBrandingError(`Could not upload the ${kind === "logo" ? "logo" : "favicon"}`);
    }
  }

  // A plain <a href> can't carry the Authorization header this protected
  // endpoint needs, so fetch it via apiFetch and trigger the save manually.
  async function exportSettings() {
    const res = await apiFetch("/api/admin/settings/export");
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "cloudigo-settings.json";
    a.click();
    URL.revokeObjectURL(url);
  }

  async function importSettings(file: File) {
    setImportError(null);
    setMessage(null);
    try {
      const text = await file.text();
      const parsed = JSON.parse(text);
      await apiFetchJSON("/api/admin/settings/import", { method: "POST", body: JSON.stringify(parsed) });
      const [newSettings, newSocial] = await Promise.all([
        apiFetchJSON<Settings>("/api/admin/settings"),
        apiFetchJSON<Social>("/api/admin/social"),
      ]);
      setSettings(newSettings);
      setSocial(newSocial);
      setMessage("Settings imported.");
    } catch (err) {
      setImportError(err instanceof ApiError ? err.message : "Could not import — check the file is a valid Cloudigo settings export");
    }
  }

  async function save() {
    if (!settings || !social) return;
    setMessage(null);
    setError(null);
    try {
      await apiFetchJSON("/api/admin/settings", { method: "PUT", body: JSON.stringify(settings) });
      await apiFetchJSON("/api/admin/social", { method: "PUT", body: JSON.stringify(social) });
      setMessage("Saved.");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not save settings");
    }
  }

  if (!settings || !social) {
    return (
      <div className="flex items-center justify-center p-8 text-muted">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <PageHeader
        title="Settings"
        action={
          <button onClick={save} className={primaryButtonClass}>
            Save all changes
          </button>
        }
      />
      {message && <p className="text-sm text-green-600 dark:text-green-400">{message}</p>}
      {error && <p className="text-sm text-red-500">{error}</p>}

      <Section title="General">
        <div className="grid grid-cols-2 gap-4">
          <Field label="Site name"><input className={inputClass} value={settings.siteName} onChange={(e) => set("siteName", e.target.value)} /></Field>
          <Field label="Site URL"><input className={inputClass} value={settings.siteUrl} onChange={(e) => set("siteUrl", e.target.value)} /></Field>
        </div>
      </Section>

      <Section title="Upload">
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
          <Field label="Max file size (MB)"><input type="number" className={inputClass} value={settings.maxUploadSizeMb} onChange={(e) => set("maxUploadSizeMb", Number(e.target.value))} /></Field>
          <Field label="Chunk size (MB)"><input type="number" className={inputClass} value={settings.maxChunkSizeMb} onChange={(e) => set("maxChunkSizeMb", Number(e.target.value))} /></Field>
          <Field label="Max number of files"><input type="number" className={inputClass} value={settings.maxFiles} onChange={(e) => set("maxFiles", Number(e.target.value))} /></Field>
          <Field label="Max number of recipients"><input type="number" className={inputClass} value={settings.maxRecipients} onChange={(e) => set("maxRecipients", Number(e.target.value))} /></Field>
          <Field label="Default expiry (sec)"><input type="number" className={inputClass} value={settings.defaultExpireSeconds} onChange={(e) => set("defaultExpireSeconds", Number(e.target.value))} /></Field>
          <Field label="Upload ID length"><input type="number" className={inputClass} value={settings.uploadIdLength} onChange={(e) => set("uploadIdLength", Number(e.target.value))} /></Field>
          <Field label="IP upload limit/hr (0=none)"><input type="number" className={inputClass} value={settings.ipUploadLimit} onChange={(e) => set("ipUploadLimit", Number(e.target.value))} /></Field>
          <Field label="Default share type">
            <select className={inputClass} value={settings.defaultShareType} onChange={(e) => set("defaultShareType", e.target.value)}>
              <option value="link">link</option>
              <option value="mail">mail</option>
            </select>
          </Field>
          <Field label="Email verification">
            <select className={inputClass} value={settings.emailVerify} onChange={(e) => set("emailVerify", e.target.value)}>
              <option value="false">Off</option>
              <option value="once">Once</option>
              <option value="always">Always</option>
            </select>
          </Field>
        </div>
        <Field label="Blocked file extensions (comma-separated)"><input className={inputClass} value={settings.blockedFileTypes} onChange={(e) => set("blockedFileTypes", e.target.value)} /></Field>
        <Field label="Blocked email addresses (comma-separated)"><input className={inputClass} value={settings.blockedEmails} onChange={(e) => set("blockedEmails", e.target.value)} /></Field>
        <div className="flex flex-wrap gap-x-6 gap-y-2 border-t pt-4">
          <label className={checkboxRow}><input type="checkbox" checked={settings.passwordEnabled} onChange={(e) => set("passwordEnabled", e.target.checked)} /> Password allowed</label>
          <label className={checkboxRow}><input type="checkbox" checked={settings.destructEnabled} onChange={(e) => set("destructEnabled", e.target.checked)} /> Self-destruct allowed</label>
          <label className={checkboxRow}><input type="checkbox" checked={settings.shareEnabled} onChange={(e) => set("shareEnabled", e.target.checked)} /> Sharing allowed</label>
          <label className={checkboxRow}><input type="checkbox" checked={settings.encryptFiles} onChange={(e) => set("encryptFiles", e.target.checked)} /> Encrypt files</label>
        </div>
      </Section>

      <Section title="Mail (SMTP)">
        <div className="grid grid-cols-2 gap-4">
          <Field label="SMTP host"><input className={inputClass} value={settings.smtpHost} onChange={(e) => set("smtpHost", e.target.value)} /></Field>
          <Field label="SMTP port"><input type="number" className={inputClass} value={settings.smtpPort} onChange={(e) => set("smtpPort", Number(e.target.value))} /></Field>
          <Field label="SMTP username"><input className={inputClass} value={settings.smtpUsername} onChange={(e) => set("smtpUsername", e.target.value)} /></Field>
          <Field label="SMTP password (blank = unchanged)"><input type="password" className={inputClass} value={settings.smtpPassword} onChange={(e) => set("smtpPassword", e.target.value)} /></Field>
          <Field label="Sender name"><input className={inputClass} value={settings.emailFromName} onChange={(e) => set("emailFromName", e.target.value)} /></Field>
          <Field label="Sender address"><input className={inputClass} value={settings.emailFromAddress} onChange={(e) => set("emailFromAddress", e.target.value)} /></Field>
        </div>
        <p className="text-xs text-muted">Leave SMTP host blank to log emails to the server console instead of actually sending them.</p>
      </Section>

      <Section title="Contact">
        <label className={checkboxRow}><input type="checkbox" checked={settings.contactEnabled} onChange={(e) => set("contactEnabled", e.target.checked)} /> Contact form enabled</label>
        <Field label="Contact email"><input className={inputClass} value={settings.contactEmail} onChange={(e) => set("contactEmail", e.target.value)} /></Field>
      </Section>

      <Section title="Appearance">
        <div className="grid grid-cols-2 gap-4">
          <Field label="Accent color">
            <div className="flex items-center gap-2">
              <input type="color" className="h-9 w-12 shrink-0 cursor-pointer rounded-lg border bg-surface p-1" value={settings.themeColor} onChange={(e) => set("themeColor", e.target.value)} />
              <span className="text-sm text-muted">{settings.themeColor}</span>
            </div>
          </Field>
          <Field label="Secondary color">
            <div className="flex items-center gap-2">
              <input type="color" className="h-9 w-12 shrink-0 cursor-pointer rounded-lg border bg-surface p-1" value={settings.themeColorSecondary} onChange={(e) => set("themeColorSecondary", e.target.value)} />
              <span className="text-sm text-muted">{settings.themeColorSecondary}</span>
            </div>
          </Field>
        </div>
        {brandingError && <p className="text-sm text-red-500">{brandingError}</p>}
        <div className="grid grid-cols-2 gap-4 border-t pt-4">
          <div className="flex flex-col gap-2">
            <span className="text-xs font-medium text-muted">Logo</span>
            <div className="flex items-center gap-3">
              <div className="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-surface-muted">
                {settings.logoPath ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={`${API_URL}${settings.logoPath}`} alt="Logo" className="h-full w-full object-contain" />
                ) : (
                  <span className="text-xs text-muted">None</span>
                )}
              </div>
              <input
                ref={logoInputRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={(e) => e.target.files?.[0] && uploadBranding("logo", e.target.files[0])}
              />
              <button type="button" onClick={() => logoInputRef.current?.click()} className={secondaryButtonClass}>
                Upload
              </button>
              {settings.logoPath && (
                <button type="button" onClick={() => set("logoPath", "")} className="text-xs text-muted hover:text-red-500">
                  Remove
                </button>
              )}
            </div>
          </div>
          <div className="flex flex-col gap-2">
            <span className="text-xs font-medium text-muted">Favicon</span>
            <div className="flex items-center gap-3">
              <div className="flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-lg border bg-surface-muted">
                {settings.faviconPath ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={`${API_URL}${settings.faviconPath}`} alt="Favicon" className="h-full w-full object-contain" />
                ) : (
                  <span className="text-xs text-muted">None</span>
                )}
              </div>
              <input
                ref={faviconInputRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={(e) => e.target.files?.[0] && uploadBranding("favicon", e.target.files[0])}
              />
              <button type="button" onClick={() => faviconInputRef.current?.click()} className={secondaryButtonClass}>
                Upload
              </button>
              {settings.faviconPath && (
                <button type="button" onClick={() => set("faviconPath", "")} className="text-xs text-muted hover:text-red-500">
                  Remove
                </button>
              )}
            </div>
          </div>
        </div>
      </Section>

      <Section title="Access">
        <Field label="Lock page (requires login)">
          <select className={inputClass} value={settings.lockPage} onChange={(e) => set("lockPage", e.target.value)}>
            <option value="false">Off</option>
            <option value="upload">Upload only</option>
            <option value="download">Download only</option>
            <option value="both">Upload and download</option>
          </select>
        </Field>
        <label className={checkboxRow}>
          <input type="checkbox" checked={settings.acceptTerms} onChange={(e) => set("acceptTerms", e.target.checked)} />
          Require terms acceptance before upload
        </label>
      </Section>

      <Section title="Backup">
        <p className="text-xs text-muted">
          Export the full configuration (including SMTP credentials) as a JSON file for backup, or to move it to
          another Cloudigo instance. Importing replaces all current settings and social links.
        </p>
        <div className="flex items-center gap-3">
          <button type="button" onClick={exportSettings} className={secondaryButtonClass}>
            Export settings
          </button>
          <input
            ref={importInputRef}
            type="file"
            accept="application/json"
            className="hidden"
            onChange={(e) => e.target.files?.[0] && importSettings(e.target.files[0])}
          />
          <button type="button" onClick={() => importInputRef.current?.click()} className={secondaryButtonClass}>
            Import settings
          </button>
        </div>
        {importError && <p className="text-sm text-red-500">{importError}</p>}
      </Section>

      <Section title="Social links">
        <div className="grid grid-cols-2 gap-4">
          <Field label="Facebook"><input className={inputClass} value={social.facebook} onChange={(e) => setSocialField("facebook", e.target.value)} /></Field>
          <Field label="Twitter/X"><input className={inputClass} value={social.twitter} onChange={(e) => setSocialField("twitter", e.target.value)} /></Field>
          <Field label="Instagram"><input className={inputClass} value={social.instagram} onChange={(e) => setSocialField("instagram", e.target.value)} /></Field>
          <Field label="GitHub"><input className={inputClass} value={social.github} onChange={(e) => setSocialField("github", e.target.value)} /></Field>
        </div>
      </Section>
    </div>
  );
}

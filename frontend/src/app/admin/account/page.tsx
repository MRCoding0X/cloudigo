"use client";

import { useState, type FormEvent } from "react";

import { apiFetchJSON, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth-context";
import { Card, PageHeader, inputClass, primaryButtonClass, secondaryButtonClass } from "@/components/admin-ui";

export default function AdminAccountPage() {
  const { user, setUser } = useAuth();

  const [currentPassword, setCurrentPassword] = useState("");
  const [newEmail, setNewEmail] = useState(user?.email ?? "");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const [loggingOutOthers, setLoggingOutOthers] = useState(false);
  const [sessionsMessage, setSessionsMessage] = useState<string | null>(null);

  if (!user) return null;

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setMessage(null);

    if (newPassword && newPassword !== confirmPassword) {
      setError("New passwords do not match");
      return;
    }
    if (newPassword && newPassword.length < 8) {
      setError("New password must be at least 8 characters");
      return;
    }

    const emailChanged = newEmail !== user!.email;
    if (!emailChanged && !newPassword) {
      setError("Nothing to update");
      return;
    }

    setSubmitting(true);
    try {
      const updated = await apiFetchJSON<{ id: string; email: string; role: "admin" | "user" }>("/api/auth/me", {
        method: "PUT",
        body: JSON.stringify({
          currentPassword,
          newEmail: emailChanged ? newEmail : undefined,
          newPassword: newPassword || undefined,
        }),
      });
      setUser(updated);
      setCurrentPassword("");
      setNewPassword("");
      setConfirmPassword("");
      setMessage("Saved.");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not update your account");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleLogoutOthers() {
    setSessionsMessage(null);
    setLoggingOutOthers(true);
    try {
      const result = await apiFetchJSON<{ revoked: number }>("/api/auth/logout-others", { method: "POST" });
      setSessionsMessage(
        result.revoked > 0
          ? `Logged out ${result.revoked} other session${result.revoked === 1 ? "" : "s"}.`
          : "No other sessions were active.",
      );
    } catch (err) {
      setSessionsMessage(err instanceof ApiError ? err.message : "Could not log out other sessions");
    } finally {
      setLoggingOutOthers(false);
    }
  }

  return (
    <div className="flex max-w-md flex-col gap-6">
      <PageHeader title="My account" description="Update your own email or password." />

      <Card className="p-5">
        <form onSubmit={handleSubmit} className="flex flex-col gap-4">
          <label className="flex flex-col gap-1.5 text-sm">
            Email
            <input type="email" value={newEmail} onChange={(e) => setNewEmail(e.target.value)} className={inputClass} />
          </label>

          <div className="flex flex-col gap-4 border-t pt-4">
            <label className="flex flex-col gap-1.5 text-sm">
              New password <span className="font-normal text-muted">(leave blank to keep current)</span>
              <input
                type="password"
                value={newPassword}
                onChange={(e) => setNewPassword(e.target.value)}
                className={inputClass}
              />
            </label>
            {newPassword && (
              <label className="flex flex-col gap-1.5 text-sm">
                Confirm new password
                <input
                  type="password"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className={inputClass}
                />
              </label>
            )}
          </div>

          <label className="flex flex-col gap-1.5 border-t pt-4 text-sm">
            Current password <span className="font-normal text-muted">(required to save changes)</span>
            <input
              type="password"
              required
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
              className={inputClass}
            />
          </label>

          {error && <p className="text-sm text-red-500">{error}</p>}
          {message && <p className="text-sm text-green-600 dark:text-green-400">{message}</p>}

          <button type="submit" disabled={submitting} className={primaryButtonClass}>
            {submitting ? "Saving…" : "Save changes"}
          </button>
        </form>
      </Card>

      <Card className="p-5">
        <h2 className="text-sm font-medium">Sessions</h2>
        <p className="mt-1 text-sm text-muted">
          If you signed in on another device or browser and want to end those sessions, you can log them out here without
          affecting your current session.
        </p>
        <button onClick={handleLogoutOthers} disabled={loggingOutOthers} className={`mt-4 ${secondaryButtonClass}`}>
          {loggingOutOthers ? "Logging out…" : "Log out other sessions"}
        </button>
        {sessionsMessage && <p className="mt-3 text-sm text-muted">{sessionsMessage}</p>}
      </Card>
    </div>
  );
}

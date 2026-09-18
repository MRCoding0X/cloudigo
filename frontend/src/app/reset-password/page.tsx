"use client";

import { Suspense, useState, type FormEvent } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";

import { apiFetchJSON } from "@/lib/api";

function ResetPasswordForm() {
  const token = useSearchParams().get("token") ?? "";

  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);

    if (newPassword.length < 8) {
      setError("Password must be at least 8 characters");
      return;
    }
    if (newPassword !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    setSubmitting(true);
    try {
      await apiFetchJSON("/api/auth/reset-password", {
        method: "POST",
        body: JSON.stringify({ token, newPassword }),
      });
      setDone(true);
    } catch {
      setError("This reset link is invalid or has expired.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="relative flex flex-1 items-center justify-center overflow-hidden p-6 sm:p-8">
      <div className="pointer-events-none absolute -top-28 -left-28 h-72 w-72 animate-float-slow rounded-full bg-brand/20 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-28 -right-28 h-72 w-72 animate-float rounded-full bg-brand/12 blur-3xl" />
      <div className="animate-fade-up relative w-full max-w-sm">
        <div className="flex flex-col gap-4 rounded-2xl border bg-surface p-8 shadow-sm">
          <h1 className="text-xl font-semibold">Reset password</h1>

          {!token ? (
            <p className="text-sm text-red-500">This reset link is invalid or incomplete.</p>
          ) : done ? (
            <>
              <p className="text-sm text-muted">
                Your password has been reset. You can now log in.
              </p>
              <Link href="/login" className="text-sm text-brand hover:underline">
                Go to log in
              </Link>
            </>
          ) : (
            <form onSubmit={handleSubmit} className="flex flex-col gap-4">
              <label className="flex flex-col gap-1.5 text-sm">
                New password
                <input
                  type="password"
                  required
                  minLength={8}
                  value={newPassword}
                  onChange={(e) => setNewPassword(e.target.value)}
                  className="rounded-lg border bg-surface px-3 py-2.5 text-sm transition-colors focus:border-brand"
                />
              </label>

              <label className="flex flex-col gap-1.5 text-sm">
                Confirm password
                <input
                  type="password"
                  required
                  minLength={8}
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  className="rounded-lg border bg-surface px-3 py-2.5 text-sm transition-colors focus:border-brand"
                />
              </label>

              {error && <p className="text-sm text-red-500">{error}</p>}

              <button
                type="submit"
                disabled={submitting}
                className="mt-2 rounded-lg bg-brand px-4 py-2.5 text-sm font-medium text-white shadow-sm shadow-brand/20 transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-brand/30 active:translate-y-0 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50"
              >
                {submitting ? "Resetting…" : "Reset password"}
              </button>
            </form>
          )}
        </div>
      </div>
    </main>
  );
}

export default function ResetPasswordPage() {
  return (
    <Suspense>
      <ResetPasswordForm />
    </Suspense>
  );
}

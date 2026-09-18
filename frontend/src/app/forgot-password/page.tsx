"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";

import { apiFetchJSON } from "@/lib/api";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    try {
      await apiFetchJSON("/api/auth/forgot-password", {
        method: "POST",
        body: JSON.stringify({ email }),
      });
    } finally {
      // Always show the same confirmation, even on error, so we never leak
      // whether an account exists for the given email.
      setSubmitting(false);
      setDone(true);
    }
  }

  return (
    <main className="relative flex flex-1 items-center justify-center overflow-hidden p-6 sm:p-8">
      <div className="pointer-events-none absolute -top-28 -left-28 h-72 w-72 animate-float-slow rounded-full bg-brand/20 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-28 -right-28 h-72 w-72 animate-float rounded-full bg-brand/12 blur-3xl" />
      <div className="animate-fade-up relative w-full max-w-sm">
        <div className="flex flex-col gap-4 rounded-2xl border bg-surface p-8 shadow-sm">
          <h1 className="text-xl font-semibold">Forgot password</h1>

          {done ? (
            <>
              <p className="text-sm text-muted">
                If an account exists for that email, a reset link has been sent.
              </p>
              <Link href="/login" className="text-sm text-brand hover:underline">
                Back to log in
              </Link>
            </>
          ) : (
            <form onSubmit={handleSubmit} className="flex flex-col gap-4">
              <p className="text-sm text-muted">
                Enter your email address and we&apos;ll send you a link to reset your password.
              </p>

              <label className="flex flex-col gap-1.5 text-sm">
                Email
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className="rounded-lg border bg-surface px-3 py-2.5 text-sm transition-colors focus:border-brand"
                />
              </label>

              <button
                type="submit"
                disabled={submitting}
                className="mt-2 rounded-lg bg-brand px-4 py-2.5 text-sm font-medium text-white shadow-sm shadow-brand/20 transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-brand/30 active:translate-y-0 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50"
              >
                {submitting ? "Sending…" : "Send reset link"}
              </button>

              <Link href="/login" className="text-center text-sm text-muted hover:text-foreground">
                Back to log in
              </Link>
            </form>
          )}
        </div>
      </div>
    </main>
  );
}

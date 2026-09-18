"use client";

import { useState, type FormEvent } from "react";

import { apiFetchJSON } from "@/lib/api";
import { useSettings } from "@/lib/settings-context";

const inputClass =
  "rounded-lg border bg-surface px-3 py-2.5 text-sm transition-colors focus:border-brand";

export default function ContactPage() {
  const settings = useSettings();

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [message, setMessage] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      await apiFetchJSON("/api/contact", {
        method: "POST",
        body: JSON.stringify({ name, email, message }),
      });
      setDone(true);
    } catch {
      setError("Could not send your message. Please try again later.");
    } finally {
      setSubmitting(false);
    }
  }

  if (!settings.contactEnabled) {
    return (
      <main className="flex flex-1 flex-col items-center justify-center gap-4 p-8 text-center">
        <p className="text-sm text-muted">The contact form is not available right now.</p>
      </main>
    );
  }

  return (
    <main className="relative flex flex-1 items-center justify-center overflow-hidden p-6 sm:p-8">
      <div className="pointer-events-none absolute -top-28 -left-28 h-72 w-72 animate-float-slow rounded-full bg-brand/20 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-28 -right-28 h-72 w-72 animate-float rounded-full bg-brand/12 blur-3xl" />
      <div className="animate-fade-up relative w-full max-w-lg">
        <div className="flex flex-col gap-4 rounded-2xl border bg-surface p-8 shadow-sm">
          <h1 className="text-xl font-semibold">Contact us</h1>

          {done ? (
            <p className="text-sm text-muted">Thanks! Your message has been sent.</p>
          ) : (
            <form onSubmit={handleSubmit} className="flex flex-col gap-4">
              <p className="text-sm text-muted">
                Have a question or an issue? Send us a message below.
              </p>

              <label className="flex flex-col gap-1.5 text-sm">
                Name
                <input required value={name} onChange={(e) => setName(e.target.value)} className={inputClass} />
              </label>

              <label className="flex flex-col gap-1.5 text-sm">
                Your email
                <input
                  type="email"
                  required
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  className={inputClass}
                />
              </label>

              <label className="flex flex-col gap-1.5 text-sm">
                Message
                <textarea
                  required
                  rows={5}
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                  className={inputClass}
                />
              </label>

              {error && <p className="text-sm text-red-500">{error}</p>}

              <button
                type="submit"
                disabled={submitting}
                className="mt-2 rounded-lg bg-brand px-4 py-2.5 text-sm font-medium text-white shadow-sm shadow-brand/20 transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-brand/30 active:translate-y-0 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50"
              >
                {submitting ? "Sending…" : "Send message"}
              </button>
            </form>
          )}
        </div>
      </div>
    </main>
  );
}

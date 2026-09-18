"use client";

import { useEffect, useState } from "react";

import { apiFetchJSON, ApiError } from "@/lib/api";

const STORAGE_KEY = "cloudigo_terms_accepted";

type TermsPage = { title: string; content: string };

// "checking" only ever exists on the client, for the brief window before the
// localStorage read resolves — the server (and the client's first paint)
// always start from a value that doesn't depend on browser-only storage, so
// hydration never has to reconcile a mismatch.
type GateState = "checking" | "gate" | "accepted";

/**
 * Gates children behind a one-time terms-of-service acceptance screen when
 * the admin-configured accept_terms setting is on. Acceptance is remembered
 * in localStorage so returning visitors aren't asked again.
 */
export default function TermsGate({ required, children }: { required: boolean; children: React.ReactNode }) {
  const [state, setState] = useState<GateState>(required ? "checking" : "accepted");
  const [page, setPage] = useState<TermsPage | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!required) return;
    let alreadyAccepted = false;
    try {
      alreadyAccepted = localStorage.getItem(STORAGE_KEY) === "true";
    } catch {
      // localStorage unavailable (private mode) — fall through to the gate.
    }
    // localStorage can only be read after mount, so this state can only be
    // known on the client — an effect (not a lazy initializer) is required
    // here specifically to keep the server and the client's first paint
    // identical and avoid a hydration mismatch.
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setState(alreadyAccepted ? "accepted" : "gate");
  }, [required]);

  useEffect(() => {
    if (state !== "gate") return;
    apiFetchJSON<TermsPage>("/api/pages/terms_page/en")
      .then(setPage)
      .catch((err) => {
        if (err instanceof ApiError && err.status === 404) setPage({ title: "Terms of service", content: "" });
        else setError(true);
      });
  }, [state]);

  if (state === "accepted") return <>{children}</>;
  if (state === "checking") return null;

  function accept() {
    try {
      localStorage.setItem(STORAGE_KEY, "true");
    } catch {
      // localStorage unavailable (private mode) — accepting still lets the
      // visitor through this session, they'll just be asked again next time.
    }
    setState("accepted");
  }

  return (
    <main className="relative flex flex-1 items-center justify-center overflow-hidden p-6 sm:p-8">
      <div className="pointer-events-none absolute -top-28 -left-28 h-72 w-72 animate-float-slow rounded-full bg-brand/20 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-28 -right-28 h-72 w-72 animate-float rounded-full bg-brand/12 blur-3xl" />
      <div className="animate-fade-up relative flex w-full max-w-lg flex-col gap-4 rounded-2xl border bg-surface p-6 shadow-sm sm:p-8">
        <h1 className="text-xl font-semibold">{page?.title ?? "Terms of service"}</h1>
        {error ? (
          <p className="text-sm text-red-500">Could not load the terms of service.</p>
        ) : page ? (
          <div
            className="prose prose-sm dark:prose-invert max-h-96 overflow-y-auto rounded-lg border bg-surface-muted p-4"
            dangerouslySetInnerHTML={{ __html: page.content }}
          />
        ) : (
          <p className="text-sm text-muted">Loading…</p>
        )}
        <button
          onClick={accept}
          disabled={!page && !error}
          className="rounded-lg bg-brand px-4 py-3 text-sm font-medium text-white shadow-sm shadow-brand/20 transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-brand/30 active:translate-y-0 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50"
        >
          Accept and continue
        </button>
      </div>
    </main>
  );
}

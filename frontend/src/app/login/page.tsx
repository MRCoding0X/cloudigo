"use client";

import { Suspense, useState, type FormEvent } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import Link from "next/link";

import { useAuth, ApiError, type User } from "@/lib/auth-context";

const inputClass =
  "rounded-lg border bg-surface px-3 py-2.5 text-sm text-foreground transition-colors focus:border-brand";

function LoginForm() {
  const { login } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();

  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  function redirectAfterLogin(user: User) {
    const redirect = searchParams.get("redirect");
    if (redirect && redirect.startsWith("/")) {
      router.push(redirect);
      return;
    }
    router.push(user.role === "admin" ? "/admin" : "/");
  }

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setSubmitting(true);
    try {
      const user = await login(email, password);
      redirectAfterLogin(user);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Login failed");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <main className="relative flex flex-1 items-center justify-center overflow-hidden p-6 sm:p-8">
      <div className="pointer-events-none absolute -top-28 -left-28 h-72 w-72 animate-float-slow rounded-full bg-brand/20 blur-3xl" />
      <div className="pointer-events-none absolute -bottom-28 -right-28 h-72 w-72 animate-float rounded-full bg-brand/12 blur-3xl" />
      <div className="animate-fade-up relative w-full max-w-sm">
        <form
          onSubmit={handleSubmit}
          className="flex flex-col gap-4 rounded-2xl border bg-surface p-8 shadow-sm"
        >
          <h1 className="text-xl font-semibold">Log in</h1>

          <label className="flex flex-col gap-1.5 text-sm">
            Email
            <input
              type="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className={inputClass}
            />
          </label>

          <label className="flex flex-col gap-1.5 text-sm">
            Password
            <input
              type="password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className={inputClass}
            />
          </label>

          {error && <p className="text-sm text-red-500">{error}</p>}

          <button
            type="submit"
            disabled={submitting}
            className="mt-2 rounded-lg bg-brand px-4 py-2.5 text-sm font-medium text-white shadow-sm shadow-brand/20 transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-brand/30 active:translate-y-0 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50"
          >
            {submitting ? "Logging in…" : "Log in"}
          </button>

          <Link href="/forgot-password" className="text-center text-sm text-muted hover:text-foreground">
            Forgot password?
          </Link>
        </form>
      </div>
    </main>
  );
}

export default function LoginPage() {
  return (
    <Suspense>
      <LoginForm />
    </Suspense>
  );
}

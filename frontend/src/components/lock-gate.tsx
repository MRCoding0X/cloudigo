"use client";

import { useEffect } from "react";
import { usePathname, useRouter } from "next/navigation";

import { useAuth } from "@/lib/auth-context";

/**
 * Redirects to /login when the admin-configured lock_page setting requires
 * authentication for this page and no user is signed in. Renders nothing
 * while the redirect is pending so the gated content never flashes.
 */
export default function LockGate({ locked, children }: { locked: boolean; children: React.ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  const shouldRedirect = locked && !loading && !user;

  useEffect(() => {
    if (shouldRedirect) {
      router.replace(`/login?redirect=${encodeURIComponent(pathname)}`);
    }
  }, [shouldRedirect, router, pathname]);

  if (locked && (loading || !user)) return null;
  return <>{children}</>;
}

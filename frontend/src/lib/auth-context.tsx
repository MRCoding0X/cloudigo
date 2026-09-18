"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";

import { apiFetch, ApiError, apiFetchJSON, refreshAccessToken, setAccessToken } from "./api";

export type User = {
  id: string;
  email: string;
  role: "admin" | "user";
};

type AuthContextValue = {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<User>;
  logout: () => Promise<void>;
  setUser: (user: User) => void;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const didInit = useRef(false);

  useEffect(() => {
    // React Strict Mode intentionally mounts effects twice in dev, immediately
    // running this effect's cleanup in between. A second, concurrent
    // silent-refresh call would race the rotating refresh token (one call
    // rotates it, the other reuses the now-stale cookie and fails, clearing
    // the session the first call just established), so this must run at most
    // once per real page load. A ref survives Strict Mode's simulated
    // remount, unlike module state re-evaluated per mount — but note this
    // means the async work below deliberately has no cancellation/cleanup:
    // AuthProvider lives for the lifetime of the page, so there is no real
    // remount for it to race against.
    if (didInit.current) return;
    didInit.current = true;

    (async () => {
      const token = await refreshAccessToken();
      if (token) {
        try {
          const me = await apiFetchJSON<User>("/api/auth/me");
          setUser(me);
        } catch {
          setUser(null);
        }
      }
      setLoading(false);
    })();
  }, []);

  const login = useCallback(async (email: string, password: string) => {
    const data = await apiFetchJSON<{ accessToken: string; user: User }>(
      "/api/auth/login",
      { method: "POST", body: JSON.stringify({ email, password }) },
    );
    setAccessToken(data.accessToken);
    setUser(data.user);
    return data.user;
  }, []);

  const logout = useCallback(async () => {
    await apiFetch("/api/auth/logout", { method: "POST" });
    setAccessToken(null);
    setUser(null);
  }, []);

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, setUser }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used within an AuthProvider");
  return ctx;
}

export { ApiError };

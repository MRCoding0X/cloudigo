const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

let accessToken: string | null = null;

export function setAccessToken(token: string | null) {
  accessToken = token;
}

export function getAccessToken() {
  return accessToken;
}

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function rawFetch(path: string, options: RequestInit = {}): Promise<Response> {
  const headers = new Headers(options.headers);
  if (accessToken) headers.set("Authorization", `Bearer ${accessToken}`);
  // FormData needs the browser to set its own multipart boundary — never
  // force JSON on it (or on other binary body types).
  const isFormData = typeof FormData !== "undefined" && options.body instanceof FormData;
  if (options.body && !isFormData && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  return fetch(`${API_URL}${path}`, {
    ...options,
    headers,
    credentials: "include",
  });
}

async function doRefresh(): Promise<string | null> {
  const res = await rawFetch("/api/auth/refresh", { method: "POST" });
  if (!res.ok) {
    setAccessToken(null);
    return null;
  }
  const data = await res.json();
  setAccessToken(data.accessToken);
  return data.accessToken as string;
}

// The refresh token rotates on every use (old one is revoked server-side), so
// two concurrent refresh calls would race: the loser reuses an already-revoked
// cookie and fails, wiping out the token the winner just set. Callers share a
// single in-flight request instead of racing.
let refreshPromise: Promise<string | null> | null = null;

export function refreshAccessToken(): Promise<string | null> {
  if (!refreshPromise) {
    refreshPromise = doRefresh().finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

/**
 * Fetch wrapper that attaches the in-memory access token and transparently
 * retries once via a silent refresh if the access token has expired.
 */
export async function apiFetch(path: string, options: RequestInit = {}): Promise<Response> {
  let res = await rawFetch(path, options);

  if (res.status === 401 && path !== "/api/auth/refresh") {
    const newToken = await refreshAccessToken();
    if (newToken) {
      res = await rawFetch(path, options);
    }
  }

  return res;
}

export async function apiFetchJSON<T>(path: string, options: RequestInit = {}): Promise<T> {
  const res = await apiFetch(path, options);
  const body = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new ApiError(res.status, body.error ?? "Something went wrong");
  }
  return body as T;
}

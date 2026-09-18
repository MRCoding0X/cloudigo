const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const CHUNK_SIZE = 1024 * 1024; // 1MB, matches the backend default max_chunk_size_mb

const MAX_CHUNK_RETRIES = 4;
const RETRY_DELAYS_MS = [500, 1500, 3000, 5000];

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

/**
 * POSTs one chunk, retrying with backoff on network failure or a 5xx
 * response (a flaky connection or a momentarily overloaded server) — but
 * never on a 4xx, since that means the request itself is invalid and
 * retrying it verbatim would just fail the same way again. An abort is
 * never retried either; it propagates immediately so the caller's cancel
 * button actually cancels.
 */
async function uploadChunkWithRetry(
  url: string,
  init: RequestInit,
  fileName: string,
  onRetry?: (attempt: number) => void,
): Promise<void> {
  for (let attempt = 0; ; attempt++) {
    let res: Response;
    try {
      res = await fetch(url, init);
    } catch (err) {
      if (err instanceof DOMException && err.name === "AbortError") throw err;
      if (attempt >= MAX_CHUNK_RETRIES) throw err;
      onRetry?.(attempt + 1);
      await sleep(RETRY_DELAYS_MS[Math.min(attempt, RETRY_DELAYS_MS.length - 1)]);
      continue;
    }

    if (res.ok) return;

    if (res.status < 500) {
      const body = await res.json().catch(() => ({}));
      throw new Error(body.error ?? `Upload failed for ${fileName}`);
    }

    if (attempt >= MAX_CHUNK_RETRIES) {
      throw new Error(`Server error (${res.status})`);
    }
    onRetry?.(attempt + 1);
    await sleep(RETRY_DELAYS_MS[Math.min(attempt, RETRY_DELAYS_MS.length - 1)]);
  }
}

export async function createUploadSession(): Promise<{ uploadId: string; secretCode: string; shareCode: string }> {
  const res = await fetch(`${API_URL}/api/upload/session`, { method: "POST" });
  if (!res.ok) throw new Error("Could not start the upload");
  return res.json();
}

/**
 * Uploads one file in sequential, ordered chunks (required — the backend
 * detects "file complete" purely by on-disk size reaching the declared
 * total, which only holds for in-order delivery).
 */
export async function uploadFileInChunks(
  uploadId: string,
  fileId: string,
  file: File,
  signal?: AbortSignal,
  onRetry?: (attempt: number) => void,
): Promise<void> {
  const total = file.size;
  let offset = 0;

  // Zero-byte files still need one (empty) chunk request to create their DB row.
  do {
    const end = Math.min(offset + CHUNK_SIZE, total) - 1;
    const blob = file.slice(offset, end + 1);

    await uploadChunkWithRetry(
      `${API_URL}/api/upload/${uploadId}/files/${fileId}/chunk`,
      {
        method: "POST",
        headers: {
          "Content-Range": `bytes ${offset}-${end < offset ? offset : end}/${total}`,
          "X-File-Name": encodeURIComponent(file.name),
          "Content-Type": "application/octet-stream",
        },
        body: blob,
        signal,
      },
      file.name,
      onRetry,
    );

    offset += CHUNK_SIZE;
  } while (offset < total);
}

export interface RegisterInput {
  emailFrom?: string;
  message?: string;
  recipients?: string[];
  password?: string;
  destruct?: boolean;
  shareType: "link" | "mail";
  expireSeconds?: number | null;
}

export async function registerUpload(uploadId: string, input: RegisterInput) {
  const res = await fetch(`${API_URL}/api/upload/${uploadId}/register`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(input),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? "Could not register the upload");
  }
  return res.json();
}

export async function completeUpload(uploadId: string) {
  const res = await fetch(`${API_URL}/api/upload/${uploadId}/complete`, { method: "POST" });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? "Could not complete the upload");
  }
  return res.json();
}

export async function requestEmailVerification(email: string) {
  const res = await fetch(`${API_URL}/api/upload/verify-email/request`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email }),
  });
  if (!res.ok) throw new Error("Could not send verification code");
}

export async function confirmEmailVerification(email: string, code: string) {
  const res = await fetch(`${API_URL}/api/upload/verify-email/confirm`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, code }),
  });
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? "Incorrect verification code");
  }
}

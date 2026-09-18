"use client";

import { useEffect, useRef } from "react";

import { getAccessToken } from "./api";

export type UploadEvent =
  | { type: "chunk.progress"; fileId: string; fileName: string; receivedBytes: number; totalBytes: number }
  | { type: "file.complete"; fileId: string; fileName: string; size: number }
  | { type: "upload.ready"; uploadId: string }
  | { type: "upload.destroyed"; uploadId: string }
  | { type: "download.happened"; email: string; downloadedAt: string };

export type AdminEvent =
  | { type: "upload.ready"; uploadId: string; fileCount: number; size: number }
  | { type: "upload.destroyed"; uploadId: string }
  | { type: "download.happened"; uploadId: string; email: string; downloadedAt: string };

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

function wsUrl(uploadId: string): string {
  return API_URL.replace(/^http/, "ws") + `/ws/upload/${uploadId}`;
}

/**
 * Subscribes to an upload's live event room. Reconnecting whenever uploadId
 * changes is the actually-intended behavior here (unlike the auth-refresh
 * mount effect), so a plain effect + cleanup is correct — no Strict Mode
 * once-only guard needed.
 */
export function useUploadSocket(uploadId: string | null | undefined, onEvent: (event: UploadEvent) => void) {
  const handlerRef = useRef(onEvent);

  useEffect(() => {
    handlerRef.current = onEvent;
  }, [onEvent]);

  useEffect(() => {
    if (!uploadId) return;

    const socket = new WebSocket(wsUrl(uploadId));
    socket.onmessage = (e) => {
      try {
        handlerRef.current(JSON.parse(e.data));
      } catch {
        // ignore malformed frames
      }
    };

    return () => socket.close();
  }, [uploadId]);
}

/**
 * Subscribes to the admin dashboard's shared live-activity room. The browser
 * WebSocket API can't send an Authorization header, so the (short-lived)
 * access token travels as a query param instead — see internal/handler/ws_handler.go.
 * Connects once per mount; if the access token rotates via silent refresh
 * while the dashboard stays open, this socket keeps using the token it
 * connected with rather than reconnecting (acceptable for now — the
 * connection itself doesn't expire once established).
 */
export function useAdminSocket(enabled: boolean, onEvent: (event: AdminEvent) => void) {
  const handlerRef = useRef(onEvent);

  useEffect(() => {
    handlerRef.current = onEvent;
  }, [onEvent]);

  useEffect(() => {
    if (!enabled) return;
    const token = getAccessToken();
    if (!token) return;

    const socket = new WebSocket(`${API_URL.replace(/^http/, "ws")}/ws/admin?token=${encodeURIComponent(token)}`);
    socket.onmessage = (e) => {
      try {
        handlerRef.current(JSON.parse(e.data));
      } catch {
        // ignore malformed frames
      }
    };

    return () => socket.close();
  }, [enabled]);
}

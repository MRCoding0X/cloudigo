"use client";

import { createContext, useContext, type ReactNode } from "react";

import type { PublicSettings } from "./branding";

const SettingsContext = createContext<PublicSettings | null>(null);

export function SettingsProvider({
  settings,
  children,
}: {
  settings: PublicSettings;
  children: ReactNode;
}) {
  return <SettingsContext.Provider value={settings}>{children}</SettingsContext.Provider>;
}

export function useSettings() {
  const ctx = useContext(SettingsContext);
  if (!ctx) throw new Error("useSettings must be used within a SettingsProvider");
  return ctx;
}

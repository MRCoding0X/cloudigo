"use client";

import Link from "next/link";

import { useSettings } from "@/lib/settings-context";

const SOCIAL_LINKS: { key: keyof ReturnType<typeof useSettings>["social"]; label: string }[] = [
  { key: "facebook", label: "Facebook" },
  { key: "twitter", label: "Twitter" },
  { key: "instagram", label: "Instagram" },
  { key: "github", label: "GitHub" },
];

export default function SiteFooter() {
  const settings = useSettings();

  const socialEntries = SOCIAL_LINKS.filter((s) => settings.social[s.key]);
  if (!settings.contactEnabled && socialEntries.length === 0) return null;

  return (
    <footer className="flex flex-wrap items-center justify-center gap-x-5 gap-y-1 border-t p-4 text-xs text-muted">
      {settings.contactEnabled && (
        <Link href="/contact" className="transition-colors hover:text-foreground">
          Contact us
        </Link>
      )}
      {socialEntries.map((s) => (
        <a
          key={s.key}
          href={settings.social[s.key]}
          target="_blank"
          rel="noopener noreferrer"
          className="transition-colors hover:text-foreground"
        >
          {s.label}
        </a>
      ))}
    </footer>
  );
}

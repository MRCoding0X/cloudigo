"use client";

import { useEffect, useState, type ReactNode } from "react";
import { useRouter, usePathname } from "next/navigation";
import Link from "next/link";

import { useAuth } from "@/lib/auth-context";
import { useAdminSocket, type AdminEvent } from "@/lib/ws";
import {
  DashboardIcon,
  UploadsIcon,
  DownloadsIcon,
  UsersIcon,
  PagesIcon,
  BackgroundsIcon,
  SettingsIcon,
  MailIcon,
  SystemIcon,
  AuditIcon,
  BlockIcon,
  LogoutIcon,
  ProfileIcon,
  Spinner,
} from "@/components/icons";

const NAV = [
  { href: "/admin", label: "Dashboard", icon: DashboardIcon },
  { href: "/admin/uploads", label: "Uploads", icon: UploadsIcon },
  { href: "/admin/downloads", label: "Downloads", icon: DownloadsIcon },
  { href: "/admin/users", label: "Users", icon: UsersIcon },
  { href: "/admin/pages", label: "Pages", icon: PagesIcon },
  { href: "/admin/backgrounds", label: "Backgrounds", icon: BackgroundsIcon },
  { href: "/admin/settings", label: "Settings", icon: SettingsIcon },
  { href: "/admin/email-templates", label: "Email templates", icon: MailIcon },
  { href: "/admin/system", label: "System info", icon: SystemIcon },
  { href: "/admin/audit", label: "Audit log", icon: AuditIcon },
  { href: "/admin/blocked-ips", label: "Blocked IPs", icon: BlockIcon },
];

export default function AdminLayout({ children }: { children: ReactNode }) {
  const { user, loading, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const [notifications, setNotifications] = useState<string[]>([]);

  useEffect(() => {
    if (!loading && (!user || user.role !== "admin")) {
      router.replace("/login");
    }
  }, [loading, user, router]);

  useAdminSocket(!loading && user?.role === "admin", (event: AdminEvent) => {
    let text = "";
    if (event.type === "upload.ready") text = `New upload ready: ${event.uploadId} (${event.fileCount} files, ${event.size} B)`;
    else if (event.type === "upload.destroyed") text = `Upload deleted: ${event.uploadId}`;
    else if (event.type === "download.happened")
      text = event.email ? `Download: ${event.uploadId} by ${event.email}` : `Download: ${event.uploadId}`;
    const time = new Date().toLocaleTimeString("en-US");
    setNotifications((prev) => [`${time} — ${text}`, ...prev].slice(0, 20));
  });

  if (loading || !user || user.role !== "admin") {
    return (
      <main className="flex flex-1 items-center justify-center p-8 text-muted">
        <Spinner />
      </main>
    );
  }

  return (
    <div className="flex flex-1 bg-surface-muted">
      <aside className="flex w-60 shrink-0 flex-col border-r bg-surface p-4">
        <div className="mb-5 flex items-center gap-2 px-1">
          <div className="flex h-7 w-7 items-center justify-center rounded-md bg-brand text-xs font-bold text-white shadow-sm">
            C
          </div>
          <p className="font-semibold">Cloudigo</p>
        </div>
        <nav className="flex flex-1 flex-col gap-0.5 text-sm">
          {NAV.map((item) => {
            const active = pathname === item.href;
            const Icon = item.icon;
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`flex items-center gap-2.5 rounded-lg px-2.5 py-2 transition-all ${
                  active
                    ? "bg-brand-soft font-medium text-brand"
                    : "text-muted hover:translate-x-0.5 hover:bg-surface-muted hover:text-foreground"
                }`}
              >
                <Icon className="shrink-0" />
                {item.label}
              </Link>
            );
          })}
        </nav>
        <div className="mt-4 flex flex-col gap-0.5 border-t pt-4">
          <Link
            href="/admin/account"
            className={`flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm transition-colors ${
              pathname === "/admin/account"
                ? "bg-brand-soft font-medium text-brand"
                : "text-muted hover:bg-surface-muted hover:text-foreground"
            }`}
          >
            <ProfileIcon className="shrink-0" />
            <span className="truncate">{user.email}</span>
          </Link>
          <button
            onClick={async () => {
              await logout();
              router.push("/login");
            }}
            className="flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm text-muted transition-colors hover:bg-surface-muted hover:text-foreground"
          >
            <LogoutIcon className="shrink-0" />
            Log out
          </button>
        </div>
      </aside>

      <div className="flex flex-1 flex-col overflow-hidden">
        <main className="flex-1 overflow-y-auto p-6 lg:p-8">
          <div className="mx-auto max-w-6xl">{children}</div>
        </main>
        <div className="max-h-32 overflow-y-auto border-t bg-surface px-6 py-3 text-xs lg:px-8">
          <p className="mb-1 font-medium">Live activity</p>
          {notifications.length === 0 ? (
            <p className="text-muted">No events yet.</p>
          ) : (
            notifications.map((n, i) => (
              <p key={i} className="text-muted">
                {n}
              </p>
            ))
          )}
        </div>
      </div>
    </div>
  );
}

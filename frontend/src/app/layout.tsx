import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";

import "./globals.css";
import { AuthProvider } from "@/lib/auth-context";
import { getPublicSettings } from "@/lib/branding";
import { SettingsProvider } from "@/lib/settings-context";

const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

export async function generateMetadata(): Promise<Metadata> {
  const settings = await getPublicSettings();
  return {
    title: settings.siteName,
    description: "Modern file sharing, built with Go and Next.js.",
    icons: settings.faviconPath ? { icon: `${apiUrl}${settings.faviconPath}` } : undefined,
  };
}

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const settings = await getPublicSettings();

  return (
    <html
      lang="en"
      className={`${geistSans.variable} ${geistMono.variable} h-full antialiased`}
      style={
        {
          "--brand-accent": settings.themeColor,
          "--brand-accent-secondary": settings.themeColorSecondary,
        } as React.CSSProperties
      }
    >
      <body className="min-h-full flex flex-col">
        <SettingsProvider settings={settings}>
          <AuthProvider>{children}</AuthProvider>
        </SettingsProvider>
      </body>
    </html>
  );
}

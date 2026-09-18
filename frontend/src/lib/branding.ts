export type PublicSettings = {
  siteName: string;
  themeColor: string;
  themeColorSecondary: string;
  logoPath: string;
  faviconPath: string;
  contactEnabled: boolean;
  lockPage: "false" | "both" | "upload" | "download";
  acceptTerms: boolean;
  passwordEnabled: boolean;
  destructEnabled: boolean;
  shareEnabled: boolean;
  maxUploadSizeMB: number;
  maxFiles: number;
  maxRecipients: number;
  social: {
    facebook: string;
    twitter: string;
    instagram: string;
    github: string;
  };
};

const FALLBACK: PublicSettings = {
  siteName: "Cloudigo",
  themeColor: "#4f46e5",
  themeColorSecondary: "#111827",
  logoPath: "",
  faviconPath: "",
  contactEnabled: false,
  lockPage: "false",
  acceptTerms: false,
  passwordEnabled: true,
  destructEnabled: true,
  shareEnabled: true,
  maxUploadSizeMB: 0,
  maxFiles: 0,
  maxRecipients: 0,
  social: { facebook: "", twitter: "", instagram: "", github: "" },
};

/**
 * Server-side fetch of the public (non-sensitive) branding settings —
 * site name, theme colors, logo. Used by the root layout to set the page
 * title and CSS custom properties per request, so admin-configured branding
 * (Settings > Appearance) takes effect without a rebuild.
 */
export async function getPublicSettings(): Promise<PublicSettings> {
  const apiUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
  try {
    const res = await fetch(`${apiUrl}/api/settings/public`, { next: { revalidate: 60 } });
    if (!res.ok) return FALLBACK;
    const data = await res.json();
    return { ...FALLBACK, ...data };
  } catch {
    return FALLBACK;
  }
}

const EXTENSION_COLORS: Record<string, string> = {
  // documents
  pdf: "bg-red-500",
  doc: "bg-blue-500",
  docx: "bg-blue-500",
  odt: "bg-blue-500",
  rtf: "bg-blue-500",
  // spreadsheets
  xls: "bg-green-600",
  xlsx: "bg-green-600",
  csv: "bg-green-600",
  // presentations
  ppt: "bg-orange-500",
  pptx: "bg-orange-500",
  // archives
  zip: "bg-amber-600",
  rar: "bg-amber-600",
  "7z": "bg-amber-600",
  tar: "bg-amber-600",
  gz: "bg-amber-600",
  // audio
  mp3: "bg-purple-500",
  wav: "bg-purple-500",
  flac: "bg-purple-500",
  m4a: "bg-purple-500",
  aac: "bg-purple-500",
  ogg: "bg-purple-500",
  // video
  mp4: "bg-pink-500",
  mov: "bg-pink-500",
  avi: "bg-pink-500",
  mkv: "bg-pink-500",
  webm: "bg-pink-500",
  // images (only shown when no real thumbnail is available)
  jpg: "bg-teal-500",
  jpeg: "bg-teal-500",
  png: "bg-teal-500",
  gif: "bg-teal-500",
  svg: "bg-teal-500",
  webp: "bg-teal-500",
  // text/code
  txt: "bg-gray-500",
  md: "bg-gray-500",
  json: "bg-gray-500",
};

function fileExtension(fileName: string): string {
  const dot = fileName.lastIndexOf(".");
  if (dot === -1 || dot === fileName.length - 1) return "";
  return fileName.slice(dot + 1).toLowerCase();
}

const SIZES = {
  sm: { box: "h-7 w-7", svg: 16, badge: "text-[7px] px-[3px]" },
  md: { box: "h-9 w-9", svg: 20, badge: "text-[8px] px-1" },
} as const;

/**
 * A generic document glyph with a small colored pill showing the file's
 * extension — the color is a category hint (audio, video, archive, etc.),
 * not meant to be an exhaustive per-app icon set.
 */
export function FileTypeIcon({
  fileName,
  size = "md",
  className = "",
}: {
  fileName: string;
  size?: keyof typeof SIZES;
  className?: string;
}) {
  const ext = fileExtension(fileName);
  const color = EXTENSION_COLORS[ext] ?? "bg-muted";
  const label = ext ? ext.slice(0, 4).toUpperCase() : "";
  const { box, svg, badge } = SIZES[size];

  return (
    <span className={`relative flex ${box} shrink-0 items-center justify-center rounded bg-surface ${className}`}>
      <svg width={svg} height={svg} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" strokeLinejoin="round" className="text-muted">
        <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z" />
        <path d="M14 2v6h6" />
      </svg>
      {label && (
        <span
          className={`absolute -bottom-1 left-1/2 -translate-x-1/2 rounded py-px font-bold leading-tight tracking-tight text-white shadow-sm ${color} ${badge}`}
        >
          {label}
        </span>
      )}
    </span>
  );
}

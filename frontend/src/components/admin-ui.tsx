import { Spinner, ChevronLeftIcon, ChevronRightIcon } from "@/components/icons";

export function LoadingRow({ colSpan }: { colSpan: number }) {
  return (
    <tr>
      <td colSpan={colSpan} className="p-8 text-center text-muted">
        <Spinner className="mx-auto" />
      </td>
    </tr>
  );
}

export const inputClass =
  "rounded-lg border bg-surface px-3 py-2 text-sm transition-all focus:border-brand focus:shadow-[0_0_0_3px_var(--brand-accent-soft)]";

export const primaryButtonClass =
  "rounded-lg bg-brand px-3.5 py-2 text-sm font-medium text-white shadow-sm shadow-brand/20 transition-all hover:-translate-y-0.5 hover:shadow-md hover:shadow-brand/30 active:translate-y-0 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-50";

export const secondaryButtonClass =
  "rounded-lg border bg-surface px-3.5 py-2 text-sm font-medium transition-all hover:-translate-y-0.5 hover:bg-surface-muted hover:shadow-sm active:translate-y-0 active:scale-[0.98] disabled:pointer-events-none disabled:opacity-40";

const STATUS_STYLES: Record<string, string> = {
  ready: "bg-green-600/10 text-green-700 dark:text-green-400",
  processing: "bg-amber-500/10 text-amber-700 dark:text-amber-400",
  destroyed: "bg-red-500/10 text-red-600 dark:text-red-400",
  inactive: "bg-surface-muted text-muted",
  admin: "bg-brand-soft text-brand",
  user: "bg-surface-muted text-muted",
  enabled: "bg-green-600/10 text-green-700 dark:text-green-400",
  disabled: "bg-surface-muted text-muted",
};

export function StatusBadge({ value }: { value: string }) {
  const style = STATUS_STYLES[value] ?? "bg-surface-muted text-muted";
  return (
    <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium capitalize ${style}`}>
      {value}
    </span>
  );
}

export function Card({
  children,
  className = "",
  interactive = false,
  style,
}: {
  children: React.ReactNode;
  className?: string;
  interactive?: boolean;
  style?: React.CSSProperties;
}) {
  return (
    <div
      style={style}
      className={`rounded-xl border bg-surface shadow-sm transition-all ${
        interactive ? "hover:-translate-y-0.5 hover:shadow-md" : ""
      } ${className}`}
    >
      {children}
    </div>
  );
}

export function PageHeader({ title, description, action }: { title: string; description?: string; action?: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4">
      <div>
        <h1 className="text-2xl font-semibold">{title}</h1>
        {description && <p className="mt-1 text-sm text-muted">{description}</p>}
      </div>
      {action}
    </div>
  );
}

function getPageItems(page: number, totalPages: number): (number | "…")[] {
  if (totalPages <= 7) {
    return Array.from({ length: totalPages }, (_, i) => i + 1);
  }
  const items: (number | "…")[] = [1];
  if (page > 3) items.push("…");
  for (let i = Math.max(2, page - 1); i <= Math.min(totalPages - 1, page + 1); i++) {
    items.push(i);
  }
  if (page < totalPages - 2) items.push("…");
  items.push(totalPages);
  return items;
}

const pagerNavButtonClass =
  "flex h-8 w-8 items-center justify-center rounded-lg text-muted transition-all hover:bg-surface-muted hover:text-foreground active:scale-90 disabled:pointer-events-none disabled:opacity-30";

export function Pagination({
  page,
  totalPages,
  total,
  onChange,
}: {
  page: number;
  totalPages: number;
  total: number;
  onChange: (page: number) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-4">
      <span className="text-sm text-muted">{total} total</span>
      <div className="flex items-center gap-1">
        <button
          disabled={page <= 1}
          onClick={() => onChange(page - 1)}
          aria-label="Previous page"
          className={pagerNavButtonClass}
        >
          <ChevronLeftIcon />
        </button>

        {getPageItems(page, totalPages).map((item, i) =>
          item === "…" ? (
            <span key={`ellipsis-${i}`} className="flex h-8 w-8 items-center justify-center text-sm text-muted">
              …
            </span>
          ) : (
            <button
              key={item}
              onClick={() => onChange(item)}
              disabled={item === page}
              className={
                item === page
                  ? "flex h-8 min-w-8 items-center justify-center rounded-lg bg-brand px-2.5 text-sm font-semibold text-white shadow-sm"
                  : "flex h-8 min-w-8 items-center justify-center rounded-lg px-2.5 text-sm text-muted transition-colors hover:bg-surface-muted hover:text-foreground"
              }
            >
              {item}
            </button>
          ),
        )}

        <button
          disabled={page >= totalPages}
          onClick={() => onChange(page + 1)}
          aria-label="Next page"
          className={pagerNavButtonClass}
        >
          <ChevronRightIcon />
        </button>
      </div>
    </div>
  );
}

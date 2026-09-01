import type { Langs } from "@/types";
import { cn } from "@/utils/cn";

const PALETTE = [
  "bg-blue-500",
  "bg-emerald-500",
  "bg-violet-500",
  "bg-amber-500",
  "bg-rose-500",
  "bg-cyan-500",
  "bg-fuchsia-500",
  "bg-lime-500",
  "bg-indigo-500",
];

interface LangBarProps {
  langs: Langs[];
  totalSloc: number;
  className?: string;
}

export function LangBar({ langs, totalSloc, className }: LangBarProps) {
  if (!langs.length || totalSloc === 0) {
    return <p className="text-sm text-muted-foreground">Нет данных по языкам</p>;
  }
  const sorted = [...langs].sort((a, b) => b.sloc - a.sloc);

  return (
    <div className={cn("space-y-3", className)}>
      <div className="flex h-3 w-full overflow-hidden rounded-full bg-muted">
        {sorted.map((l, i) => {
          const pct = (l.sloc / totalSloc) * 100;
          return (
            <span
              key={l.lang}
              className={cn("h-full transition-all", PALETTE[i % PALETTE.length])}
              style={{ width: `${pct}%` }}
              title={`${l.lang} — ${pct.toFixed(1)}%`}
            />
          );
        })}
      </div>
      <ul className="flex flex-wrap gap-x-4 gap-y-1 text-xs">
        {sorted.map((l, i) => {
          const pct = (l.sloc / totalSloc) * 100;
          return (
            <li key={l.lang} className="flex items-center gap-1.5">
              <span
                className={cn("h-2 w-2 rounded-full", PALETTE[i % PALETTE.length])}
                aria-hidden
              />
              <span className="font-medium">{l.lang}</span>
              <span className="text-muted-foreground">
                {l.sloc.toLocaleString("ru-RU")} ({pct.toFixed(1)}%)
              </span>
            </li>
          );
        })}
      </ul>
    </div>
  );
}

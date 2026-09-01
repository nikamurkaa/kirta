import { useMemo, useState } from "react";
import { Package, Search } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import type { Libraries } from "@/types";

interface LibrariesDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  libraries: Libraries[];
}

export function LibrariesDialog({ open, onOpenChange, libraries }: LibrariesDialogProps) {
  const [query, setQuery] = useState("");

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    const sorted = [...libraries].sort((a, b) => a.package.localeCompare(b.package));
    if (!q) return sorted;
    return sorted.filter(
      (l) => l.package.toLowerCase().includes(q) || l.version.toLowerCase().includes(q),
    );
  }, [libraries, query]);

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next);
        if (!next) setQuery("");
      }}
    >
      <DialogContent className="max-w-md sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Package className="h-5 w-5 text-primary" />
            Библиотеки проекта
          </DialogTitle>
          <DialogDescription>Полный список зависимостей, найденных в репозитории.</DialogDescription>
        </DialogHeader>

        <div className="space-y-3 px-6">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              autoFocus
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Поиск по названию или версии"
              className="pl-9"
            />
          </div>

          <div className="max-h-[60vh] overflow-auto scrollbar-thin rounded-md border">
            {filtered.length === 0 ? (
              <div className="p-6 text-center text-sm text-muted-foreground">Ничего не найдено</div>
            ) : (
              <ul className="divide-y">
                {filtered.map((lib) => (
                  <li
                    key={`${lib.package}@${lib.version}`}
                    className="flex items-center justify-between gap-3 px-4 py-2.5 text-sm hover:bg-muted/50"
                  >
                    <span className="truncate font-medium">{lib.package}</span>
                    <span className="shrink-0 font-mono text-xs text-muted-foreground">
                      {lib.version}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>

        <div className="flex items-center justify-between border-t px-6 py-3 text-xs text-muted-foreground">
          <span>
            Всего: <span className="font-medium text-foreground">{libraries.length}</span>
          </span>
          <span>
            Показано: <span className="font-medium text-foreground">{filtered.length}</span>
          </span>
        </div>
      </DialogContent>
    </Dialog>
  );
}

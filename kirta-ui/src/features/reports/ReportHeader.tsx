import { useState } from "react";
import { ArrowRight, FileText, FolderGit2, Layers, Package } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { ScaReport } from "@/types";
import { LangBar } from "./LangBar";
import { LibrariesDialog } from "./LibrariesDialog";

interface ReportHeaderProps {
  report: ScaReport;
}

export function ReportHeader({ report }: ReportHeaderProps) {
  const [librariesOpen, setLibrariesOpen] = useState(false);
  const [manifestOpen, setManifestOpen] = useState(false);
  const manifestText = report.manifest.trim();

  return (
    <Card>
      <CardContent className="grid gap-6 p-5 sm:p-6 md:grid-cols-2 lg:grid-cols-3">
        <div className="space-y-2">
          <div className="flex items-center gap-2 text-xs uppercase tracking-wider text-muted-foreground">
            <FolderGit2 className="h-3.5 w-3.5" />
            Репозиторий
          </div>
          <h2 className="text-2xl font-semibold tracking-tight sm:text-3xl">{report.repository_name}</h2>
          <div className="flex items-baseline gap-2 pt-2">
            <span className="text-3xl font-semibold tabular-nums">
              {report.total_sloc.toLocaleString("ru-RU")}
            </span>
            <span className="text-sm text-muted-foreground">SLOC</span>
          </div>
          <div className="text-xs text-muted-foreground">Scan ID: #{report.scan_id}</div>
        </div>

        <div className="space-y-3">
          <div className="flex items-center gap-2 text-xs uppercase tracking-wider text-muted-foreground">
            <Layers className="h-3.5 w-3.5" />
            Состав по языкам
          </div>
          <LangBar langs={report.langs} totalSloc={report.total_sloc} />
        </div>

        <div className="space-y-3 md:col-span-2 lg:col-span-1">
          <button
            type="button"
            onClick={() => setLibrariesOpen(true)}
            className="group flex w-full flex-col gap-3 rounded-xl border bg-muted/30 p-4 text-left transition-all hover:border-primary/40 hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            aria-label="Открыть полный список библиотек"
          >
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/15 text-primary">
                  <Package className="h-5 w-5" />
                </span>
                <div>
                  <div className="text-base font-semibold">Библиотеки</div>
                  <div className="text-xs text-muted-foreground">
                    Зависимостей: {report.libraries.length}
                  </div>
                </div>
              </div>
              <ArrowRight className="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-primary" />
            </div>
            <div className="text-xs text-muted-foreground">Нажмите, чтобы открыть список</div>
          </button>

          <button
            type="button"
            onClick={() => setManifestOpen(true)}
            className="group flex w-full flex-col gap-2 rounded-xl border bg-muted/30 p-4 text-left transition-all hover:border-primary/40 hover:bg-primary/5 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            aria-label="Открыть файл зависимостей"
          >
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2">
                <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/15 text-primary">
                  <FileText className="h-5 w-5" />
                </span>
                <div>
                  <div className="text-base font-semibold">Файл зависимостей</div>
                  <div className="text-xs text-muted-foreground">Нажмите, чтобы открыть</div>
                </div>
              </div>
              <ArrowRight className="h-4 w-4 shrink-0 text-muted-foreground transition-transform group-hover:translate-x-0.5 group-hover:text-primary" />
            </div>
          </button>
        </div>
      </CardContent>

      <LibrariesDialog
        open={librariesOpen}
        onOpenChange={setLibrariesOpen}
        libraries={report.libraries}
      />

      <Dialog open={manifestOpen} onOpenChange={setManifestOpen}>
        <DialogContent className="max-h-[90vh] max-w-3xl overflow-hidden">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5 text-primary" />
              Файл зависимостей
            </DialogTitle>
            <DialogDescription>
              Содержимое файла зависимостей, переданного в сканирование.
            </DialogDescription>
          </DialogHeader>
          <div className="max-h-[65vh] overflow-auto rounded-md border bg-zinc-950 p-3">
            <pre className="whitespace-pre-wrap break-words font-mono text-xs leading-6 text-zinc-100">
              {manifestText || "Файл зависимостей не найден в архиве."}
            </pre>
          </div>
        </DialogContent>
      </Dialog>
    </Card>
  );
}

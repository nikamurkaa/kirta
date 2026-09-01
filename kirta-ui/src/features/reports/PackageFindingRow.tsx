import { useState } from "react";
import { Network } from "lucide-react";
import { ExploitabilityPillSmall } from "@/features/sca/ExploitabilityPill";
import { CallMapPanel } from "@/features/sca/CallMapPanel";
import { SafeVersionsDisclosure } from "@/features/sca/SafeVersionsDisclosure";
import { SeverityBadge } from "@/features/sca/SeverityBadge";
import { Button } from "@/components/ui/button";
import { useFindingExplanation, useGraphByLibrary } from "@/hooks";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { Finding } from "@/types";
import { cn } from "@/utils/cn";

export function PackageFindingRow({ finding, scanId }: { finding: Finding; scanId: number }) {
  const [mapOpen, setMapOpen] = useState(false);
  const [explanationError, setExplanationError] = useState<string | null>(null);
  const explanation = finding.explanation.trim();
  const hasExplanation = explanation.length > 0 && explanation !== ":";
  const isUnknown = finding.exploitable === "unknown";

  const { data: graph, isLoading: isGraphLoading, isError: isGraphError } = useGraphByLibrary(
    mapOpen
      ? {
          scanId,
          packageName: finding.package,
          version: finding.version,
        }
      : undefined,
  );
  const explanationMutation = useFindingExplanation();

  const cveLabel = finding.cve.filter((cve) => cve.trim()).join(", ") || "NO-CVE";
  const findingTitle = `${cveLabel} (${finding.package}@${finding.version})`;
  const findingToneClass =
    finding.exploitable === "exploitable"
      ? "border-[hsl(var(--severity-critical))] bg-[hsl(var(--severity-critical)/0.06)]"
      : finding.exploitable === "not_exploitable"
        ? "border-[hsl(var(--status-ok))] bg-[hsl(var(--status-ok)/0.05)]"
        : "border-border bg-muted/20";

  function handleGetExplanation() {
    setExplanationError(null);
    explanationMutation.mutate(
      { scanId, findingId: finding.id },
      {
        onError: (error) => {
          const message =
            error instanceof Error
              ? error.message
              : "Не удалось получить признаки эксплуатируемости.";
          setExplanationError(message);
        },
      },
    );
  }

  return (
    <>
      <div
        className={cn(
          "flex flex-wrap items-start justify-between gap-3 rounded-xl border-2 bg-card p-4 shadow-sm",
          findingToneClass,
        )}
      >
        <div className="min-w-0 flex-1 space-y-3">
          <div className="flex flex-wrap items-center gap-2">
            <span className="font-mono text-xs text-muted-foreground">#{finding.id}</span>
            <h3 className="truncate font-mono text-base font-semibold sm:text-lg">{findingTitle}</h3>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <SeverityBadge value={finding.severity} />
            <span className="text-xs tabular-nums text-muted-foreground">{finding.cve.length} CVE</span>
            <ExploitabilityPillSmall exploitable={finding.exploitable} />
          </div>
          {finding.description ? (
            <p className="line-clamp-3 text-sm leading-relaxed text-foreground/90">
              {finding.description}
            </p>
          ) : (
            <p className="text-sm italic text-muted-foreground">Нет описания</p>
          )}
          {hasExplanation ? (
            <div className="rounded-md border border-primary/20 bg-primary/5 p-3 text-sm">
              <div className="mb-1 text-xs font-semibold uppercase tracking-wider text-primary">
                Признаки эксплуатируемости
              </div>
              <p className="text-foreground/90">{explanation}</p>
            </div>
          ) : null}
          {isUnknown ? (
            <div className="space-y-2">
              <Button
                type="button"
                size="sm"
                variant="secondary"
                onClick={handleGetExplanation}
                disabled={explanationMutation.isPending}
              >
                {explanationMutation.isPending
                  ? "Получаем признаки..."
                  : "Получить признаки эксплуатируемости"}
              </Button>
              {explanationError ? (
                <p className="text-sm text-destructive" role="alert">
                  {explanationError}
                </p>
              ) : null}
            </div>
          ) : null}
          <SafeVersionsDisclosure packageName={finding.package} versions={finding.fixed_version} />
        </div>
        <div className="flex shrink-0 flex-col items-stretch gap-2 sm:items-end">
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="gap-2"
            title="Открыть карту вызовов"
            onClick={() => setMapOpen(true)}
          >
            <Network className="h-4 w-4" />
            Карта вызовов
          </Button>
        </div>
      </div>

      <Dialog open={mapOpen} onOpenChange={setMapOpen}>
        <DialogContent className="flex max-h-[92vh] w-[96vw] max-w-7xl flex-col gap-0 overflow-hidden p-0">
          <DialogHeader className="shrink-0 space-y-2 border-b border-slate-700/70 bg-slate-950/85 p-6 pb-5">
            <DialogTitle className="font-mono text-xl text-slate-100 sm:text-3xl">
              Карта вызовов: {finding.package}
              <span className="text-slate-400"> @{finding.version}</span>
            </DialogTitle>
            <DialogDescription className="text-base text-slate-300 sm:text-lg">
              Файлы и вызовы, связанные с библиотекой в этом сканировании. Нажмите файл, чтобы
              открыть исходник.
            </DialogDescription>
          </DialogHeader>
          <div className="min-h-0 flex-1 overflow-y-auto bg-slate-950/65 px-6 pb-6">
            {isGraphLoading ? (
              <p className="pt-4 text-sm text-slate-400">Загружаем call_map...</p>
            ) : isGraphError ? (
              <p className="pt-4 text-sm text-red-400">
                Не удалось получить call_map для этой библиотеки.
              </p>
            ) : (
              <CallMapPanel
                embedded
                packageName={finding.package}
                version={finding.version}
                callMap={graph?.call_map ?? []}
                scanId={scanId}
              />
            )}
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

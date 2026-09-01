import { useMemo, useState } from "react";
import { Filter, Search } from "lucide-react";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { EmptyState } from "@/components/ui/empty-state";
import { PackageFindingRow } from "./PackageFindingRow";
import { normalizeSeverity, severityRank, type SeverityLevel } from "@/utils/severity";
import type { Finding } from "@/types";

function findingMatchesQuery(f: Finding, q: string): boolean {
  if (!q) return true;
  const ql = q.toLowerCase();
  if (f.package.toLowerCase().includes(ql)) return true;
  if (f.cve.some((c) => c.toLowerCase().includes(ql))) return true;
  if (f.description.toLowerCase().includes(ql)) return true;
  if (f.explanation.toLowerCase().includes(ql)) return true;
  return false;
}

interface ScaFindingsTabProps {
  findings: Finding[];
  scanId: number;
}

const SEVERITY_FILTERS: { value: SeverityLevel | "all"; label: string }[] = [
  { value: "all", label: "Все" },
  { value: "critical", label: "Critical" },
  { value: "high", label: "High" },
  { value: "medium", label: "Medium" },
  { value: "low", label: "Low" },
];

export function ScaFindingsTab({ findings, scanId }: ScaFindingsTabProps) {
  const [query, setQuery] = useState("");
  const [severity, setSeverity] = useState<SeverityLevel | "all">("all");

  const stats = useMemo(() => {
    const counters: Record<SeverityLevel, number> = {
      critical: 0,
      high: 0,
      medium: 0,
      low: 0,
      unknown: 0,
    };
    for (const f of findings) {
      counters[normalizeSeverity(f.severity)]++;
    }
    return counters;
  }, [findings]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    return [...findings]
      .filter((f) => {
        if (severity !== "all" && normalizeSeverity(f.severity) !== severity) return false;
        return findingMatchesQuery(f, q);
      })
      .sort((a, b) => severityRank(a.severity) - severityRank(b.severity));
  }, [findings, query, severity]);

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative min-w-[220px] flex-1">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Поиск по пакету, CVE или описанию"
            className="pl-9"
          />
        </div>
        <div className="flex flex-wrap items-center gap-1.5">
          <Filter className="mr-1 h-4 w-4 text-muted-foreground" />
          {SEVERITY_FILTERS.map((f) => (
            <Button
              key={f.value}
              type="button"
              size="sm"
              variant={severity === f.value ? "default" : "outline"}
              onClick={() => setSeverity(f.value)}
            >
              {f.label}
              {f.value !== "all" ? (
                <Badge variant="outline" className="ml-1 px-1.5 py-0 font-mono text-[10px]">
                  {stats[f.value]}
                </Badge>
              ) : null}
            </Button>
          ))}
        </div>
      </div>

      {filtered.length === 0 ? (
        <EmptyState
          title="Findings не найдены"
          description="Попробуйте сбросить фильтры или поисковый запрос."
        />
      ) : (
        <div className="space-y-4">
          {filtered.map((f) => (
            <PackageFindingRow key={f.id} finding={f} scanId={scanId} />
          ))}
        </div>
      )}
    </div>
  );
}

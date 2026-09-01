import { Lock } from "lucide-react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { ScaFindingsTab } from "./ScaFindingsTab";
import type { Finding } from "@/types";

interface ReportTabsProps {
  scanId: number;
  findings: Finding[];
}

export function ReportTabs({ scanId, findings }: ReportTabsProps) {
  return (
    <Tabs defaultValue="sca" className="space-y-4">
      <TabsList className="h-11">
        <TabsTrigger value="sca">
          SCA
          <span className="ml-1 rounded-full bg-muted-foreground/15 px-2 py-0.5 text-[10px] font-mono">
            {findings.length}
          </span>
        </TabsTrigger>
        <TabsTrigger
          value="sast"
          disabled
          className="cursor-not-allowed text-muted-foreground/60 data-[disabled]:opacity-60"
        >
          <Lock className="h-3 w-3" />
          SAST
        </TabsTrigger>
        <TabsTrigger
          value="dast"
          disabled
          className="cursor-not-allowed text-muted-foreground/60 data-[disabled]:opacity-60"
        >
          <Lock className="h-3 w-3" />
          DAST
        </TabsTrigger>
      </TabsList>

      <TabsContent value="sca">
        <ScaFindingsTab scanId={scanId} findings={findings} />
      </TabsContent>
    </Tabs>
  );
}

import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import CountAnimation from "@/components/ui/custom/count-animation";
import { Badge } from "@/components/ui/badge";

export function FailedConversionsCard({ total }: { total: number }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Failed deliveries</CardTitle>
        <CardAction>
          <Badge variant="outline">{total ? "Check activity" : "Looking good"}</Badge>
        </CardAction>
      </CardHeader>
      <CardContent>
        <div className="font-display text-3xl">
          <CountAnimation number={total} />
        </div>
        <div className="text-muted-foreground text-sm">Details in your activity log</div>
      </CardContent>
    </Card>
  );
}

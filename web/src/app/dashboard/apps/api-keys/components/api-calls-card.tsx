import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import CountAnimation from "@/components/ui/custom/count-animation";
import { Badge } from "@/components/ui/badge";

export function ApiCallsCard({ total }: { total: number }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Total deliveries</CardTitle>
        <CardAction>
          <Badge variant="outline">All time</Badge>
        </CardAction>
      </CardHeader>
      <CardContent>
        <div className="font-display text-3xl">
          <CountAnimation number={total} />
        </div>
        <div className="text-muted-foreground text-sm">Every desktop delivery attempt</div>
      </CardContent>
    </Card>
  );
}

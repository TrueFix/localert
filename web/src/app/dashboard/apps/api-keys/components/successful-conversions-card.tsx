import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import CountAnimation from "@/components/ui/custom/count-animation";
import { Badge } from "@/components/ui/badge";

export function SuccessfulConversionsCard({ total }: { total: number }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Notifications sent</CardTitle>
        <CardAction>
          <Badge variant="outline" className="bg-status-muted text-status border-status/15">Delivered</Badge>
        </CardAction>
      </CardHeader>
      <CardContent>
        <div className="font-display text-3xl">
          <CountAnimation number={total} />
        </div>
        <div className="text-muted-foreground text-sm">Accepted by your desktop</div>
      </CardContent>
    </Card>
  );
}

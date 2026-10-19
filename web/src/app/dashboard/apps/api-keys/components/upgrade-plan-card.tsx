import { KeyRoundIcon } from "lucide-react"
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import CountAnimation from "@/components/ui/custom/count-animation"

export function ConnectedAppsCard({ total }: { total: number }) {
  return <Card><CardHeader><CardTitle>Connected apps</CardTitle><CardAction><KeyRoundIcon className="text-muted-foreground size-4" /></CardAction></CardHeader><CardContent><div className="font-display text-3xl"><CountAnimation number={total} /></div><div className="text-muted-foreground text-sm">Individual keys, individual control</div></CardContent></Card>
}

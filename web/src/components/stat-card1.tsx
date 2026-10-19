import { BellRingIcon, CheckCheckIcon, HistoryIcon, PlugZapIcon, Settings2Icon, MoreHorizontal } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { useLocalert } from "@/lib/context"

export function Stats() {
  const { data } = useLocalert()
  const stats = [
    { title: "Connected apps", value: data.keys.length, note: "Your tiny team of messengers", tone: "lavender", icon: PlugZapIcon, url: "#keys", label: "Connected" },
    { title: "Happy deliveries", value: data.stats.sent, note: "A little nudge, right on time", tone: "mint", icon: CheckCheckIcon, url: "#activity", label: "All time" },
    { title: "Saved moments", value: data.history_total, note: "Tucked away in your activity", tone: "peach", icon: HistoryIcon, url: "#activity", label: "Retained" },
    { title: "Needs a peek", value: data.stats.failed, note: data.stats.failed ? "Delivery details are in activity" : "No delivery hiccups so far", tone: "rose", icon: BellRingIcon, url: "#activity", label: "All time" },
  ]
  return <div className="metrics bg-muted flex w-full items-center justify-center px-4 py-10 lg:py-20"><div className="mx-auto grid max-w-7xl grow grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">{stats.map(stat => <Card key={stat.title} data-tone={stat.tone} className="shadow-none"><CardHeader><CardTitle className="text-muted-foreground text-sm font-medium">{stat.title}</CardTitle><CardAction><DropdownMenu><DropdownMenuTrigger asChild><Button variant="ghost" size="sm" aria-label={`${stat.title} options`}><MoreHorizontal /></Button></DropdownMenuTrigger><DropdownMenuContent align="end" side="bottom"><DropdownMenuItem asChild><a href={stat.url}><stat.icon />View details</a></DropdownMenuItem><DropdownMenuItem asChild><a href="#settings"><Settings2Icon />Settings</a></DropdownMenuItem></DropdownMenuContent></DropdownMenu></CardAction></CardHeader><CardContent className="flex items-center gap-2.5 space-y-2.5"><span className="text-foreground text-2xl font-bold tracking-tight">{stat.value.toLocaleString()}</span><Badge variant="outline">{stat.label}</Badge></CardContent><CardFooter className="text-muted-foreground border-t text-xs">{stat.note}</CardFooter></Card>)}</div></div>
}

import { useEffect, useState } from "react"
import { BellIcon, CheckIcon, ChevronLeftIcon, ChevronRightIcon, HistoryIcon, LoaderCircleIcon, SearchIcon, Trash2Icon, XIcon } from "lucide-react"
import { toast } from "sonner"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { api, errorMessage, formatDate, retentionLabel, type Delivery } from "@/lib/api"
import { useLocalert } from "@/lib/context"
import { LittleBell, PageHeading } from "@/components/design"

function EventRow({ event }: { event: Delivery }) {
  const sent = event.status === "sent"
  return <div className="event-row"><span className={`event-icon ${sent ? "bg-mint text-mint-foreground" : "bg-rose text-rose-foreground"}`}>{sent ? <CheckIcon /> : <XIcon />}</span><div className="min-w-0 flex-1 space-y-1"><div className="flex flex-wrap items-center justify-between gap-1"><p className="truncate text-sm font-bold">{event.title}</p><span className="whitespace-nowrap text-[0.875rem] text-muted-foreground">{formatDate(event.created_at)}</span></div><p className="line-clamp-2 whitespace-pre-wrap break-words text-xs leading-relaxed text-muted-foreground">{event.message}</p><span className="inline-flex items-center gap-1.5 text-[0.875rem] text-muted-foreground"><BellIcon className="size-3.5" />{event.app_name}</span>{event.error && <p className="break-words text-xs leading-relaxed text-destructive">{event.error}</p>}</div></div>
}

export function RecentActivity() {
  const { data } = useLocalert()
  return <Card className="gap-0 py-0"><CardHeader className="border-b py-5"><CardTitle className="flex items-center justify-between"><span className="flex items-center gap-2"><span className="section-icon bg-mint text-mint-foreground"><HistoryIcon /></span>Fresh little pings</span><a href="#activity" className="subtle-link">View all →</a></CardTitle><CardDescription>The latest from your desktop.</CardDescription></CardHeader><CardContent className="p-0">{data.events.length ? data.events.slice(0, 3).map(event => <EventRow key={event.id} event={event} />) : <div className="empty-nook"><LittleBell className="size-20" /><p>A little quiet. A lot of possibility.</p><small>Your first notification will feel right at home here.</small><Button variant="link" size="sm" asChild><a href="#playground">Send yourself a little hello →</a></Button></div>}</CardContent></Card>
}

type HistoryPage = { events: Delivery[]; total: number; limit: number; offset: number }

export function ActivityPage() {
  const { data, refresh } = useLocalert()
  const [input, setInput] = useState("")
  const [search, setSearch] = useState("")
  const [filter, setFilter] = useState("")
  const [offset, setOffset] = useState(0)
  const [page, setPage] = useState<HistoryPage>({ events: [], total: 0, limit: 25, offset: 0 })
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState("")
  const [confirm, setConfirm] = useState(false)
  const [busy, setBusy] = useState(false)
  useEffect(() => { const timeout = setTimeout(() => { setSearch(input); setOffset(0) }, 250); return () => clearTimeout(timeout) }, [input])
  useEffect(() => {
    const controller = new AbortController()
    setLoading(true); setError("")
    void api<HistoryPage>(`/api/admin/history?${new URLSearchParams({ limit: "25", offset: String(offset), status: filter, search })}`, { signal: controller.signal }).then(value => { setPage(value); if (offset > 0 && offset >= value.total) setOffset(0) }).catch(error => { if (!controller.signal.aborted) setError(errorMessage(error)) }).finally(() => { if (!controller.signal.aborted) setLoading(false) })
    return () => controller.abort()
  }, [search, filter, offset, data.history_total, data.stats.sent, data.stats.failed])
  return <div className="page-stack"><PageHeading eyebrow="A LITTLE WALK DOWN MEMORY LANE" title="Every ping has a story." description={retentionLabel(data.settings.retention)+". Your preferences, your pace."} action={<Button variant="outline" disabled={!data.history_total} onClick={() => setConfirm(true)}><Trash2Icon />Clear history</Button>} /><Card className="gap-0 py-0"><CardHeader className="flex flex-wrap items-center justify-between gap-3 border-b py-4"><div className="flex gap-1">{[["", "All pings"], ["sent", "Delivered"], ["failed", "Hiccups"]].map(([value, label]) => <Button key={value} size="sm" variant={filter === value ? "secondary" : "ghost"} onClick={() => { setFilter(value); setOffset(0) }}>{label}</Button>)}</div><div className="relative w-full sm:w-60"><SearchIcon className="absolute top-2.5 left-3 size-4 text-muted-foreground" /><Input aria-label="Search notifications" placeholder="Find a little moment…" className="pl-9" value={input} onChange={e => setInput(e.target.value)} /></div></CardHeader><CardContent className="p-0">{error ? <p className="p-5 text-sm text-destructive" role="alert">{error}</p> : page.events.length ? page.events.map(event => <EventRow key={event.id} event={event} />) : <div className="empty-nook">{loading ? <LoaderCircleIcon className="animate-spin text-primary" /> : <LittleBell className="size-24" />}<p>{data.history_total ? "Nothing here matches just yet." : "Your story starts with a little ping."}</p><small>{data.history_total ? "Try another word or filter." : "Send a notification and we’ll keep its little story here."}</small></div>}</CardContent><div className="flex items-center justify-between gap-3 border-t px-5 py-3"><p className="text-xs text-muted-foreground">{page.total ? offset+1 : 0}–{Math.min(offset+25, page.total)} of {page.total.toLocaleString()} notifications</p><div className="flex gap-1"><Button variant="outline" size="icon-sm" disabled={offset === 0 || loading} aria-label="Previous notifications" onClick={() => setOffset(Math.max(0, offset-25))}><ChevronLeftIcon /></Button><Button variant="outline" size="icon-sm" disabled={offset+25 >= page.total || loading} aria-label="Next notifications" onClick={() => setOffset(offset+25)}><ChevronRightIcon /></Button></div></div></Card><p className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground"><Badge variant="outline">{data.history_total.toLocaleString()} saved</Badge>All-time totals stay safe, even when history is tidied away.<a href="#settings" className="text-primary">Change retention →</a></p><Dialog open={confirm} onOpenChange={value => { if (!busy) setConfirm(value) }}><DialogContent><DialogHeader><DialogTitle>A fresh page for your history?</DialogTitle><DialogDescription>This clears the saved notifications. Your app keys and all-time counters will stay right where they are.</DialogDescription></DialogHeader><DialogFooter><Button variant="outline" disabled={busy} onClick={() => setConfirm(false)}>Keep my moments</Button><Button variant="destructive" disabled={busy} onClick={async () => { setBusy(true); try { await api("/api/admin/history", { method: "DELETE" }); setConfirm(false); setOffset(0); await refresh(); toast.success("A fresh little start for your history") } catch (error) { toast.error(errorMessage(error)) } finally { setBusy(false) } }}>{busy ? "Tidying…" : "Clear history"}</Button></DialogFooter></DialogContent></Dialog></div>
}

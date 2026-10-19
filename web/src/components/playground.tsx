import { useState } from "react"
import { BellIcon, CheckCircle2Icon, CircleAlertIcon, InfoIcon, LoaderCircleIcon, SendIcon, TriangleAlertIcon } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { CodeBlock } from "@/components/code-block"
import { api, ApiError, errorMessage } from "@/lib/api"
import { useLocalert } from "@/lib/context"
import { PageHeading } from "@/components/design"

const iconChoices = { bell: BellIcon, info: InfoIcon, success: CheckCircle2Icon, warning: TriangleAlertIcon, error: CircleAlertIcon }

export function Playground() {
  const [title, setTitle] = useState("Hello from Localert 👋")
  const [message, setMessage] = useState("Your apps and your desktop are now connected.")
  const [icon, setIcon] = useState("bell")
  const [customIcon, setCustomIcon] = useState("")
  const [appKey, setAppKey] = useState("")
  const [busy, setBusy] = useState(false)
  const [result, setResult] = useState<{ ok: boolean; text: string; detail?: string } | null>(null)
  const { refresh } = useLocalert()
  const Icon = iconChoices[icon as keyof typeof iconChoices] || BellIcon
  const payload = { icon: icon === "custom" ? customIcon : icon, title, message }

  async function send(event: React.FormEvent) {
    event.preventDefault(); setBusy(true); setResult(null)
    try {
      const response = await api<{ id: string; status: string; warning?: string }>(appKey.trim() ? "/api/notify" : "/api/admin/test", {
        method: "POST", headers: appKey.trim() ? { Authorization: `Bearer ${appKey.trim()}` } : {}, body: JSON.stringify(payload),
      })
      setResult({ ok: true, text: "Notification accepted by your desktop.", detail: response.warning || response.id })
      toast.success("Notification sent to your desktop")
    } catch (error) {
      setResult({ ok: false, text: errorMessage(error), detail: error instanceof ApiError ? error.detail : undefined })
    } finally { setBusy(false); await refresh() }
  }

  return <div className="page-stack">
    <PageHeading eyebrow="MAKE A LITTLE NOISE" title="Say hello to your desktop." description="A title, a message, and a little personality. Go on, give it a ping." />
    <div className="grid min-w-0 grid-cols-1 gap-4 lg:grid-cols-2 lg:gap-6 [&>*]:min-w-0">
      <Card><CardHeader><CardTitle>Compose a notification</CardTitle><CardDescription>A title, a message, and a little personality.</CardDescription></CardHeader><CardContent><form onSubmit={send} className="space-y-6">
        <div className="space-y-2"><Label htmlFor="notification-title">Title</Label><Input id="notification-title" value={title} onChange={e => setTitle(e.target.value)} maxLength={128} required /></div>
        <div className="space-y-2"><Label htmlFor="notification-message">Message</Label><Textarea id="notification-message" className="min-h-28 resize-y" value={message} onChange={e => setMessage(e.target.value)} maxLength={4096} required /><p className="text-muted-foreground text-right text-xs">{message.length.toLocaleString()} / 4,096</p></div>
        <div className="space-y-2"><Label htmlFor="notification-icon">Icon</Label><Select value={icon} onValueChange={setIcon}><SelectTrigger className="w-full" id="notification-icon"><SelectValue /></SelectTrigger><SelectContent>{Object.keys(iconChoices).map(value => <SelectItem key={value} value={value}><span className="capitalize">{value}</span></SelectItem>)}<SelectItem value="custom">Custom image</SelectItem></SelectContent></Select></div>
        {icon === "custom" && <div className="space-y-2"><Label htmlFor="custom-icon">Image URL or data URL</Label><Input id="custom-icon" value={customIcon} onChange={e => setCustomIcon(e.target.value)} placeholder="https://example.com/icon.png" required /><p className="text-muted-foreground text-xs">PNG or JPEG, up to 1 MiB and 1024 × 1024 pixels.</p></div>}
        <div className="space-y-2"><Label htmlFor="test-app-key">App key <span className="text-muted-foreground font-normal">(optional)</span></Label><Input id="test-app-key" type="password" value={appKey} onChange={e => setAppKey(e.target.value)} placeholder="la_…" autoComplete="off" /><p className="text-muted-foreground text-xs">Leave blank to send as “Dashboard test”. Add a key to test an app’s authentication.</p></div>
        <Button type="submit" disabled={busy || !title.trim() || !message.trim()} className="w-full">{busy ? <LoaderCircleIcon className="animate-spin" /> : <SendIcon />}{busy ? "Sending…" : "Send to my desktop"}</Button>
        {result && <div role="status" className={`rounded-lg border p-3 text-sm ${result.ok ? "border-status/20 bg-status-muted text-status" : "border-destructive/20 bg-destructive/5 text-destructive"}`}><p>{result.text}</p>{result.detail && <p className="mt-2 break-words font-mono text-xs opacity-80">{result.detail}</p>}</div>}
      </form></CardContent></Card>
      <div className="space-y-4 lg:space-y-6"><Card><CardHeader><CardTitle>Live preview</CardTitle><CardDescription>The final appearance follows your operating system.</CardDescription></CardHeader><CardContent><div className="bg-muted/50 flex min-h-48 items-center justify-center rounded-xl border border-dashed p-5"><div className="bg-card flex w-full max-w-sm items-start gap-3 rounded-xl border p-4 shadow-lg shadow-foreground/5"><span className="bg-muted flex size-10 shrink-0 items-center justify-center rounded-xl"><Icon className="size-5" /></span><div className="min-w-0"><div className="text-muted-foreground mb-1 flex justify-between gap-4 text-[0.875rem]"><span>LOCALERT</span><span>now</span></div><p className="break-words text-sm font-semibold">{title || "Notification title"}</p><p className="text-muted-foreground mt-1 whitespace-pre-wrap break-words text-xs leading-relaxed">{message || "Your message appears here."}</p></div></div></div></CardContent></Card><Card><CardHeader><CardTitle>Request body</CardTitle><CardDescription>This is the JSON your software sends.</CardDescription></CardHeader><CardContent><CodeBlock label="application/json" code={JSON.stringify(payload, null, 2)} /></CardContent></Card></div>
    </div>
  </div>
}

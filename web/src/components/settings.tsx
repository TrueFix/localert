import { useEffect, useState } from "react"
import { ArrowDownToLineIcon, CheckIcon, CopyIcon, DownloadCloudIcon, HeartIcon, HistoryIcon, KeyRoundIcon, LoaderCircleIcon, MonitorIcon, RefreshCwIcon, RotateCcwIcon, SaveIcon, ShieldCheckIcon, SlidersHorizontalIcon, SparklesIcon, WifiIcon } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { PageHeading } from "@/components/design"
import { useLocalert } from "@/lib/context"
import { CodeBlock } from "@/components/code-block"
import { api, copy, errorMessage, formatDate, platformNames, retentionLabel, type Retention, type UpdateStatus } from "@/lib/api"

function SectionHeader({ title, description, icon: Icon }: { title: string; description: string; icon: typeof HistoryIcon }) {
  return <div className="space-y-1"><h3 className="flex items-center gap-2 text-base font-semibold"><span className="section-icon bg-lavender text-primary"><Icon /></span>{title}</h3><p className="text-muted-foreground text-sm">{description}</p></div>
}

function RetentionSettings() {
  const { data, refresh } = useLocalert()
  const [value, setValue] = useState<Retention>(data.settings.retention)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState("")
  const [saved, setSaved] = useState(false)
  const choices = [
    { mode: "count", title: "By count", description: "Just the latest little moments.", emoji: "#" },
    { mode: "days", title: "By days", description: "Make room as time goes by.", emoji: "◷" },
    { mode: "both", title: "A bit of both", description: "Whichever limit comes first.", emoji: "✦" },
    { mode: "forever", title: "Keep everything", description: "For the memory keepers.", emoji: "∞" },
  ] as const
  return <form className="flex flex-col gap-6" onSubmit={async event => {
    event.preventDefault(); setBusy(true); setError(""); setSaved(false)
    try { await api("/api/admin/settings/retention", { method: "PUT", body: JSON.stringify(value) }); await refresh(); setSaved(true); toast.success("Your history preferences are saved") }
    catch (error) { setError(errorMessage(error)) }
    finally { setBusy(false) }
  }}>
    <SectionHeader title="A home for your history" description="Choose how much notification history you’d like to keep." icon={HistoryIcon} />
    <div className="grid gap-3 sm:grid-cols-2" role="group" aria-label="History retention mode">{choices.map(choice => <button key={choice.mode} type="button" aria-pressed={value.mode === choice.mode} onClick={() => { setValue({ ...value, mode: choice.mode }); setSaved(false) }} className={`preference-tile ${value.mode === choice.mode ? "selected" : ""}`}><span className="preference-symbol">{choice.emoji}</span><span className="min-w-0 flex-1 text-left"><strong>{choice.title}</strong><small>{choice.description}</small></span>{value.mode === choice.mode && <CheckIcon className="size-4 text-primary" />}</button>)}</div>
    <div className="grid gap-5 sm:grid-cols-2">{(value.mode === "count" || value.mode === "both") && <div className="space-y-2"><Label htmlFor="retention-count">Notifications to keep</Label><Input id="retention-count" type="number" min={1} max={100000} step={1} required value={value.count} onChange={event => { setValue({ ...value, count: Number(event.target.value) }); setSaved(false) }} /><p className="text-muted-foreground text-xs">From 1 to 100,000 notifications.</p></div>}{(value.mode === "days" || value.mode === "both") && <div className="space-y-2"><Label htmlFor="retention-days">Days to keep</Label><Input id="retention-days" type="number" min={1} max={3650} step={1} required value={value.days} onChange={event => { setValue({ ...value, days: Number(event.target.value) }); setSaved(false) }} /><p className="text-muted-foreground text-xs">From 1 to 3,650 days.</p></div>}</div>
    <div className="rounded-2xl border border-status/15 bg-mint p-4 text-sm text-mint-foreground"><p className="flex items-center gap-2 font-bold"><HeartIcon className="size-4" />{retentionLabel(value)}</p><p className="mt-1 text-xs leading-5 opacity-85">Saving applies your limits immediately. Older messages are tidied away automatically, and all-time counters and app keys stay safe.</p></div>
    {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
    <div className="flex flex-wrap items-center gap-3 border-t pt-5"><Button disabled={busy} type="submit">{busy ? <LoaderCircleIcon className="animate-spin" /> : <SaveIcon />}Save retention</Button>{saved && <span className="flex items-center gap-1 text-xs text-status" role="status"><CheckIcon className="size-3.5" />All tucked away.</span>}</div>
  </form>
}

function UpdateSettings() {
  const { data, refresh, watchRestart } = useLocalert()
  const [feed, setFeed] = useState(data.settings.update_feed)
  const [status, setStatus] = useState<UpdateStatus | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState("")
  useEffect(() => { void api<UpdateStatus>("/api/admin/updates").then(setStatus).catch(error => setError(errorMessage(error))) }, [])
  async function check() {
    setBusy(true); setError("")
    try { setStatus(await api<UpdateStatus>("/api/admin/updates/check", { method: "POST" })) }
    catch (error) { setError(errorMessage(error)) }
    finally { setBusy(false) }
  }
  async function restart(upgrade: boolean) {
    setBusy(true); setError("")
    try {
      const health = await api<{ instance_id: string }>("/api/health")
      await api(upgrade ? "/api/admin/updates/install" : "/api/admin/restart", { method: "POST" })
      watchRestart(upgrade ? "upgrade" : "restart", health.instance_id, upgrade ? status?.latest_version : undefined)
    } catch (error) { setError(errorMessage(error)); setBusy(false) }
  }
  return <div className="flex flex-col gap-6">
    <SectionHeader title="Fresh little improvements" description="Check your version, pick up an update, or give Localert a fresh start." icon={DownloadCloudIcon} />
    <div className="version-card"><span className="version-icon"><SparklesIcon /></span><div className="flex-1"><p className="text-xs text-muted-foreground">YOUR LOCALERT</p><strong className="text-2xl tracking-tight">v{data.version}</strong><p className="text-xs text-muted-foreground">{platformNames[data.platform]} · {status?.platform.split("-").at(-1) || "Desktop"}</p></div><Badge variant="outline">Installed</Badge></div>
    {status?.phase === "available" && <div className="rounded-2xl border border-primary/15 bg-lavender p-4"><p className="flex items-center gap-2 font-bold text-primary"><SparklesIcon className="size-4" />A fresh v{status.latest_version} is ready for you.</p>{status.notes && <p className="mt-2 max-h-32 overflow-y-auto whitespace-pre-wrap text-sm leading-6 text-muted-foreground">{status.notes}</p>}{!status.container && <Button className="mt-4" onClick={() => void restart(true)} disabled={busy}><ArrowDownToLineIcon />Upgrade & restart</Button>}</div>}
    {status?.container && <div className="space-y-3"><p className="text-sm text-muted-foreground">Docker manages this installation. Pull the new image to upgrade; your mounted database is preserved.</p><CodeBlock code="docker compose pull && docker compose up -d" /></div>}
    <a href="https://github.com/TrueFix/localert/releases" target="_blank" rel="noreferrer" className="text-sm text-primary underline underline-offset-4">GitHub releases & changelog ↗</a>
    {status?.phase === "up_to_date" && <p className="flex items-center gap-2 rounded-xl bg-mint p-3 text-sm text-mint-foreground"><CheckIcon className="size-4" />All cozy and up to date. Latest: v{status.latest_version}.</p>}
    {(error || status?.error) && <p role="alert" className="rounded-xl bg-destructive/5 p-3 text-sm text-destructive">{error || status?.error}</p>}
    <div className="flex flex-wrap items-center gap-3"><Button variant="outline" onClick={() => void check()} disabled={busy || !data.settings.update_feed || feed !== data.settings.update_feed}>{busy ? <LoaderCircleIcon className="animate-spin" /> : <RefreshCwIcon />}Check for updates</Button>{status?.checked_at && <span className="text-xs text-muted-foreground">Last checked {formatDate(status.checked_at)}</span>}</div>
    <details className="source-details" open={!data.settings.update_feed}><summary>Update source <span>Set it once, stay in the loop.</span></summary><form className="mt-4 space-y-4" onSubmit={async event => {
      event.preventDefault(); setBusy(true); setError("")
      try { await api("/api/admin/settings/updates", { method: "PUT", body: JSON.stringify({ url: feed }) }); await refresh(); setStatus(await api<UpdateStatus>("/api/admin/updates")); toast.success("Update source saved") }
      catch (error) { setError(errorMessage(error)) }
      finally { setBusy(false) }
    }}><div className="space-y-2"><Label htmlFor="update-feed">Release manifest URL</Label><Input id="update-feed" type="url" placeholder="https://github.com/TrueFix/localert/releases/latest/download/update-manifest.json" value={feed} onChange={event => setFeed(event.target.value)} /><p className="text-xs leading-5 text-muted-foreground">Updates come from GitHub releases. SHA-256 checksums are verified before installation. Local loopback feeds remain available for release testing.</p></div><div className="flex flex-wrap gap-2"><Button type="submit" size="sm" disabled={busy}>Save update source</Button><Button type="button" size="sm" variant="outline" onClick={() => setFeed("https://github.com/TrueFix/localert/releases/latest/download/update-manifest.json")}>Use official GitHub feed</Button></div></form></details>
    <div className="space-y-3 border-t pt-5"><div className="flex flex-wrap items-center justify-between gap-3"><div><h4 className="text-sm font-bold">A little fresh start</h4><p className="mt-1 text-xs text-muted-foreground">Restart without changing your installed version.</p></div><Button variant="outline" onClick={() => void restart(false)} disabled={busy}><RotateCcwIcon />Restart Localert</Button></div><p className="text-xs leading-5 text-muted-foreground">Updates and restarts keep your database, app keys, and preferences. The dashboard reconnects automatically; you’ll sign in again once Localert is back.</p></div>
  </div>
}

function NetworkSettings() {
  const { data, watchRestart } = useLocalert()
  const [savedHost, savedPort] = splitAddress(data.settings.listen_address)
  const [scope, setScope] = useState<"local" | "network">(savedHost === "0.0.0.0" ? "network" : "local")
  const [port, setPort] = useState(savedPort)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState("")
  const [moving, setMoving] = useState("")
  const target = `${scope === "network" ? "0.0.0.0" : "127.0.0.1"}:${port}`
  const changed = target !== data.settings.listen_address
  const remote = !["127.0.0.1", "localhost", "[::1]"].includes(window.location.hostname)
  const portNumber = Number(port)
  const validPort = /^\d+$/.test(port) && portNumber >= 1024 && portNumber <= 65535
  const choices = [
    { value: "local", title: "Just this computer", description: "Only apps on this PC can reach Localert (127.0.0.1).", symbol: "⌂" },
    { value: "network", title: "My whole network", description: "Phones, servers, and other PCs can send notifications (0.0.0.0).", symbol: "⌘" },
  ] as const

  async function save(event: React.FormEvent) {
    event.preventDefault()
    if (!validPort) { setError("Choose a port from 1024 to 65535."); return }
    setBusy(true); setError("")
    try {
      const health = await api<{ instance_id: string }>("/api/health")
      await api("/api/admin/settings/network", { method: "PUT", body: JSON.stringify({ listen_address: target }) })
      if (port !== window.location.port || (scope === "local" && remote)) {
        // The dashboard moves to a new address; navigation is allowed across ports.
        const host = scope === "local" && remote ? "127.0.0.1" : window.location.hostname
        const next = `${window.location.protocol}//${host}:${port}/#settings`
        setMoving(next)
        window.setTimeout(() => { window.location.href = next }, 3000)
      } else {
        watchRestart("restart", health.instance_id)
      }
    } catch (error) { setError(errorMessage(error)); setBusy(false) }
  }

  if (moving) return <div className="flex flex-col gap-4"><SectionHeader title="Moving to a new address" description="Localert is restarting. This page will follow it in a moment." icon={WifiIcon} /><div className="flex items-center gap-3 rounded-2xl bg-lavender p-4 text-sm"><LoaderCircleIcon className="size-5 shrink-0 animate-spin text-primary" /><span className="break-all">Opening <strong>{moving}</strong></span></div><p className="text-sm text-muted-foreground">If nothing happens, open that address yourself. If the port is busy, Localert falls back to <code>http://127.0.0.1:8787</code>.</p></div>

  return <form className="flex flex-col gap-6" onSubmit={save}>
    <SectionHeader title="Who can reach Localert?" description="Choose whether only this computer, or every device on your network, can send notifications." icon={WifiIcon} />
    {data.address_locked && <p role="status" className="rounded-xl bg-peach p-4 text-sm text-peach-foreground">Localert was started with <code className="whitespace-nowrap">--addr {data.address}</code>, which overrides this setting. Remove that flag to manage the address here.</p>}
    <div className="grid gap-3 sm:grid-cols-2" role="group" aria-label="Network access">{choices.map(choice => <button key={choice.value} type="button" disabled={data.address_locked} aria-pressed={scope === choice.value} onClick={() => setScope(choice.value)} className={`preference-tile ${scope === choice.value ? "selected" : ""}`}><span className="preference-symbol">{choice.symbol}</span><span className="min-w-0 flex-1 text-left"><strong>{choice.title}</strong><small>{choice.description}</small></span>{scope === choice.value && <CheckIcon className="size-4 shrink-0 text-primary" />}</button>)}</div>
    <div className="max-w-xs space-y-2"><Label htmlFor="listen-port">Port</Label><Input id="listen-port" inputMode="numeric" disabled={data.address_locked} value={port} onChange={event => setPort(event.target.value.trim())} aria-invalid={!validPort} /><p className="text-xs text-muted-foreground">From 1024 to 65535. The default is 8787.</p></div>
    {scope === "network" && <div className="space-y-3 rounded-2xl bg-mint p-4 text-mint-foreground"><p className="text-sm font-bold">Reach Localert from other devices</p>{data.lan_addresses.length ? <ul className="space-y-2">{data.lan_addresses.map(ip => { const url = `http://${ip}:${port}/api/notify`; return <li key={ip} className="flex items-center gap-2"><code className="min-w-0 flex-1 break-all text-sm">{url}</code><Button type="button" size="icon-sm" variant="ghost" aria-label={`Copy ${url}`} onClick={async () => { try { await copy(url); toast.success("Address copied") } catch (error) { toast.error(errorMessage(error)) } }}><CopyIcon /></Button></li> })}</ul> : <p className="text-sm">No network address found. Connect to Wi-Fi or Ethernet first.</p>}</div>}
    {scope === "network" && <p className="rounded-xl bg-peach p-4 text-sm leading-6 text-peach-foreground"><strong>Use trusted networks only.</strong> Other devices can open the sign-in page and call the API. App keys and your password still protect everything, but traffic is plain HTTP. First-time setup is always limited to this computer.</p>}
    {scope === "local" && remote && changed && <p className="rounded-xl bg-peach p-4 text-sm text-peach-foreground">You're connected from another device. After this change, open the dashboard on the Localert computer itself.</p>}
    {data.settings.listen_address !== data.address && !data.address_locked && !changed && <p className="text-sm text-muted-foreground">Saved address <code>{data.settings.listen_address}</code> was unavailable, so Localert is using <code>{data.address}</code>. Try another port.</p>}
    {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
    <div className="flex flex-wrap items-center gap-3 border-t pt-5"><Button type="submit" disabled={busy || data.address_locked || !changed}>{busy ? <LoaderCircleIcon className="animate-spin" /> : <RotateCcwIcon />}Save & restart</Button><span className="text-sm text-muted-foreground">Listening on <code>{data.address}</code></span></div>
  </form>
}

function splitAddress(value: string): [string, string] {
  const index = value.lastIndexOf(":")
  return index < 0 ? ["127.0.0.1", "8787"] : [value.slice(0, index), value.slice(index + 1)]
}

function PasswordSettings() {
  const [current, setCurrent] = useState("")
  const [replacement, setReplacement] = useState("")
  const [confirmation, setConfirmation] = useState("")
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState("")
  return <form className="flex flex-col gap-6" onSubmit={async event => {
    event.preventDefault()
    if (replacement !== confirmation) { setError("The passwords do not match."); return }
    setError(""); setBusy(true)
    try { await api("/api/admin/password", { method: "POST", body: JSON.stringify({ current_password: current, new_password: replacement }) }); setCurrent(""); setReplacement(""); setConfirmation(""); toast.success("Dashboard password changed") }
    catch (error) { setError(errorMessage(error)) }
    finally { setBusy(false) }
  }}><SectionHeader title="Just your little corner" description="A password keeps your dashboard yours. App keys stay separate." icon={ShieldCheckIcon} /><div className="space-y-2"><Label htmlFor="current-password">Current password</Label><Input id="current-password" type="password" autoComplete="current-password" value={current} onChange={e => setCurrent(e.target.value)} required maxLength={128} /></div><div className="grid gap-5 sm:grid-cols-2"><div className="space-y-2"><Label htmlFor="new-password">New password</Label><Input id="new-password" type="password" autoComplete="new-password" value={replacement} onChange={e => setReplacement(e.target.value)} required minLength={8} maxLength={128} /></div><div className="space-y-2"><Label htmlFor="confirm-new-password">Confirm new password</Label><Input id="confirm-new-password" type="password" autoComplete="new-password" value={confirmation} onChange={e => setConfirmation(e.target.value)} required minLength={8} maxLength={128} /></div></div><p className="rounded-xl bg-lavender p-4 text-xs leading-5 text-muted-foreground">Your password is salted and hashed locally. Changing it signs out other dashboard sessions; your app keys keep working.</p>{error && <p className="text-destructive text-sm" role="alert">{error}</p>}<div className="border-t pt-5"><Button type="submit" disabled={busy}>{busy ? <LoaderCircleIcon className="animate-spin" /> : <KeyRoundIcon />}Change password</Button></div></form>
}

function InstanceSettings() {
  const { data } = useLocalert()
  return <div className="flex flex-col gap-6"><SectionHeader title="This is your little Localert" description="One binary. Your machine. No place like local." icon={MonitorIcon} /><dl className="space-y-4">{[["Version", `v${data.version}`], ["Computer", platformNames[data.platform]], ["Listening on", data.address], ["Started", formatDate(data.started_at)], ["History", retentionLabel(data.settings.retention)], ["Database", data.database_path]].map(([label, value]) => <div className="flex justify-between gap-6 border-b pb-3 last:border-0" key={label}><dt className="shrink-0 text-sm text-muted-foreground">{label}</dt><dd className="break-all text-right text-sm font-semibold">{value}</dd></div>)}</dl><div className="rounded-2xl bg-mint p-4"><p className="mb-2 text-xs font-bold text-mint-foreground">YOUR NOTIFICATION ENDPOINT</p><div className="flex items-center gap-2"><code className="min-w-0 flex-1 break-all text-xs">{window.location.origin}/api/notify</code><Button size="icon-sm" variant="ghost" aria-label="Copy endpoint" onClick={async () => { try { await copy(`${window.location.origin}/api/notify`); toast.success("Endpoint copied") } catch (error) { toast.error(errorMessage(error)) } }}><CopyIcon /></Button></div></div><p className="text-xs leading-5 text-muted-foreground">The dashboard, fonts, icons, and SQLite engine are inside the executable. Installed autostart runs in your desktop session, ready for your apps even when this page is closed.</p></div>
}

export function Settings() {
  const [tab, setTab] = useState("preferences")
  const sections = [{ id: "preferences", label: "Preferences", icon: SlidersHorizontalIcon }, { id: "network", label: "Network", icon: WifiIcon }, { id: "updates", label: "Software updates", icon: DownloadCloudIcon }, { id: "security", label: "Security", icon: ShieldCheckIcon }, { id: "about", label: "About this nook", icon: MonitorIcon }]
  return <div className="page-stack"><PageHeading eyebrow="MAKE YOURSELF AT HOME" title="Just the way you like it." description="A few thoughtful touches to make Localert feel like yours." /><Card className="settings-card"><Tabs value={tab} onValueChange={setTab} orientation="vertical" className="settings-tabs"><TabsList className="settings-tab-list">{sections.map(section => <TabsTrigger value={section.id} key={section.id}><section.icon />{section.label}</TabsTrigger>)}</TabsList><div className="settings-content"><TabsContent value="preferences"><RetentionSettings /></TabsContent><TabsContent value="network"><NetworkSettings /></TabsContent><TabsContent value="updates"><UpdateSettings /></TabsContent><TabsContent value="security"><PasswordSettings /></TabsContent><TabsContent value="about"><InstanceSettings /></TabsContent></div></Tabs></Card></div>
}

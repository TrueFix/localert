import { useCallback, useEffect, useRef, useState } from "react"
import { ArrowRightIcon, BellIcon, CircleAlertIcon, LoaderCircleIcon, LockKeyholeIcon, SendIcon, TerminalIcon } from "lucide-react"
import { toast } from "sonner"
import Logo from "@/components/layout/logo"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ApiKeysDataTable } from "@/app/dashboard/apps/api-keys/components/datatable"
import { ActivityPage } from "@/components/activity"
import { Playground } from "@/components/playground"
import { ApiReference } from "@/components/api-reference"
import { Settings } from "@/components/settings"
import { api, ApiError, errorMessage, type Overview, type SetupStatus, type UpdateStatus } from "@/lib/api"
import { LocalertContext, pages, type Page } from "@/lib/context"
import { DashboardShell } from "@/components/dashboard-shell9"
import { OverviewPage } from "@/components/overview"
import { LittleBell, PageHeading } from "@/components/design"

function Login({ onLogin, status }: { onLogin: () => Promise<void>; status: SetupStatus }) {
  const [password, setPassword] = useState("")
  const [confirmation, setConfirmation] = useState("")
  const [legacyToken, setLegacyToken] = useState("")
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState("")
  const setup = status.setup_required

  async function submit(event: React.FormEvent) {
    event.preventDefault()
    if (setup && password !== confirmation) { setError("The passwords do not match."); return }
    setBusy(true); setError("")
    try {
      await api(setup ? "/api/admin/setup" : "/api/admin/login", { method: "POST", body: JSON.stringify(setup ? { password, legacy_token: legacyToken } : { password }) })
      setPassword(""); setConfirmation(""); setLegacyToken("")
      await onLogin()
    } catch (error) {
      setError(errorMessage(error))
      if (error instanceof ApiError && (error.status === 409 || error.status === 401)) await onLogin()
    } finally { setBusy(false) }
  }

  return <div className="auth-page"><div className="auth-nook w-full max-w-sm space-y-7">
    <a href="/" className="flex items-center justify-center gap-2"><Logo /><span className="text-2xl font-extrabold tracking-tight">Localert<span className="text-primary">.</span></span></a>
    <Card><CardHeader className="text-center"><LittleBell className="mx-auto mb-1 size-24" /><CardTitle className="text-xl">{setup ? status.legacy_token_required ? "Finish upgrading Localert" : "Welcome to Localert" : "Welcome back, lovely human."}</CardTitle><CardDescription>{setup ? "Your very own notification nook. Let’s make it yours." : "Your apps, your alerts, your happy little space."}</CardDescription></CardHeader><CardContent><form className="space-y-5" onSubmit={submit}>
      {setup && !status.local_setup_allowed && <p role="alert" className="text-destructive text-sm">Open http://127.0.0.1:8787 on the computer running Localert to finish first-time setup.</p>}
      {setup && status.legacy_token_required && <div className="space-y-2"><Label htmlFor="legacy-token">Previous admin token</Label><Input id="legacy-token" type="password" value={legacyToken} onChange={event => setLegacyToken(event.target.value)} autoComplete="off" required /><p className="text-muted-foreground text-xs leading-5">Your existing keys and history were imported. Enter the admin_token from your original state.json once to protect this upgrade.</p></div>}
      <div className="space-y-2"><Label htmlFor="dashboard-password">Password</Label><Input id="dashboard-password" type="password" placeholder={setup ? "Choose your dashboard password" : "Your dashboard password"} value={password} onChange={event => setPassword(event.target.value)} autoComplete={setup ? "new-password" : "current-password"} minLength={setup ? 8 : undefined} maxLength={128} required />{setup && <p className="text-muted-foreground text-xs leading-5">Use 8–128 characters. Your password is securely hashed and saved locally in SQLite.</p>}</div>
      {setup && <div className="space-y-2"><Label htmlFor="confirm-password">Confirm password</Label><Input id="confirm-password" type="password" value={confirmation} onChange={event => setConfirmation(event.target.value)} autoComplete="new-password" minLength={8} maxLength={128} required /><p className="text-muted-foreground text-xs leading-5">With autostart installed, Localert starts quietly at sign-in. Your apps can send notifications without opening the dashboard.</p></div>}
      {error && <p className="text-destructive text-sm" role="alert">{error}</p>}
      {!setup && <p className="text-muted-foreground text-xs leading-5">Forgot your password? On this computer, run <code className="text-foreground">localert --reset-password</code>, then refresh this page to choose a new one. App keys and history stay put.</p>}
      <Button className="w-full" type="submit" disabled={busy || !password || (setup && !status.local_setup_allowed)}>{busy ? <LoaderCircleIcon className="animate-spin" /> : null}{busy ? "Connecting…" : setup ? "Finish setup" : "Open my workspace"}{!busy && <ArrowRightIcon />}</Button>
    </form></CardContent></Card>
    <p className="text-muted-foreground flex items-center justify-center gap-2 text-xs"><BellIcon className="size-3.5" />One binary. Your machine. Instant notifications.</p>
  </div></div>
}

function currentPage(): Page {
  const value = window.location.hash.slice(1)
  return Object.hasOwn(pages, value) ? value as Page : "overview"
}

export default function App() {
  const [data, setData] = useState<Overview | null>(null)
  const [authenticated, setAuthenticated] = useState<boolean | null>(null)
  const [setupStatus, setSetupStatus] = useState<SetupStatus | null>(null)
  const [connected, setConnected] = useState(true)
  const [page, setPage] = useState<Page>(currentPage)
  const [operation, setOperation] = useState<{ kind: "restart" | "upgrade"; phase: string; error?: string; progress?: number } | null>(null)
  const refreshVersion = useRef(0)
  const refresh = useCallback(async () => {
    const attempt = ++refreshVersion.current
    try {
      const status = await api<SetupStatus>("/api/admin/setup")
      if (attempt !== refreshVersion.current) return
      setSetupStatus(status)
      if (status.setup_required) { setData(null); setAuthenticated(false); setConnected(true); return }
      const value = await api<Overview>("/api/admin/overview")
      if (attempt !== refreshVersion.current) return
      setData(value); setAuthenticated(true); setConnected(true)
    } catch (error) {
      if (attempt !== refreshVersion.current) return
      if (error instanceof ApiError && error.status === 401) {
        try {
          const status = await api<SetupStatus>("/api/admin/setup")
          if (attempt !== refreshVersion.current) return
          setSetupStatus(status)
        } catch { /* Setup status is best-effort when the session ended. */ }
        setAuthenticated(false); setData(null); setConnected(true)
      } else { setConnected(false) }
    }
  }, [])

  useEffect(() => { void refresh() }, [refresh])
  useEffect(() => {
    if (authenticated !== false || !setupStatus || operation) return
    const interval = setInterval(() => { if (!document.hidden) void refresh() }, 4000)
    return () => clearInterval(interval)
  }, [authenticated, setupStatus, operation, refresh])
  useEffect(() => { if (!authenticated || operation) return; const interval = setInterval(() => { if (!document.hidden) void refresh() }, 5000); return () => clearInterval(interval) }, [authenticated, refresh, operation])
  useEffect(() => { const update = () => { setPage(currentPage()); window.scrollTo(0, 0) }; window.addEventListener("hashchange", update); return () => window.removeEventListener("hashchange", update) }, [])
  useEffect(() => { document.title = `${pages[page]} · Localert` }, [page])
  useEffect(() => { try { const updated = sessionStorage.getItem("localert-updated"); if (updated) { sessionStorage.removeItem("localert-updated"); toast.success(`All fresh! Localert v${updated} is ready.`) } } catch { /* Storage is optional. */ } }, [])

  async function logout() {
    ++refreshVersion.current
    try { await api("/api/admin/logout", { method: "POST" }); setData(null); setAuthenticated(false) }
    catch (error) { toast.error(errorMessage(error)) }
  }

  function watchRestart(kind: "restart" | "upgrade", previous: string, expected?: string) {
    ++refreshVersion.current
    setOperation({ kind, phase: kind === "upgrade" ? "downloading" : "restarting" })
    void (async () => {
      const deadline = Date.now() + 180_000
      while (Date.now() < deadline) {
        try {
          const health = await api<{ instance_id: string; version: string }>("/api/health", { signal: AbortSignal.timeout(2500) })
          if (health.instance_id !== previous && (!expected || health.version.replace(/^v/, "") === expected.replace(/^v/, ""))) {
            if (kind === "upgrade") { try { sessionStorage.setItem("localert-updated", health.version) } catch { /* Storage is optional. */ } window.location.reload(); return }
            await refresh(); setOperation(null); toast.success(`Back and feeling fresh. Localert v${health.version} is ready!`); return
          }
          if (kind === "upgrade") {
            const status = await api<UpdateStatus>("/api/admin/updates", { signal: AbortSignal.timeout(2500) })
            if (status.phase === "error") { setOperation({ kind, phase: "error", error: status.error || "The update did not complete." }); return }
            setOperation({ kind, phase: status.phase, progress: status.total ? Math.min(100, Math.round(status.downloaded/status.total*100)) : undefined })
          }
        } catch { /* A short offline interval is expected while Localert restarts. */ }
        await new Promise(resolve => setTimeout(resolve, 600))
      }
      setOperation({ kind, phase: "error", error: "Localert is taking longer than expected. Check the local log, then try reconnecting." })
    })()
  }

  if (operation) return <div className="auth-page"><Card className="w-full max-w-sm text-center"><CardHeader><LittleBell className="mx-auto size-28" /><CardTitle>{operation.error ? "A little hiccup." : operation.phase === "downloading" ? "Fetching something lovely…" : "Be right back!"}</CardTitle><CardDescription>{operation.error || (operation.kind === "upgrade" ? "A fresh version is on its way. Your apps, keys, and preferences are staying right here." : "Giving your notification nook a quick fresh start.")}</CardDescription></CardHeader><CardContent>{operation.error ? <Button onClick={() => { setOperation(null); void refresh() }}>Return to dashboard</Button> : <div className="flex items-center justify-center gap-2 text-sm text-primary"><LoaderCircleIcon className="size-4 animate-spin" /><span>{operation.phase === "downloading" ? `Downloading ${operation.progress === undefined ? "release" : operation.progress+"%"}` : operation.phase === "installing" ? "Installing your update…" : "Restarting and reconnecting…"}</span></div>}</CardContent></Card></div>

  if (authenticated === false && setupStatus) return <Login onLogin={refresh} status={setupStatus} />
  if (!data) return <div className="flex min-h-svh flex-col items-center justify-center gap-4 p-6"><Logo />{connected ? <LoaderCircleIcon className="text-muted-foreground size-5 animate-spin" /> : <><p className="text-muted-foreground text-sm">Can’t connect to Localert. Check that the executable is running.</p><Button variant="outline" onClick={() => void refresh()}>Try again</Button></>}</div>

  return <LocalertContext.Provider value={{ data, refresh, page, connected, logout, watchRestart }}>
    <DashboardShell>
        {!connected && <div className="bg-destructive/5 text-destructive mb-5 flex items-center gap-2 rounded-lg border border-destructive/20 p-3 text-sm" role="status"><CircleAlertIcon className="size-4 shrink-0" />Connection lost. Retrying automatically…<Button className="ml-auto" variant="ghost" size="sm" onClick={() => void refresh()}>Retry now</Button></div>}
        {page === "overview" && <OverviewPage />}
        {page === "keys" && <div className="page-stack"><PageHeading eyebrow="BETTER TOGETHER" title="Your app collection." description="A key for every project. A little peace of mind for you." /><ApiKeysDataTable data={data.keys} /><p className="text-muted-foreground flex items-center gap-2 text-xs"><LockKeyholeIcon className="size-3.5" />App keys can send notifications. Your dashboard password keeps everything else just yours.</p></div>}
        {page === "playground" && <Playground />}{page === "activity" && <ActivityPage />}{page === "docs" && <ApiReference />}{page === "settings" && <Settings />}
    </DashboardShell>
  </LocalertContext.Provider>
}

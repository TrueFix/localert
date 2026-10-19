import { useState } from "react"
import { PlusCircledIcon } from "@radix-ui/react-icons"
import { CheckIcon, CopyIcon, KeyRoundIcon, LoaderCircleIcon } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { api, copy, errorMessage, type AppKey } from "@/lib/api"
import { useLocalert } from "@/lib/context"

export function CreateApiKeyDialog() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [name, setName] = useState("")
  const [created, setCreated] = useState<{ key: AppKey; secret: string } | null>(null)
  const [busy, setBusy] = useState(false)
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState("")
  const { refresh } = useLocalert()

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault()
    setBusy(true); setError("")
    try {
      const result = await api<{ key: AppKey; secret: string }>("/api/admin/keys", { method: "POST", body: JSON.stringify({ name }) })
      setCreated(result)
      toast.success(`App key created for ${result.key.name}`)
      await refresh()
    } catch (error) { setError(errorMessage(error)) }
    finally { setBusy(false) }
  }

  return (
    <Dialog open={dialogOpen} onOpenChange={value => { if (busy) return; setDialogOpen(value); if (!value) { setCreated(null); setName(""); setError(""); setCopied(false) } }}>
      <DialogTrigger asChild><Button><PlusCircledIcon /><span>Create app key</span></Button></DialogTrigger>
      <DialogContent className="min-w-0 max-h-[90dvh] overflow-y-auto sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{created ? "Your app is ready to connect" : "Create an app key"}</DialogTitle>
          <DialogDescription className="break-words">{created ? `Save this key for ${created.key.name}. It is shown only once; only its hash is stored.` : "Give each app its own key so you can track usage and revoke access individually."}</DialogDescription>
        </DialogHeader>
        {created ? <div className="min-w-0 space-y-6">
          <div className="min-w-0 space-y-3 rounded-xl border bg-muted/35 p-4">
            <Label htmlFor="created-app-key">Your app key</Label>
            <textarea id="created-app-key" aria-label="Your app key" readOnly value={created.secret} rows={3} className="block w-full min-w-0 resize-none rounded-lg border bg-background p-3 font-mono text-[1rem] leading-7 break-all" onFocus={event => event.target.select()} />
            <Button variant="outline" className="w-full" onClick={async () => { try { await copy(created.secret); setCopied(true); toast.success("App key copied") } catch (error) { toast.error(errorMessage(error)) } }}>{copied ? <CheckIcon /> : <CopyIcon />}{copied ? "Copied!" : "Copy app key"}</Button>
          </div>
          <p className="text-muted-foreground flex items-start gap-2 text-xs leading-5"><KeyRoundIcon className="mt-0.5 size-4 shrink-0" />Use this key in the Authorization: Bearer header when posting to /api/notify.</p>
          <Button className="w-full" onClick={() => { setDialogOpen(false); setCreated(null); setName("") }}>I’ve saved my key</Button>
        </div> : <form onSubmit={onSubmit} className="space-y-6">
          <div className="space-y-2"><Label htmlFor="app-name">Application name</Label><Input id="app-name" placeholder="e.g. Build pipeline" value={name} onChange={e => setName(e.target.value)} required maxLength={64} autoComplete="off" /><p className="text-muted-foreground text-xs">Something you’ll recognize in your notification history.</p></div>
          {error && <p role="alert" className="text-destructive text-sm">{error}</p>}
          <Button type="submit" disabled={busy || !name.trim()}>{busy && <LoaderCircleIcon className="animate-spin" />}Create app key</Button>
        </form>}
      </DialogContent>
    </Dialog>
  )
}

import { useState } from "react"
import { PlusCircledIcon } from "@radix-ui/react-icons"
import { KeyRoundIcon, LoaderCircleIcon } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { CodeBlock } from "@/components/code-block"
import { api, errorMessage, type AppKey } from "@/lib/api"
import { useLocalert } from "@/lib/context"

export function CreateApiKeyDialog() {
  const [dialogOpen, setDialogOpen] = useState(false)
  const [name, setName] = useState("")
  const [created, setCreated] = useState<{ key: AppKey; secret: string } | null>(null)
  const [busy, setBusy] = useState(false)
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
    <Dialog open={dialogOpen} onOpenChange={value => { if (busy) return; setDialogOpen(value); if (!value) { setCreated(null); setName(""); setError("") } }}>
      <DialogTrigger asChild><Button><PlusCircledIcon /><span>Create app key</span></Button></DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle>{created ? "Your app is ready to connect" : "Create an app key"}</DialogTitle>
          <DialogDescription>{created ? `Save this key for ${created.key.name}. It is shown only once; only its hash is stored.` : "Give each app its own key so you can track usage and revoke access individually."}</DialogDescription>
        </DialogHeader>
        {created ? <div className="space-y-6">
          <CodeBlock code={created.secret} label="APP KEY · COPY AND SAVE" />
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

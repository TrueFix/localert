import { useState } from "react"
import { CheckIcon, CopyIcon } from "lucide-react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { copy, errorMessage } from "@/lib/api"

export function CodeBlock({ code, label = "Terminal" }: { code: string; label?: string }) {
  const [copied, setCopied] = useState(false)
  return (
    <div className="overflow-hidden rounded-lg border bg-muted/35">
      <div className="flex min-h-12 items-center justify-between gap-3 border-b px-4 py-1">
        <span className="text-muted-foreground font-mono text-xs">{label}</span>
        <Button size="icon-sm" variant="ghost" aria-label="Copy code" onClick={async () => {
          try { await copy(code); setCopied(true); setTimeout(() => setCopied(false), 1800) }
          catch (error) { toast.error(errorMessage(error)) }
        }}>{copied ? <CheckIcon className="text-status" /> : <CopyIcon />}</Button>
      </div>
      <pre className="overflow-x-auto p-4 text-[1rem] leading-7"><code>{code}</code></pre>
    </div>
  )
}

import { ArrowRightIcon, KeyRoundIcon, PlugZapIcon, SendIcon, SparklesIcon, TerminalIcon } from "lucide-react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { CreateApiKeyDialog } from "@/app/dashboard/apps/api-keys/components/create-api-key-dialog"
import { CodeBlock } from "@/components/code-block"
import { RecentActivity } from "@/components/activity"
import { Stats } from "@/components/stat-card1"
import { LittleBell } from "@/components/design"
import { useLocalert } from "@/lib/context"
import { curlExample } from "@/lib/api"

export function OverviewPage() {
  const { data } = useLocalert()
  return <div className="page-stack">
    <section className="welcome-panel"><div><p className="eyebrow"><SparklesIcon className="size-3.5" />HOME SWEET DESKTOP</p><h1>Good things deserve<br className="hidden sm:block" /> a little ping.</h1><p>Your apps have news. Let’s make sure you never miss the good stuff.</p><Button asChild><a href="#playground"><SendIcon />Send a little hello<ArrowRightIcon /></a></Button></div><LittleBell /></section>
    <Stats />
    <div className="grid min-w-0 gap-5 lg:grid-cols-2"><Card><CardHeader><CardTitle className="flex items-center justify-between gap-2"><span className="flex items-center gap-2"><span className="section-icon bg-lavender text-primary"><PlugZapIcon /></span>Your app collection</span><a href="#keys" className="subtle-link">View all<ArrowRightIcon /></a></CardTitle><CardDescription>A little connection for every big idea.</CardDescription></CardHeader><CardContent className="space-y-4">{data.keys.length ? <div className="space-y-2">{data.keys.slice(0, 3).map((key, i) => <a key={key.id} href="#keys" data-app-id={key.id} className="app-peek"><span className={`app-letter ${["bg-lavender text-primary", "bg-mint text-mint-foreground", "bg-peach text-peach-foreground"][i%3]}`}>{key.name[0].toUpperCase()}</span><div className="min-w-0 flex-1"><p className="truncate text-sm font-bold">{key.name}</p><p className="text-muted-foreground truncate font-mono text-xs">{key.prefix}</p></div><span className="text-muted-foreground shrink-0 text-xs">{key.sent} sent</span><span className="bg-status size-1.5 shrink-0 rounded-full" /></a>)}</div> : <div className="empty-nook"><span className="bg-mint text-mint-foreground"><KeyRoundIcon /></span><p>Your next connection starts here.</p><small>Create a key for a script, a side project, or your favorite app.</small></div>}<div className="flex flex-wrap items-center justify-between gap-3 border-t pt-4"><p className="text-muted-foreground text-xs">One app, one key.<br />Nice and tidy.</p><CreateApiKeyDialog /></div></CardContent></Card><RecentActivity /></div>
    <Card className="quickstart"><CardHeader><CardTitle className="flex items-center gap-2"><span className="section-icon bg-peach text-peach-foreground"><TerminalIcon /></span>From “done” to ding.</CardTitle><CardDescription>One tiny request. One real desktop notification.</CardDescription></CardHeader><CardContent><CodeBlock code={curlExample()} label="A LITTLE COPY-PASTE MAGIC" /><a href="#docs" className="subtle-link mt-4">Make yourself at home in the API guide<ArrowRightIcon /></a></CardContent></Card>
  </div>
}

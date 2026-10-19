import type { ReactNode } from "react"
import { HeartIcon, SparklesIcon } from "lucide-react"
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { AppHeader } from "./components/app-header"
import { MobileNavDrawer } from "./components/mobile-nav-drawer"
import { useLocalert } from "@/lib/context"

export function DashboardShell({ children }: { children: ReactNode }) {
  const { data, connected } = useLocalert()
  return <div className="dashboard-wrap"><div className="workspace-caption"><span><SparklesIcon className="size-3.5" />A LITTLE LESS CHECKING. A LITTLE MORE LIVING.</span><span className={connected ? "text-status" : "text-destructive"}><span className={`size-1.5 rounded-full ${connected ? "bg-status" : "bg-destructive"}`} />{connected ? "Listening on your desktop" : "Reconnecting…"}</span></div><div className="dashboard-frame"><SidebarProvider><MobileNavDrawer /><SidebarInset className="bg-muted/50"><AppHeader /><div className="flex flex-1 flex-col"><div className="@container/main flex flex-1 flex-col gap-2"><div className="flex flex-1 flex-col gap-4 p-4 pt-0 md:gap-6">{children}</div></div></div><footer className="workspace-footer"><span><HeartIcon className="size-3" />Small app. Thoughtful notifications.</span><span>All yours, all local <span className="mx-2 opacity-40">·</span> v{data.version}</span></footer></SidebarInset></SidebarProvider></div><p className="below-frame"><span className="bg-status size-1.5 rounded-full" />Your data stays on this computer. Always.</p></div>
}

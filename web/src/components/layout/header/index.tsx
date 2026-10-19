import { LogOutIcon, PanelLeftClose, PanelLeftOpen } from "lucide-react"
import { Separator } from "@/components/ui/separator"
import ThemeSwitch from "@/components/layout/header/theme-switch"
import { Button } from "@/components/ui/button"
import { useSidebar } from "@/components/ui/sidebar"
import { pages, useLocalert } from "@/lib/context"

export function SiteHeader() {
  const { toggleSidebar, open } = useSidebar()
  const { page, connected, logout } = useLocalert()
  return (
    <header className="bg-background/40 sticky top-0 z-50 flex h-(--header-height) shrink-0 items-center gap-2 border-b backdrop-blur-md transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height) md:rounded-tl-xl md:rounded-tr-xl">
      <div className="flex w-full items-center gap-1 px-4 lg:gap-2">
        <Button onClick={toggleSidebar} size="icon" variant="ghost" aria-label="Toggle sidebar">{open ? <PanelLeftClose /> : <PanelLeftOpen />}</Button>
        <Separator orientation="vertical" className="mx-2 data-[orientation=vertical]:h-4 data-[orientation=vertical]:self-center" />
        <span className="text-muted-foreground text-sm">Workspace <span className="mx-2 opacity-50">/</span> <span className="text-foreground">{pages[page]}</span></span>
        <div className="ml-auto flex items-center gap-2">
          <span className={`hidden items-center gap-2 rounded-full px-2.5 py-1 text-xs sm:flex ${connected ? "bg-status-muted text-status" : "bg-destructive/10 text-destructive"}`}><span className={`size-1.5 rounded-full ${connected ? "bg-status" : "bg-destructive"}`} />{connected ? "Service running" : "Reconnecting…"}</span>
          <ThemeSwitch />
          <Separator orientation="vertical" className="mx-2 data-[orientation=vertical]:h-4 data-[orientation=vertical]:self-center" />
          <Button size="icon-sm" variant="ghost" aria-label="Sign out" onClick={() => void logout()}><LogOutIcon /></Button>
        </div>
      </div>
    </header>
  )
}

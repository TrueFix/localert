import { LogOutIcon } from "lucide-react"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { Separator } from "@/components/ui/separator"
import { Button } from "@/components/ui/button"
import { NavMainHorizontal } from "./nav-main-horizontal"
import { navigationLinks } from "./app-sidebar"
import { NavUser } from "./nav-user"
import { Logo } from "./logo"
import ThemeSwitch from "@/components/layout/header/theme-switch"
import { useLocalert } from "@/lib/context"

export function AppHeader() {
  const { logout } = useLocalert()
  return <header data-slot="header" className="flex h-16 shrink-0 items-center justify-between gap-2 px-4 [&_[data-logo=description]]:hidden"><div className="flex grow items-center"><Logo /><span className="brand-name ml-2 text-lg font-extrabold tracking-tight">Localert<span className="text-primary">.</span></span><Separator orientation="vertical" className="mx-4 self-center! data-[orientation=vertical]:h-4" /><SidebarTrigger className="-ml-1 md:hidden" aria-label="Toggle sidebar" /><NavMainHorizontal items={navigationLinks} /></div><div className="flex items-center gap-6"><div className="flex items-center gap-2"><ThemeSwitch /><Button size="icon-sm" variant="ghost" aria-label="Sign out" onClick={() => void logout()}><LogOutIcon /></Button></div><NavUser /></div></header>
}

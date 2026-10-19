import { ActivityIcon, BookOpenIcon, KeyRoundIcon, LayoutDashboardIcon, SendIcon, Settings2Icon } from "lucide-react"
import { SidebarGroup, SidebarGroupContent, SidebarGroupLabel, SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from "@/components/ui/sidebar"
import { useLocalert, type Page } from "@/lib/context"

const navItems = [
  { title: "Workspace", items: [
    { title: "Overview", href: "overview", icon: LayoutDashboardIcon },
    { title: "App keys", href: "keys", icon: KeyRoundIcon },
    { title: "Playground", href: "playground", icon: SendIcon },
    { title: "Activity", href: "activity", icon: ActivityIcon },
  ] },
  { title: "Resources", items: [
    { title: "API reference", href: "docs", icon: BookOpenIcon },
    { title: "Settings", href: "settings", icon: Settings2Icon },
  ] },
]

export function NavMain() {
  const { page } = useLocalert()
  const { isMobile, setOpenMobile } = useSidebar()
  return <>{navItems.map(nav => (
    <SidebarGroup key={nav.title}>
      <SidebarGroupLabel>{nav.title}</SidebarGroupLabel>
      <SidebarGroupContent className="flex flex-col gap-2">
        <SidebarMenu>{nav.items.map(item => (
          <SidebarMenuItem key={item.href}>
            <SidebarMenuButton isActive={page === item.href as Page} tooltip={item.title} asChild>
              <a href={`#${item.href}`} onClick={() => { if (isMobile) setOpenMobile(false) }}><item.icon /><span>{item.title}</span></a>
            </SidebarMenuButton>
          </SidebarMenuItem>
        ))}</SidebarMenu>
      </SidebarGroupContent>
    </SidebarGroup>
  ))}</>
}

import type { ComponentProps } from "react"
import { Sidebar, SidebarContent, SidebarHeader, SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar"
import { useIsMobile } from "@/hooks/use-mobile"
import { Logo } from "./logo"
import { NavMainVertical } from "./nav-main-vertical"

export const navigationLinks = [
  { title: "Overview", url: "#overview" },
  { title: "Apps", url: "#keys" },
  { title: "Playground", url: "#playground" },
  { title: "Activity", url: "#activity" },
  { title: "Guide", url: "#docs" },
  { title: "Settings", url: "#settings" },
]

export function AppSidebar(props: ComponentProps<typeof Sidebar>) {
  const isMobile = useIsMobile()
  if (!isMobile) return null
  return <Sidebar variant="inset" {...props}><SidebarHeader><SidebarMenu><SidebarMenuItem><SidebarMenuButton size="lg" asChild><Logo /></SidebarMenuButton></SidebarMenuItem></SidebarMenu></SidebarHeader><SidebarContent><NavMainVertical items={navigationLinks} /></SidebarContent></Sidebar>
}

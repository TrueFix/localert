import { SidebarGroup, SidebarGroupLabel, SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from "@/components/ui/sidebar"
import { useLocalert } from "@/lib/context"

export function NavMainVertical({ items }: { items: { title: string; url: string }[] }) {
  const { page } = useLocalert()
  const { setOpenMobile } = useSidebar()
  return <SidebarGroup><SidebarGroupLabel>Your little workspace</SidebarGroupLabel><SidebarMenu>{items.map(item => <SidebarMenuItem key={item.url}><SidebarMenuButton isActive={item.url === `#${page}`} asChild><a href={item.url} onClick={() => setOpenMobile(false)}><span>{item.title}</span></a></SidebarMenuButton></SidebarMenuItem>)}</SidebarMenu></SidebarGroup>
}

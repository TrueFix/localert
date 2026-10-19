import type { ComponentProps } from "react"
import { ArrowUpRightIcon, MonitorIcon } from "lucide-react"
import { Sidebar, SidebarContent, SidebarFooter, SidebarHeader, SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar"
import { NavMain } from "@/components/layout/sidebar/nav-main"
import { ScrollArea } from "@/components/ui/scroll-area"
import Logo from "@/components/layout/logo"
import { Card, CardContent, CardDescription, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { useLocalert } from "@/lib/context"
import { platformNames } from "@/lib/api"

export function AppSidebar(props: ComponentProps<typeof Sidebar>) {
  const { data } = useLocalert()
  return (
    <Sidebar collapsible="icon" variant="inset" {...props}>
      <SidebarHeader>
        <SidebarMenu><SidebarMenuItem>
          <SidebarMenuButton asChild className="hover:text-foreground h-10 group-data-[collapsible=icon]:px-0!">
            <a href="#overview"><Logo /><span className="text-foreground font-semibold">Localert<span className="text-muted-foreground ml-2 text-[10px] font-normal">LOCAL</span></span></a>
          </SidebarMenuButton>
        </SidebarMenuItem></SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        <ScrollArea className="h-full [&>[data-slot=scroll-area-viewport]]:scroll-fade"><NavMain /></ScrollArea>
      </SidebarContent>
      <SidebarFooter>
        <Card className="overflow-hidden group-data-[collapsible=icon]:hidden">
          <CardContent className="space-y-3 p-3">
            <div className="space-y-1"><CardTitle>Small binary. Big possibilities.</CardTitle><CardDescription>Connect any app to your desktop with a single HTTP request.</CardDescription></div>
            <Button className="w-full" variant="outline" asChild><a href="#docs">Quick start <ArrowUpRightIcon /></a></Button>
          </CardContent>
        </Card>
        <div className="text-muted-foreground flex items-center gap-2 px-2 py-2 text-xs group-data-[collapsible=icon]:hidden"><MonitorIcon className="size-3.5" />{platformNames[data.platform] || data.platform}<span className="ml-auto font-mono">v{data.version}</span></div>
      </SidebarFooter>
    </Sidebar>
  )
}

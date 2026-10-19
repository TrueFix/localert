import { LogOutIcon, Settings2Icon, SmileIcon } from "lucide-react"
import { Button } from "@/components/ui/button"
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { useLocalert } from "@/lib/context"

export function NavUser() {
  const { logout } = useLocalert()
  return <DropdownMenu><DropdownMenuTrigger asChild><Button variant="ghost" className="p-0" size="icon-sm" aria-label="Open user menu"><span className="bg-peach text-peach-foreground flex size-8 items-center justify-center rounded-full"><SmileIcon className="size-4" /></span></Button></DropdownMenuTrigger><DropdownMenuContent className="min-w-56 rounded-lg" align="end" sideOffset={4}><DropdownMenuLabel>Your happy little workspace</DropdownMenuLabel><DropdownMenuSeparator /><DropdownMenuItem asChild><a href="#settings"><Settings2Icon />Settings</a></DropdownMenuItem><DropdownMenuItem onClick={() => void logout()}><LogOutIcon />Sign out</DropdownMenuItem></DropdownMenuContent></DropdownMenu>
}

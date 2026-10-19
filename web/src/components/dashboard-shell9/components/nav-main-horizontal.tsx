import { NavigationMenu, NavigationMenuItem, NavigationMenuLink, NavigationMenuList } from "@/components/ui/navigation-menu"
import { useLocalert } from "@/lib/context"

export function NavMainHorizontal({ items }: { items: { title: string; url: string }[] }) {
  const { page } = useLocalert()
  return <NavigationMenu viewport={false} className="max-md:hidden"><NavigationMenuList className="gap-2">{items.map(link => <NavigationMenuItem key={link.url}><NavigationMenuLink href={link.url} active={link.url === `#${page}`} aria-current={link.url === `#${page}` ? "page" : undefined} className="text-muted-foreground hover:text-primary py-1.5 font-medium">{link.title}</NavigationMenuLink></NavigationMenuItem>)}</NavigationMenuList></NavigationMenu>
}

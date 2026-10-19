import { useEffect } from "react"
import { createPortal } from "react-dom"
import {
  BookOpenIcon,
  HeartIcon,
  HistoryIcon,
  KeyRoundIcon,
  LayoutDashboardIcon,
  SendIcon,
  Settings2Icon,
  SparklesIcon,
  XIcon,
} from "lucide-react"
import { Button } from "@/components/ui/button"
import { useSidebar } from "@/components/ui/sidebar"
import { LittleBell } from "@/components/design"
import { useIsMobile } from "@/hooks/use-mobile"
import { useLocalert } from "@/lib/context"
import { navigationLinks } from "./app-sidebar"

const linkMeta: Record<string, { icon: typeof LayoutDashboardIcon; hint: string; tone: string }> = {
  "#overview": { icon: LayoutDashboardIcon, hint: "Your cozy home base", tone: "bg-lavender text-primary" },
  "#keys": { icon: KeyRoundIcon, hint: "Keys for every app", tone: "bg-mint text-mint-foreground" },
  "#playground": { icon: SendIcon, hint: "Try a little ping", tone: "bg-peach text-peach-foreground" },
  "#activity": { icon: HistoryIcon, hint: "Every ping remembered", tone: "bg-rose text-rose-foreground" },
  "#docs": { icon: BookOpenIcon, hint: "Copy-paste friendly", tone: "bg-lavender text-primary" },
  "#settings": { icon: Settings2Icon, hint: "Make it yours", tone: "bg-mint text-mint-foreground" },
}

export function MobileNavDrawer() {
  const isMobile = useIsMobile()
  const { openMobile, setOpenMobile } = useSidebar()
  const { page } = useLocalert()

  useEffect(() => {
    if (!openMobile) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpenMobile(false)
    }
    document.body.style.overflow = "hidden"
    window.addEventListener("keydown", onKeyDown)
    return () => {
      document.body.style.overflow = ""
      window.removeEventListener("keydown", onKeyDown)
    }
  }, [openMobile, setOpenMobile])

  if (!isMobile || !openMobile) return null

  return createPortal(
    <>
      <button
        type="button"
        className="localert-mobile-nav-backdrop"
        aria-label="Close navigation"
        onClick={() => setOpenMobile(false)}
      />
      <aside className="localert-mobile-nav" role="dialog" aria-modal="true" aria-label="Navigation">
        <div className="localert-mobile-nav-hero">
          <div className="localert-mobile-nav-brand">
            <LittleBell className="localert-mobile-nav-bell" />
            <div className="min-w-0">
              <p className="localert-mobile-nav-eyebrow">
                <SparklesIcon className="size-3.5" />
                YOUR LITTLE NOOK
              </p>
              <p className="truncate text-lg font-extrabold tracking-tight">
                Localert<span className="text-primary">.</span>
              </p>
            </div>
          </div>
          <Button
            type="button"
            size="icon-sm"
            variant="ghost"
            className="localert-mobile-nav-close"
            aria-label="Close navigation"
            onClick={() => setOpenMobile(false)}
          >
            <XIcon />
          </Button>
        </div>
        <p className="localert-mobile-nav-tagline">Hop anywhere—same cozy dashboard, pocket-sized.</p>
        <nav className="localert-mobile-nav-list" aria-label="Main">
          {navigationLinks.map(item => {
            const meta = linkMeta[item.url]
            const Icon = meta.icon
            const active = item.url === `#${page}`
            return (
              <a
                key={item.url}
                href={item.url}
                className={`mobile-nav-link${active ? " selected" : ""}`}
                aria-label={item.title}
                aria-current={active ? "page" : undefined}
                onClick={() => setOpenMobile(false)}
              >
                <span className={`section-icon ${meta.tone}`}>
                  <Icon />
                </span>
                <span className="min-w-0 flex-1">
                  <strong>{item.title}</strong>
                  <small>{meta.hint}</small>
                </span>
              </a>
            )
          })}
        </nav>
        <footer className="localert-mobile-nav-foot">
          <HeartIcon className="text-primary size-3.5" />
          Small app. Thoughtful notifications.
        </footer>
      </aside>
    </>,
    document.body,
  )
}

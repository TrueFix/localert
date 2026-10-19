import type { ReactNode } from "react"
import { cn } from "@/lib/utils"

export function PageHeading({ eyebrow, title, description, action }: { eyebrow: string; title: string; description: string; action?: ReactNode }) {
  return <div className="page-heading"><div><p className="eyebrow">{eyebrow}</p><h1>{title}</h1><p className="page-description">{description}</p></div>{action && <div className="shrink-0">{action}</div>}</div>
}

export function LittleBell({ className }: { className?: string }) {
  return <svg viewBox="0 0 128 128" fill="none" className={cn("little-bell", className)} aria-hidden="true"><circle cx="64" cy="64" r="56" className="fill-lavender" /><path d="M34 76c7-8 8-15 8-29 0-29 44-29 44 0 0 14 1 21 8 29 3 5 0 9-5 9H39c-5 0-8-4-5-9Z" className="fill-card stroke-primary" strokeWidth="3" strokeLinejoin="round" /><path d="M55 94c2 8 16 8 18 0M59 24v-3c0-5 10-5 10 0v3" className="stroke-primary" strokeWidth="3" strokeLinecap="round" /><path d="M57 61v4m14-4v4m-13 8c4 4 8 4 12 0" className="stroke-foreground" strokeWidth="3" strokeLinecap="round" /><ellipse cx="48" cy="70" rx="5" ry="3" className="fill-rose" /><ellipse cx="80" cy="70" rx="5" ry="3" className="fill-rose" /><path d="m102 24 2 6 6 2-6 2-2 6-2-6-6-2 6-2 2-6Z" className="fill-peach-foreground" /><circle cx="23" cy="84" r="5" className="fill-status-muted" /><circle cx="102" cy="96" r="4" className="fill-rose" /></svg>
}

import { createContext, useContext } from "react"
import type { Overview } from "@/lib/api"

export type Page = "overview" | "keys" | "playground" | "activity" | "docs" | "settings"

export const pages: Record<Page, string> = {
  overview: "Overview", keys: "App keys", playground: "Playground", activity: "Activity", docs: "API reference", settings: "Settings",
}

export const LocalertContext = createContext<{
  data: Overview
  refresh: () => Promise<void>
  page: Page
  connected: boolean
  logout: () => Promise<void>
  watchRestart: (kind: "restart" | "upgrade", instance: string, version?: string) => void
} | null>(null)

export function useLocalert() {
  const value = useContext(LocalertContext)
  if (!value) throw new Error("Localert provider is missing")
  return value
}

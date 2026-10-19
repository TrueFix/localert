export type AppKey = {
  id: string
  name: string
  prefix: string
  created_at: string
  last_used_at: string | null
  sent: number
}

export type Delivery = {
  id: string
  app_id: string
  app_name: string
  title: string
  message: string
  icon: string
  status: "sent" | "failed"
  error?: string
  created_at: string
}

export type Overview = {
  keys: AppKey[]
  events: Delivery[]
  stats: { sent: number; failed: number }
  version: string
  platform: string
  started_at: string
  address: string
  database_path: string
  settings: Settings
  history_total: number
  address_locked: boolean
  lan_addresses: string[]
}

export type Retention = { mode: "count" | "days" | "both" | "forever"; count: number; days: number }
export type Settings = { retention: Retention; update_feed: string; listen_address: string }
export type UpdateStatus = { current_version: string; latest_version?: string; phase: "idle" | "checking" | "available" | "up_to_date" | "downloading" | "installing" | "restarting" | "error"; notes?: string; checked_at?: string; error?: string; downloaded: number; total: number; platform: string; container: boolean }

export function retentionLabel(value: Retention) {
  if (value.mode === "forever") return "Keep every little moment"
  if (value.mode === "days") return `Keep ${value.days.toLocaleString()} days of history`
  if (value.mode === "both") return `Up to ${value.count.toLocaleString()} events from the last ${value.days} days`
  return `Keep the latest ${value.count.toLocaleString()} notifications`
}

export type SetupStatus = {
  setup_required: boolean
  legacy_token_required: boolean
  password_recovery: boolean
  local_setup_allowed: boolean
}

export class ApiError extends Error {
  status: number
  detail?: string
  constructor(message: string, status: number, detail?: string) {
    super(message)
    this.status = status
    this.detail = detail
  }
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    ...options,
    credentials: "same-origin",
    headers: { ...(options.body ? { "Content-Type": "application/json" } : {}), ...options.headers },
  })
  if (!response.ok) {
    const body = await response.json().catch(() => ({}))
    throw new ApiError(body.error || `Request failed (${response.status})`, response.status, body.detail)
  }
  if (response.status === 204) return undefined as T
  return response.json()
}

export function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : "Something went wrong. Please try again."
}

export async function copy(text: string) {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }
  // Clipboard API is unavailable on plain HTTP LAN addresses.
  const input = document.createElement("textarea")
  input.value = text
  input.style.position = "fixed"
  input.style.opacity = "0"
  document.body.append(input)
  input.select()
  const copied = document.execCommand("copy")
  input.remove()
  if (!copied) throw new Error("Select and copy the text manually in this browser.")
}

export function formatDate(value: string | null) {
  if (!value) return "Not used yet"
  return new Intl.DateTimeFormat(undefined, { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }).format(new Date(value))
}

export const platformNames: Record<string, string> = { linux: "Linux", windows: "Windows", darwin: "macOS" }

export function curlExample(key = "YOUR_APP_KEY", origin = window.location.origin) {
  return [
    `curl -X POST ${origin}/api/notify \\`,
    `  -H "Authorization: Bearer ${key}" \\`,
    '  -H "Content-Type: application/json" \\',
    '  -d \'{"icon":"success","title":"Build complete","message":"Your project is ready to ship."}\'',
  ].join("\n")
}

import { createContext, useContext, useEffect, useState, type ReactNode } from "react"

type Theme = "light" | "dark"
const ThemeContext = createContext<{ theme: Theme; setTheme: (theme: Theme) => void }>({ theme: "light", setTheme: () => {} })

// Apply themes from the bundled script so the dashboard needs no inline scripts or CDN.
export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => {
    try { return localStorage.getItem("localert-theme") === "dark" ? "dark" : "light" }
    catch { return "light" }
  })
  useEffect(() => {
    document.documentElement.classList.toggle("dark", theme === "dark")
    document.documentElement.style.colorScheme = theme
    try { localStorage.setItem("localert-theme", theme) } catch { /* Storage is optional. */ }
  }, [theme])
  return <ThemeContext.Provider value={{ theme, setTheme }}>{children}</ThemeContext.Provider>
}

export function useTheme() { return useContext(ThemeContext) }

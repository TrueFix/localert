import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import { ThemeProvider } from "@/lib/theme"
import { TooltipProvider } from "@/components/ui/tooltip"
import { Toaster } from "@/components/ui/sonner"
import App from "./App"
import "./index.css"

createRoot(document.getElementById("root")!).render(
  <StrictMode><ThemeProvider><TooltipProvider><App /><Toaster position="bottom-right" richColors /></TooltipProvider></ThemeProvider></StrictMode>,
)

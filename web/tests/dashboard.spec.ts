import { test, expect } from "@playwright/test"
import { execFile, spawn, type ChildProcess } from "node:child_process"
import { chmod, copyFile, mkdtemp, readFile, rm } from "node:fs/promises"
import { tmpdir } from "node:os"
import { resolve, join } from "node:path"
import { createServer, type Server } from "node:http"
import { createHash } from "node:crypto"
import type { AddressInfo } from "node:net"
import { promisify } from "node:util"

let child: ChildProcess
let directory: string
let origin = ""
const password = "Localert desktop test password"
const replacementPassword = "Localert replacement password"
let releases: Server
let feed = ""
let executable = ""
let releaseVersion: string | undefined
let releaseBinary: Buffer

test.beforeAll(async () => {
  directory = await mkdtemp(join(tmpdir(), "localert-e2e-"))
  executable = join(directory, process.platform === "win32" ? "localert.exe" : "localert")
  const binary = process.env.LOCALERT_BINARY || resolve("..", "bin", process.platform === "win32" ? "localert.exe" : "localert")
  await copyFile(binary, executable)
  await chmod(executable, 0o755)
  const published = join(directory, process.platform === "win32" ? "release.exe" : "release")
  await promisify(execFile)("go", ["build", "-trimpath", "-ldflags", "-s -w -X main.version=99.0.0", "-o", published, "../cmd/localert"], { env: { ...process.env, CGO_ENABLED: "0" } })
  releaseBinary = await readFile(published)
  await start()
  releases = createServer(async (request, response) => {
    try {
      if (request.url === "/binary") { response.setHeader("Content-Length", releaseBinary.length); response.end(releaseBinary); return }
      const health = await fetch(`${origin}/api/health`).then(r => r.json())
      const platform = process.platform === "win32" ? "windows" : process.platform
      const arch = process.arch === "x64" ? "amd64" : process.arch
      response.setHeader("Content-Type", "application/json")
      response.end(JSON.stringify({ version: releaseVersion || health.version, notes: "A tiny test release.", assets: { [`${platform}-${arch}`]: { url: "binary", sha256: createHash("sha256").update(releaseBinary).digest("hex"), size: releaseBinary.length } } }))
    } catch { response.statusCode = 500; response.end("Release service unavailable") }
  })
  await new Promise<void>(resolve => releases.listen(0, "127.0.0.1", resolve))
  feed = `http://127.0.0.1:${(releases.address() as AddressInfo).port}/update-manifest.json`
})

async function start() {
  child = spawn(executable, ["--addr", "127.0.0.1:0", "--data-dir", directory, "--no-open", "--pid-file", join(directory, "localert.pid")], { cwd: directory, stdio: ["ignore", "pipe", "pipe"] })
  await new Promise<void>((done, reject) => {
    let output = ""
    const timeout = setTimeout(() => reject(new Error(`Localert did not start: ${output}`)), 15_000)
    child.on("error", error => { clearTimeout(timeout); reject(error) })
    child.stderr?.on("data", chunk => { output += chunk.toString() })
    child.stdout?.on("data", chunk => {
      output += chunk.toString()
      const url = output.match(/Dashboard\s+(http:\/\/[^\s]+)/)
      if (url) { origin = url[1]; clearTimeout(timeout); done() }
    })
  })
}

async function stop() {
  const actualPid = Number(await readFile(join(directory, "localert.pid"), "utf8").catch(() => "0"))
  if (actualPid && actualPid !== child?.pid) { try { process.kill(actualPid, "SIGTERM") } catch { /* Already stopped. */ } }
  if (child?.exitCode === null) {
    const exited = new Promise<void>(done => child.once("exit", () => done()))
    child.kill("SIGTERM")
    await exited
  }
}

test.afterAll(async () => {
  await stop()
  if (releases) await new Promise<void>(resolve => releases.close(() => resolve()))
  if (directory) await rm(directory, { recursive: true, force: true, maxRetries: 20, retryDelay: 100 })
})

test("embedded dashboard manages app keys and native delivery on desktop and mobile", async ({ page, request }, info) => {
  const errors: string[] = []
  page.on("pageerror", error => errors.push(error.message))
  await page.goto(origin)
  await expect(page.getByText("Welcome to Localert", { exact: true })).toBeVisible()
  await page.screenshot({ path: info.outputPath("first-run-setup.png"), fullPage: true })
  await page.getByLabel("Password", { exact: true }).fill(password)
  await page.getByLabel("Confirm password", { exact: true }).fill("does not match")
  await page.getByRole("button", { name: "Finish setup" }).click()
  await expect(page.getByRole("alert")).toContainText("do not match")
  await page.getByLabel("Confirm password", { exact: true }).fill(password)
  await page.getByRole("button", { name: "Finish setup" }).click()
  await expect(page.getByRole("heading", { name: "Good things deserve a little ping." })).toBeVisible()
  await expect(page.locator('[data-slot="navigation-menu-link"][aria-current="page"]')).toHaveCount(1)

  await page.getByRole("button", { name: "Create app key", exact: true }).click()
  await page.getByLabel("Application name", { exact: true }).fill("Desktop smoke test")
  await page.getByRole("dialog").getByRole("button", { name: "Create app key", exact: true }).click()
  const secret = await page.getByRole("dialog").getByLabel("Your app key", { exact: true }).inputValue()
  expect(secret).toMatch(/^la_[A-Za-z0-9_-]+$/)
  for (const width of [320, 390, 1440]) {
    await page.setViewportSize({ width, height: 800 })
    const dialog = page.getByRole("dialog")
    await expect(dialog.getByRole("button", { name: "Copy app key", exact: true })).toBeVisible()
    expect(await dialog.evaluate(element => element.scrollWidth <= element.clientWidth)).toBe(true)
    const box = await dialog.boundingBox()
    expect(box!.x).toBeGreaterThanOrEqual(0)
    expect(box!.x + box!.width).toBeLessThanOrEqual(width)
    await page.screenshot({ path: info.outputPath(`created-key-${width}.png`), fullPage: true })
  }
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.getByRole("button", { name: "I’ve saved my key" }).click()
  await expect(page.locator("[data-app-id]").filter({ hasText: "Desktop smoke test" })).toBeVisible()
  await expect(page.locator("body")).not.toContainText(secret)
  await page.reload()
  await expect(page.getByRole("heading", { name: "Good things deserve a little ping." })).toBeVisible()
  await page.screenshot({ path: info.outputPath("desktop-overview.png"), fullPage: true })

  await page.getByRole("link", { name: "Playground", exact: true }).click()
  await page.getByLabel("Title", { exact: true }).fill("Localert is connected")
  await page.getByLabel("Message", { exact: true }).fill("Your desktop notification API is ready.")
  const soundToggle = page.getByRole("switch", { name: "Play notification sound", exact: true })
  await expect(soundToggle).not.toBeChecked()
  await expect(page.locator("pre").filter({ hasText: '"sound": false' })).toBeVisible()
  await soundToggle.check()
  await expect(page.locator("pre").filter({ hasText: '"sound": true' })).toBeVisible()
  await page.getByLabel("App key", { exact: false }).fill(secret)
  const delivery = page.waitForResponse(response => response.url() === `${origin}/api/notify` && response.request().method() === "POST")
  await page.getByRole("button", { name: "Send to my desktop" }).click()
  const response = await delivery
  expect(response.request().postDataJSON().sound).toBe(true)
  expect([200, 502]).toContain(response.status())
  if (process.env.LOCALERT_E2E_NATIVE === "1") expect(response.status(), await response.text()).toBe(200)
  await expect(page.getByRole("status").filter({ hasText: response.status() === 200 ? "Notification accepted" : "could not be delivered" })).toBeVisible()
  await soundToggle.uncheck()
  await page.getByLabel("Title", { exact: true }).fill("Silent dashboard test")
  await page.getByLabel("App key", { exact: false }).fill("")
  await expect(page.locator("pre").filter({ hasText: '"sound": false' })).toBeVisible()
  const silentDelivery = page.waitForResponse(response => response.url() === `${origin}/api/admin/test` && response.request().method() === "POST")
  await page.getByRole("button", { name: "Send to my desktop" }).click()
  const silentResponse = await silentDelivery
  expect(silentResponse.request().postDataJSON().sound).toBe(false)
  expect([200, 502]).toContain(silentResponse.status())
  await page.getByRole("link", { name: "Activity", exact: true }).click()
  await expect(page.getByText("Localert is connected", { exact: true })).toBeVisible()

  // A real process restart must retain credentials, keys, counters, and history.
  await stop()
  await start()
  await page.goto(origin)
  await expect(page.getByLabel("Confirm password", { exact: true })).toHaveCount(0)
  await page.getByLabel("Password", { exact: true }).fill("incorrect password")
  await page.getByRole("button", { name: "Open my workspace" }).click()
  await expect(page.getByRole("alert")).toContainText("incorrect password")
  await page.getByLabel("Password", { exact: true }).fill(password)
  await page.getByRole("button", { name: "Open my workspace" }).click()
  await expect(page.locator("[data-app-id]").filter({ hasText: "Desktop smoke test" })).toBeVisible()
  await expect(page.getByText("Localert is connected", { exact: true })).toBeVisible()

  await page.getByRole("link", { name: "Guide", exact: true }).click()
  await page.getByRole("button", { name: "Python", exact: true }).click()
  await expect(page.locator("pre").filter({ hasText: "requests.post" })).toBeVisible()
  await page.getByRole("button", { name: "Toggle theme", exact: true }).click()
  await expect(page.locator("html")).toHaveClass(/dark/)
  await page.getByRole("button", { name: "Toggle theme", exact: true }).click()
  await page.getByRole("link", { name: "Overview", exact: true }).click()
  await page.setViewportSize({ width: 390, height: 844 })
  const createButton = page.getByRole("button", { name: "Create app key", exact: true })
  const bounds = await createButton.boundingBox()
  expect(bounds).not.toBeNull()
  expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(390)
  await page.screenshot({ path: info.outputPath("mobile-overview.png"), fullPage: true })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.getByRole("button", { name: "Toggle sidebar", exact: true }).click()
  await page.getByRole("link", { name: "Apps", exact: true }).click()
  await expect(page.getByRole("heading", { name: "Your app collection." })).toBeVisible()
  await page.setViewportSize({ width: 1440, height: 1000 })
  await page.getByRole("button", { name: "Actions for Desktop smoke test", exact: true }).click()
  await page.getByRole("menuitem", { name: "Revoke key", exact: true }).click()
  await page.getByRole("dialog").getByRole("button", { name: "Revoke key", exact: true }).click()
  await expect(page.getByRole("row").filter({ hasText: "Desktop smoke test" })).toHaveCount(0)
  const revoked = await request.post(`${origin}/api/notify`, { headers: { Authorization: `Bearer ${secret}` }, data: { title: "Revoked", message: "Must not send" } })
  expect(revoked.status()).toBe(401)
  await page.getByRole("link", { name: "Settings", exact: true }).click()
  await page.getByRole("button", { name: /A bit of both/ }).click()
  await page.getByLabel("Notifications to keep", { exact: true }).fill("25")
  await page.getByLabel("Days to keep", { exact: true }).fill("7")
  await page.getByRole("button", { name: "Save retention", exact: true }).click()
  await expect(page.getByText("All tucked away.", { exact: true })).toBeVisible()
  const overview = await page.request.get(`${origin}/api/admin/overview`).then(response => response.json())
  expect(overview.settings.retention).toEqual({ mode: "both", count: 25, days: 7 })
  await page.screenshot({ path: info.outputPath("settings-retention.png"), fullPage: true })
  await page.getByRole("tab", { name: "Security", exact: true }).click()
  await page.getByLabel("Current password", { exact: true }).fill(password)
  await page.getByLabel("New password", { exact: true }).fill(replacementPassword)
  await page.getByLabel("Confirm new password", { exact: true }).fill(replacementPassword)
  await page.getByRole("button", { name: "Change password", exact: true }).click()
  await expect(page.getByText("Dashboard password changed", { exact: true })).toBeVisible()
  await page.getByRole("tab", { name: "Software updates", exact: true }).click()
  const updateSource = page.locator("details.source-details")
  if (await updateSource.getAttribute("open") === null) await updateSource.locator("summary").click()
  await page.getByLabel("Release manifest URL", { exact: true }).fill(feed)
  await page.getByRole("button", { name: "Save update source", exact: true }).click()
  await expect(page.getByRole("button", { name: "Check for updates", exact: true })).toBeEnabled()
  await page.getByRole("button", { name: "Check for updates", exact: true }).click()
  await expect(page.getByText(/All cozy and up to date/)).toBeVisible()
  await page.screenshot({ path: info.outputPath("settings-updates.png"), fullPage: true })
  await page.setViewportSize({ width: 390, height: 844 })
  await page.screenshot({ path: info.outputPath("mobile-settings.png"), fullPage: true })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
  await page.setViewportSize({ width: 1440, height: 1000 })
  releaseVersion = "99.0.0"
  await page.getByRole("button", { name: "Check for updates", exact: true }).click()
  await expect(page.getByRole("button", { name: "Upgrade & restart", exact: true })).toBeVisible()
  const upgradedPageLoaded = page.waitForEvent("load")
  await page.getByRole("button", { name: "Upgrade & restart", exact: true }).click()
  await upgradedPageLoaded
  await expect(page.getByLabel("Password", { exact: true })).toBeVisible()
  await page.getByLabel("Password", { exact: true }).fill(replacementPassword)
  await page.getByRole("button", { name: "Open my workspace" }).click()
  await expect(page.getByRole("button", { name: "Sign out", exact: true })).toBeVisible()
  const upgraded = await page.request.get(`${origin}/api/admin/overview`).then(response => response.json())
  expect(upgraded.version).toBe("99.0.0")
  expect(upgraded.settings.retention).toEqual({ mode: "both", count: 25, days: 7 })
  await page.getByRole("tab", { name: "Software updates", exact: true }).click()
  await page.getByRole("button", { name: "Restart Localert", exact: true }).click()
  await expect(page.getByLabel("Password", { exact: true })).toBeVisible()
  await page.getByLabel("Password", { exact: true }).fill(replacementPassword)
  await page.getByRole("button", { name: "Open my workspace" }).click()
  await expect(page.getByRole("button", { name: "Sign out", exact: true })).toBeVisible()
  const persisted = await page.request.get(`${origin}/api/admin/overview`).then(response => response.json())
  expect(persisted.settings.retention).toEqual({ mode: "both", count: 25, days: 7 })
  expect(errors).toEqual([])
})

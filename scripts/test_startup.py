"""Exercise first-run browser behavior and password recovery using the real binary.

The browser opener and user directories are isolated; no real browser is opened.
"""

import http.cookiejar
import http.server
import hashlib
import json
import os
import platform
from pathlib import Path
import re
import subprocess
import shutil
import sys
import tempfile
import time
import threading
import unittest
import urllib.error
import urllib.request


BINARY = Path(os.environ.get("LOCALERT_TEST_BINARY", Path(__file__).resolve().parents[1] / "bin/localert")).resolve()


@unittest.skipUnless(sys.platform.startswith("linux") and BINARY.is_file(), "build bin/localert on Linux to check browser startup")
class StartupTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="localert-startup-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.data = self.root / "database with spaces"
        self.calls = self.root / "browser calls"
        tools = self.root / "tools"
        tools.mkdir()
        opener = tools / "xdg-open"
        opener.write_text('#!/bin/sh\nprintf "%s\\n" "$1" >> "$TEST_BROWSER_CALLS"\n')
        opener.chmod(0o755)
        self.env = {**os.environ, "PATH": str(tools) + os.pathsep + os.environ["PATH"], "HOME": str(self.root), "TEST_BROWSER_CALLS": str(self.calls)}
        self.process = None
        self.output = None
        self.origin = ""
        self.executable = BINARY
        self.addCleanup(self.stop)
        self.client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def start(self, *options, address="127.0.0.1:0"):
        self.output = (self.root / "server.log").open("w")
        command = [str(self.executable), "--data-dir", str(self.data), *options]
        if address:
            command[1:1] = ["--addr", address]
        self.process = subprocess.Popen(command, env=self.env, stdout=self.output, stderr=subprocess.STDOUT)
        deadline = time.monotonic() + 15
        while time.monotonic() < deadline:
            text = (self.root / "server.log").read_text()
            match = re.search(r"Dashboard\s+(http://\S+)", text)
            if match:
                self.origin = match[1]
                try:
                    self.request("GET", "/api/health")
                    return
                except OSError:
                    pass
            if self.process.poll() is not None:
                self.fail(f"Localert exited during startup: {text}")
            time.sleep(0.02)
        self.fail("Localert did not start")

    def stop(self):
        if self.process is not None:
            if self.process.poll() is None:
                self.process.terminate()
            self.process.wait(timeout=20)
            self.process = None
        if self.output is not None:
            self.output.close()
            self.output = None

    def request(self, method, path, body=None):
        request = urllib.request.Request(self.origin + path, method=method, data=None if body is None else json.dumps(body).encode(), headers={"Content-Type": "application/json"})
        with self.client.open(request, timeout=10) as response:
            return json.load(response)

    def test_first_start_opens_setup_once_then_restart_is_quiet(self):
        self.start()
        deadline = time.monotonic() + 5
        while not self.calls.exists() and time.monotonic() < deadline:
            time.sleep(0.02)
        self.assertEqual(self.calls.read_text().splitlines(), [self.origin])
        self.request("POST", "/api/admin/setup", {"password": "installed user password"})
        created = self.request("POST", "/api/admin/keys", {"name": "Autostart integration"})
        self.stop()
        self.start()
        self.assertFalse(self.request("GET", "/api/admin/setup")["setup_required"])
        time.sleep(0.1)
        self.assertEqual(len(self.calls.read_text().splitlines()), 1)
        self.request("POST", "/api/admin/login", {"password": "installed user password"})
        self.assertEqual(self.request("GET", "/api/admin/keys")["keys"][0]["id"], created["key"]["id"])
        recovery = subprocess.run([str(BINARY), "--data-dir", str(self.data), "--reset-password"], env=self.env, capture_output=True, text=True, timeout=15)
        self.assertEqual(recovery.returncode, 0, recovery.stderr)
        self.assertTrue(self.request("GET", "/api/admin/setup")["setup_required"])
        self.request("POST", "/api/admin/setup", {"password": "recovered user password"})
        self.assertEqual(self.request("GET", "/api/admin/keys")["keys"][0]["id"], created["key"]["id"])

    def test_no_open_suppresses_first_run_browser(self):
        self.start("--no-open")
        self.assertTrue(self.request("GET", "/api/admin/setup")["setup_required"])
        self.assertFalse(self.calls.exists())

    def test_packaged_installer_detects_the_release_binary(self):
        installer = BINARY.with_name("install.sh")
        if not installer.is_file() or os.getuid() == 0:
            self.skipTest("run make installers as a desktop user to check packaging")
        env = {
            **self.env,
            "XDG_CONFIG_HOME": str(self.root / "config"),
            "XDG_DATA_HOME": str(self.root / "data"),
            "XDG_STATE_HOME": str(self.root / "state"),
            "LOCALERT_INSTALL_DIR": str(self.root / "installed binaries"),
        }
        installed = subprocess.run(["sh", str(installer), "--no-start"], env=env, capture_output=True, text=True, timeout=15)
        self.assertEqual(installed.returncode, 0, installed.stderr)
        executable = Path(env["LOCALERT_INSTALL_DIR"]) / "localert"
        version = subprocess.run([str(executable), "--version"], capture_output=True, text=True, check=True)
        self.assertTrue(version.stdout.startswith("Localert "))
        self.assertTrue((Path(env["XDG_CONFIG_HOME"]) / "autostart/localert.desktop").is_file())
        self.assertFalse(self.calls.exists())

    def test_dashboard_restart_preserves_port_and_user_data(self):
        self.start("--no-open", "--pid-file", str(self.root / "localert.pid"))
        self.request("POST", "/api/admin/setup", {"password": "restart user password"})
        created = self.request("POST", "/api/admin/keys", {"name": "Restart-proof app"})
        previous = self.request("GET", "/api/health")["instance_id"]
        self.request("POST", "/api/admin/restart")
        self.wait_for_restart(previous)
        self.assertEqual((self.root / "localert.pid").read_text().strip(), str(self.process.pid))
        self.request("POST", "/api/admin/login", {"password": "restart user password"})
        self.assertEqual(self.request("GET", "/api/admin/keys")["keys"][0]["id"], created["key"]["id"])

    def test_network_setting_moves_to_all_interfaces_after_restart(self):
        import socket, sqlite3
        def free_port():
            with socket.socket() as sock:
                sock.bind(("127.0.0.1", 0))
                return sock.getsockname()[1]
        first, second = free_port(), free_port()
        subprocess.run([str(BINARY), "--data-dir", str(self.data), "--reset-password"], env=self.env, check=True, capture_output=True, timeout=15)
        with sqlite3.connect(self.data / "localert.db") as db:
            db.execute("UPDATE preferences SET listen_address = ?", (f"127.0.0.1:{first}",))
        self.start("--no-open", address=None)
        self.assertTrue(self.origin.endswith(f":{first}"))
        self.request("POST", "/api/admin/setup", {"password": "network user password"})
        self.assertFalse(self.request("GET", "/api/admin/overview")["address_locked"])
        previous = self.request("GET", "/api/health")["instance_id"]
        self.request("PUT", "/api/admin/settings/network", {"listen_address": f"0.0.0.0:{second}"})
        self.origin = f"http://127.0.0.1:{second}"
        self.wait_for_restart(previous)
        self.request("POST", "/api/admin/login", {"password": "network user password"})
        overview = self.request("GET", "/api/admin/overview")
        self.assertEqual(overview["address"], f"0.0.0.0:{second}")
        self.assertEqual(overview["settings"]["listen_address"], f"0.0.0.0:{second}")

    def wait_for_restart(self, previous, version=None):
        deadline = time.monotonic() + 30
        while time.monotonic() < deadline:
            try:
                health = self.request("GET", "/api/health")
                if health["instance_id"] != previous and (version is None or health["version"] == version):
                    return health
            except OSError:
                pass
            time.sleep(0.05)
        self.fail("Localert did not come back after dashboard restart: " + (self.root / "server.log").read_text())

    def test_one_click_upgrade_downloads_a_real_binary_and_restarts(self):
        project = Path(__file__).resolve().parents[1]
        candidate = self.root / "published-release"
        build = subprocess.run(["go", "build", "-trimpath", "-ldflags", "-s -w -X main.version=99.0.0", "-o", str(candidate), "./cmd/localert"], cwd=project, env={**os.environ, "CGO_ENABLED": "0"}, capture_output=True, text=True, timeout=90)
        self.assertEqual(build.returncode, 0, build.stderr)
        content = candidate.read_bytes()
        checksum = hashlib.sha256(content).hexdigest()

        class Feed(http.server.BaseHTTPRequestHandler):
            def do_GET(handler):
                if handler.path == "/binary":
                    body = content
                else:
                    architecture = "arm64" if platform.machine() in ("aarch64", "arm64") else "amd64"
                    body = json.dumps({"version": "99.0.0", "assets": {f"linux-{architecture}": {"url": "binary", "sha256": checksum, "size": len(content)}}}).encode()
                handler.send_response(200)
                handler.send_header("Content-Length", str(len(body)))
                handler.end_headers()
                handler.wfile.write(body)

            def log_message(self, *args):
                pass

        feed = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Feed)
        thread = threading.Thread(target=feed.serve_forever, daemon=True)
        thread.start()
        self.addCleanup(feed.server_close)
        self.addCleanup(feed.shutdown)
        self.executable = self.root / "installed-localert"
        shutil.copy2(BINARY, self.executable)
        self.start("--no-open")
        self.request("POST", "/api/admin/setup", {"password": "upgrade user password"})
        created = self.request("POST", "/api/admin/keys", {"name": "Upgrade-proof app"})
        self.request("PUT", "/api/admin/settings/retention", {"mode": "both", "count": 500, "days": 14})
        self.request("PUT", "/api/admin/settings/updates", {"url": f"http://127.0.0.1:{feed.server_port}/manifest.json"})
        checked = self.request("POST", "/api/admin/updates/check")
        self.assertEqual(checked["phase"], "available", checked)
        previous = self.request("GET", "/api/health")["instance_id"]
        self.request("POST", "/api/admin/updates/install")
        self.wait_for_restart(previous, "99.0.0")
        self.request("POST", "/api/admin/login", {"password": "upgrade user password"})
        overview = self.request("GET", "/api/admin/overview")
        self.assertEqual(overview["keys"][0]["id"], created["key"]["id"])
        self.assertEqual(overview["settings"]["retention"], {"mode": "both", "count": 500, "days": 14})
        self.assertTrue(self.executable.with_name(self.executable.name + ".previous").is_file())


if __name__ == "__main__":
    unittest.main()

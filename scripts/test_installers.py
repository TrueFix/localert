"""Black-box checks for the Unix installer in isolated user directories.

No real desktop autostart files or services are changed. macOS launchctl is mocked.
Run: python -m unittest discover -s scripts -p 'test_*.py'
"""

import configparser
import os
from pathlib import Path
import plistlib
import shlex
import subprocess
import tempfile
import unittest
import sys


INSTALLER = Path(__file__).with_name("install.sh").resolve()


@unittest.skipUnless(os.name == "posix", "Unix installer checks require a POSIX host")
class InstallerTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="localert-installer-")
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name)
        self.home = self.root / "user's space & 50% Ω"
        self.tools = self.root / "tools"
        self.tools.mkdir()
        self.home.mkdir()
        self.binary = self.root / "downloaded localert"
        self.binary.write_text(
            '#!/bin/sh\nif [ "$1" = --version ]; then printf "Localert test\\n"; '
            'else printf "%s\\n" "$@" > "$TEST_BINARY_ARGS"; fi\n'
        )
        self.env = {
            **os.environ,
            "HOME": str(self.home),
            "XDG_CONFIG_HOME": str(self.home / "config files"),
            "XDG_DATA_HOME": str(self.home / "app data"),
            "XDG_STATE_HOME": str(self.home / "state files"),
            "LOCALERT_INSTALL_DIR": str(self.home / "bin folder"),
            "PATH": str(self.tools) + os.pathsep + os.environ["PATH"],
            "MOCK_OS": "Linux",
            "TEST_BINARY_ARGS": str(self.root / "received args"),
            "TEST_LAUNCHCTL_ARGS": str(self.root / "launchctl args"),
        }
        for name, body in {
            "uname": 'if [ "$1" = -s ]; then printf "%s\\n" "$MOCK_OS"; else printf "x86_64\\n"; fi',
            "id": 'printf "1000\\n"',
            "launchctl": 'printf "%s\\n" "$@" >> "$TEST_LAUNCHCTL_ARGS"',
        }.items():
            path = self.tools / name
            path.write_text("#!/bin/sh\n" + body + "\n")
            path.chmod(0o755)

    def install(self, *options):
        return subprocess.run(
            ["sh", str(INSTALLER), "--binary", str(self.binary), *options],
            env=self.env, text=True, capture_output=True, check=True,
        )

    def test_linux_autostart_launcher_and_reinstall_keep_data(self):
        self.install("--no-start")
        target = Path(self.env["LOCALERT_INSTALL_DIR"]) / "localert"
        desktop = Path(self.env["XDG_CONFIG_HOME"]) / "autostart/localert.desktop"
        launcher = Path(self.env["XDG_DATA_HOME"]) / "localert/launch.sh"
        database = Path(self.env["XDG_CONFIG_HOME"]) / "localert/localert.db"
        self.assertTrue(target.is_file())
        self.assertTrue(os.access(target, os.X_OK))
        config = configparser.ConfigParser(interpolation=None)
        config.read(desktop)
        command = config["Desktop Entry"]["Exec"].replace("\\\\", "\\").replace("%%", "%")
        self.assertEqual(shlex.split(command), ["/bin/sh", str(launcher)])
        self.assertEqual(config["Desktop Entry"]["Terminal"], "false")
        subprocess.run(["sh", str(launcher)], env=self.env, check=True)
        self.assertEqual(Path(self.env["TEST_BINARY_ARGS"]).read_text().splitlines(), ["--data-dir", str(database.parent), "--no-open", "--pid-file", str(Path(self.env["XDG_STATE_HOME"]) / "localert/localert.pid"), "--log-file", str(Path(self.env["XDG_STATE_HOME"]) / "localert/localert.log")])
        database.write_bytes(b"existing database, must not be touched")
        self.install("--no-start")
        self.assertEqual(database.read_bytes(), b"existing database, must not be touched")
        self.install("--uninstall")
        self.assertFalse(target.exists())
        self.assertFalse(desktop.exists())
        self.assertFalse(launcher.exists())
        self.assertTrue(database.exists())

    def test_uninstall_purge_removes_only_localert_files(self):
        self.install("--no-start")
        data = Path(self.env["XDG_CONFIG_HOME"]) / "localert"
        logs = Path(self.env["XDG_STATE_HOME"]) / "localert"
        (data / "localert.db").write_text("db")
        (data / "icons").mkdir()
        (data / "icons" / "a.png").write_text("png")
        (logs / "localert.log").write_text("log")
        keep = data / "my-notes.txt"
        keep.write_text("not ours")
        self.install("--uninstall")
        self.assertTrue((data / "localert.db").exists(), "plain uninstall keeps data")
        self.install("--uninstall", "--purge")
        self.assertFalse((data / "localert.db").exists())
        self.assertFalse((data / "icons").exists())
        self.assertFalse((logs / "localert.log").exists())
        self.assertTrue(keep.exists(), "purge must never delete unrelated files")

    def test_macos_launchagent_and_custom_data_directory(self):
        self.env["MOCK_OS"] = "Darwin"
        data = self.home / "custom data & backups"
        self.install("--no-start", "--data-dir", str(data))
        agent = self.home / "Library/LaunchAgents/dev.localert.agent.plist"
        with agent.open("rb") as stream:
            plist = plistlib.load(stream)
        launcher = self.home / "Library/Application Support/localert/install/launch.sh"
        self.assertEqual(plist["ProgramArguments"], ["/bin/sh", str(launcher)])
        self.assertTrue(plist["RunAtLoad"])
        self.assertTrue(plist["KeepAlive"])
        self.assertEqual(plist["LimitLoadToSessionType"], "Aqua")
        calls = Path(self.env["TEST_LAUNCHCTL_ARGS"]).read_text()
        self.assertNotIn("bootstrap", calls)
        subprocess.run(["sh", str(launcher)], env=self.env, check=True)
        self.assertEqual(Path(self.env["TEST_BINARY_ARGS"]).read_text().splitlines(), ["--data-dir", str(data), "--no-open", "--pid-file", str(self.home / "Library/Logs/Localert/localert.pid"), "--log-file", str(self.home / "Library/Logs/Localert/localert.log")])
        self.install("--uninstall")
        self.assertFalse(agent.exists())
        self.assertTrue(data.exists())

    def test_macos_install_starts_only_its_login_agent(self):
        self.env["MOCK_OS"] = "Darwin"
        self.install()
        calls = Path(self.env["TEST_LAUNCHCTL_ARGS"]).read_text().splitlines()
        self.assertIn("bootstrap", calls)
        self.assertIn("gui/1000", calls)
        self.assertIn(str(self.home / "Library/LaunchAgents/dev.localert.agent.plist"), calls)

    def test_piped_install_downloads_and_verifies_release(self):
        curl = self.tools / "curl"
        curl.write_text(f'#!{sys.executable}\n' + '''import hashlib, os, pathlib, sys
args = sys.argv[1:]
url = next(a for a in args if a.startswith("https://"))
binary = pathlib.Path(os.environ["MOCK_BINARY"]).read_bytes()
if "api.github.com" in url:
    print('  "tag_name": "v0.4.0",')
else:
    output = pathlib.Path(args[args.index("-o") + 1])
    if url.endswith("SHA256SUMS"):
        digest = "0" * 64 if os.environ.get("BAD_CHECKSUM") else hashlib.sha256(binary).hexdigest()
        output.write_text(digest + "  localert-linux-amd64\\n")
    else:
        assert "/download/v0.4.0/localert-linux-amd64" in url
        output.write_bytes(binary)
''')
        curl.chmod(0o755)
        self.env["MOCK_BINARY"] = str(self.binary)
        result = subprocess.run(["bash", "-s", "--", "--no-start"], input=INSTALLER.read_text(), cwd=self.root, env=self.env, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        target = Path(self.env["LOCALERT_INSTALL_DIR"]) / "localert"
        self.assertEqual(target.read_bytes(), self.binary.read_bytes())
        self.env["BAD_CHECKSUM"] = "1"
        result = subprocess.run(["bash", "-s", "--", "--version", "v0.4.0", "--no-start"], input=INSTALLER.read_text(), cwd=self.root, env=self.env, capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Checksum mismatch", result.stderr)
        self.assertEqual(target.read_bytes(), self.binary.read_bytes())


if __name__ == "__main__":
    unittest.main()

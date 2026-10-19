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
        self.assertEqual(Path(self.env["TEST_BINARY_ARGS"]).read_text().splitlines(), ["--data-dir", str(database.parent), "--pid-file", str(Path(self.env["XDG_STATE_HOME"]) / "localert/localert.pid")])
        database.write_bytes(b"existing database, must not be touched")
        self.install("--no-start")
        self.assertEqual(database.read_bytes(), b"existing database, must not be touched")
        self.install("--uninstall")
        self.assertFalse(target.exists())
        self.assertFalse(desktop.exists())
        self.assertFalse(launcher.exists())
        self.assertTrue(database.exists())

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
        self.assertEqual(Path(self.env["TEST_BINARY_ARGS"]).read_text().splitlines(), ["--data-dir", str(data), "--pid-file", str(self.home / "Library/Logs/Localert/localert.pid")])
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


if __name__ == "__main__":
    unittest.main()

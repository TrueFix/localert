import subprocess
import tempfile
import unittest
from pathlib import Path

import release_notes


class ReleaseNotesTests(unittest.TestCase):
    def test_changelog_body_extracts_version_section(self):
        changelog = """# Changelog

## [1.0.0] - 2026-01-01

### Added
- First release

## [0.9.0] - 2025-12-01

### Added
- Beta
"""
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "CHANGELOG.md"
            path.write_text(changelog)
            body = release_notes.changelog_body("1.0.0", path)
        self.assertIn("First release", body)
        self.assertNotIn("Beta", body)

    def test_render_includes_commits_and_compare_link(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            (root / "CHANGELOG.md").write_text(
                "# Changelog\n\n## [0.2.0] - 2026-01-02\n\n### Added\n- Two\n"
            )
            subprocess.run(["git", "init", "-q"], cwd=root, check=True)
            subprocess.run(["git", "config", "user.email", "test@example.com"], cwd=root, check=True)
            subprocess.run(["git", "config", "user.name", "Test"], cwd=root, check=True)
            (root / "README.md").write_text("hi\n")
            subprocess.run(["git", "add", "README.md", "CHANGELOG.md"], cwd=root, check=True)
            subprocess.run(["git", "commit", "-qm", "initial"], cwd=root, check=True)
            subprocess.run(["git", "tag", "v0.1.0"], cwd=root, check=True)
            (root / "README.md").write_text("hello\n")
            subprocess.run(["git", "commit", "-am", "second release"], cwd=root, check=True)
            subprocess.run(["git", "tag", "v0.2.0"], cwd=root, check=True)
            subprocess.run(
                ["git", "remote", "add", "origin", "https://github.com/TrueFix/localert.git"],
                cwd=root,
                check=True,
            )
            notes = release_notes.render("0.2.0", root / "CHANGELOG.md", root)
        self.assertIn("Two", notes)
        self.assertIn("second release", notes)
        self.assertIn("v0.1.0...v0.2.0", notes)


if __name__ == "__main__":
    unittest.main()

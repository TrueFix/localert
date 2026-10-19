"""Extract a version's checked-in changelog for GitHub releases and update feeds."""
import re
import sys
from pathlib import Path

version = sys.argv[1].removeprefix("v")
if not re.fullmatch(r"[0-9A-Za-z.+-]+", version):
    raise SystemExit("Invalid version")
changelog = Path("CHANGELOG.md").read_text()
match = re.search(r"^## \[" + re.escape(version) + r"\][^\n]*\n(.*?)(?=^## |^\[[0-9]|\Z)", changelog, re.MULTILINE | re.DOTALL)
if not match or not match.group(1).strip():
    raise SystemExit(f"Add a CHANGELOG.md entry for {version} before tagging a release")
print(match.group(1).strip())

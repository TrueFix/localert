"""Build GitHub release notes from CHANGELOG.md and commits since the previous tag."""
from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path


def validate_version(version: str) -> str:
    version = version.removeprefix("v")
    if not re.fullmatch(r"[0-9A-Za-z.+-]+", version):
        raise SystemExit("Invalid version")
    return version


def changelog_body(version: str, changelog_path: Path) -> str:
    changelog = changelog_path.read_text()
    match = re.search(
        r"^## \[" + re.escape(version) + r"\][^\n]*\n(.*?)(?=^## |^\[[0-9]|\Z)",
        changelog,
        re.MULTILINE | re.DOTALL,
    )
    if not match or not match.group(1).strip():
        raise SystemExit(f"Add a CHANGELOG.md entry for {version} before tagging a release")
    return match.group(1).strip()


def _git(repo_root: Path, *args: str) -> str:
    return subprocess.check_output(["git", *args], text=True, cwd=repo_root)


def git_previous_tag(repo_root: Path, tag: str) -> str | None:
    tags = _git(repo_root, "tag", "--sort=v:refname", "-l", "v*").split()
    try:
        index = tags.index(tag)
    except ValueError:
        return tags[-1] if tags else None
    return tags[index - 1] if index > 0 else None


def release_revision(repo_root: Path, tag: str) -> str:
    tags = _git(repo_root, "tag", "--sort=v:refname", "-l", "v*").split()
    return tag if tag in tags else "HEAD"


def git_commit_lines(repo_root: Path, previous: str | None, tag: str) -> list[str]:
    end = release_revision(repo_root, tag)
    revision_range = f"{previous}..{end}" if previous else end
    output = _git(repo_root, "log", revision_range, "--pretty=format:%h %s").strip()
    if not output:
        return []
    return [f"- `{line.split(' ', 1)[0]}` {line.split(' ', 1)[1]}" for line in output.splitlines()]


def github_repo(repo_root: Path) -> str:
    repo = os.environ.get("GITHUB_REPOSITORY", "").strip()
    if repo:
        return repo
    remote = _git(repo_root, "remote", "get-url", "origin").strip()
    match = re.search(r"github\.com[:/](?P<repo>[^/]+/[^/.]+)", remote)
    if not match:
        raise SystemExit("Set GITHUB_REPOSITORY or use a github.com origin remote")
    return match.group("repo").removesuffix(".git")


def compare_url(repo: str, previous: str | None, tag: str) -> str:
    base = f"https://github.com/{repo}/compare"
    if previous:
        return f"{base}/{previous}...{tag}"
    return f"https://github.com/{repo}/releases/tag/{tag}"


def render(version: str, changelog_path: Path | None = None, repo_root: Path | None = None) -> str:
    repo_root = repo_root or Path.cwd()
    changelog_path = changelog_path or repo_root / "CHANGELOG.md"
    tag = f"v{version}"
    body = changelog_body(version, changelog_path)
    previous = git_previous_tag(repo_root, tag)
    commits = git_commit_lines(repo_root, previous, tag)
    repo = github_repo(repo_root)
    parts = [body, "", "---", ""]
    if commits:
        parts.extend(["### Commits", "", *commits, ""])
    parts.append(f"**Full Changelog**: {compare_url(repo, previous, tag)}")
    return "\n".join(parts).strip() + "\n"


def main() -> None:
    version = validate_version(sys.argv[1])
    sys.stdout.write(render(version))


if __name__ == "__main__":
    main()

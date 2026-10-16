#!/usr/bin/env python3
"""Reserve a stable release tag for a manual GitHub Actions run."""

import argparse
import os
from pathlib import Path
import re
import subprocess
import sys


STABLE_TAG = re.compile(r"v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)")
SHA = re.compile(r"[0-9a-fA-F]{40}")


def git(*args):
    return subprocess.run(
        ["git", *args], check=True, text=True, capture_output=True
    ).stdout.strip()


def ancestor(older, newer):
    result = subprocess.run(
        ["git", "merge-base", "--is-ancestor", older, newer], check=False
    )
    if result.returncode not in (0, 1):
        raise ValueError("Cannot determine release ancestry")
    return result.returncode == 0


def prepare(bump, commit, run_id):
    if bump not in ("major", "minor", "patch"):
        raise ValueError("Release type must be major, minor, or patch")
    if not SHA.fullmatch(commit):
        raise ValueError("Commit must be a full 40-character commit SHA")
    if not run_id.isdecimal():
        raise ValueError("Missing or invalid GitHub workflow run ID")
    sha = git("rev-parse", "--verify", "--end-of-options", commit + "^{commit}")
    if not ancestor(sha, "origin/master"):
        raise ValueError("The release commit must already be merged into master")

    tags = []
    for tag in git("tag", "--list").splitlines():
        match = STABLE_TAG.fullmatch(tag)
        if match:
            tags.append((tuple(map(int, match.groups())), tag))
    tags.sort()

    # A rerun keeps its version even if the first attempt already pushed the
    # tag. GITHUB_RUN_ID is stable across attempts; no force-push is used.
    marker = "rwr-release-run: " + run_id
    for _, tag in tags:
        if git("cat-file", "-t", "refs/tags/" + tag) != "tag":
            continue
        lines = git("for-each-ref", "--format=%(contents)", "refs/tags/" + tag).splitlines()
        if marker not in lines:
            continue
        if git("rev-parse", tag + "^{commit}") != sha or "rwr-release-type: " + bump not in lines:
            raise ValueError("This workflow run already reserved a different release")
        previous = next((line.removeprefix("rwr-previous-tag: ") for line in lines
                         if line.startswith("rwr-previous-tag: ")), "")
        git("push", "origin", f"refs/tags/{tag}:refs/tags/{tag}")
        return tag, sha, previous

    # A second dispatch for an already versioned commit is usually an accidental
    # double click. Retrying the existing run is the path for failed releases.
    for _, tag in tags:
        if git("rev-parse", tag + "^{commit}") == sha:
            raise ValueError(f"Commit already has release tag {tag}; rerun the original release instead")

    version, previous = tags[-1] if tags else ((0, 0, 0), "")
    if previous and not ancestor(previous + "^{commit}", sha):
        raise ValueError("The selected commit predates or excludes the latest stable release")
    major, minor, patch = version
    if bump == "major":
        major, minor, patch = major + 1, 0, 0
    elif bump == "minor":
        minor, patch = minor + 1, 0
    else:
        patch += 1
    tag = f"v{major}.{minor}.{patch}"
    message = (f"Release {tag}\n\nrwr-release-run: {run_id}\n"
               f"rwr-release-type: {bump}\nrwr-previous-tag: {previous}\n")
    git("-c", "user.name=github-actions[bot]", "-c",
        "user.email=41898282+github-actions[bot]@users.noreply.github.com",
        "tag", "--annotate", tag, sha, "--message", message)
    git("push", "origin", f"refs/tags/{tag}:refs/tags/{tag}")
    return tag, sha, previous


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--bump", required=True, choices=("major", "minor", "patch"))
    parser.add_argument("--commit", default="")
    args = parser.parse_args()
    try:
        if os.environ.get("GITHUB_REF") != "refs/heads/master":
            raise ValueError("Run the release workflow from the master branch")
        tag, sha, previous = prepare(
            args.bump, args.commit or os.environ.get("GITHUB_SHA", ""),
            os.environ.get("GITHUB_RUN_ID", ""),
        )
        if output := os.environ.get("GITHUB_OUTPUT"):
            with Path(output).open("a", encoding="utf-8") as stream:
                stream.write(f"tag={tag}\nsha={sha}\nprevious_tag={previous}\n")
        if summary := os.environ.get("GITHUB_STEP_SUMMARY"):
            with Path(summary).open("a", encoding="utf-8") as stream:
                stream.write(f"## Release {tag}\n\nType: `{args.bump}`\n\n"
                             f"Commit: `{sha}`\n\nPrevious stable tag: `{previous or 'none'}`\n")
        print(f"Reserved {tag} at {sha}")
    except (ValueError, subprocess.CalledProcessError) as exc:
        print(f"Release preparation failed: {exc}", file=sys.stderr)
        if isinstance(exc, subprocess.CalledProcessError) and exc.stderr:
            print(exc.stderr.strip(), file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())

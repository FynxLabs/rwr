"""Exercise release reservation against disposable local Git remotes."""

import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("prepare-release.py")


class ReleaseTests(unittest.TestCase):
    def setUp(self):
        temp = tempfile.TemporaryDirectory()
        self.addCleanup(temp.cleanup)
        self.root = Path(temp.name)
        self.remote = self.root / "remote.git"
        self.work = self.root / "work"
        self.work.mkdir()
        self.git_env = {key: value for key, value in os.environ.items()
                        if not key.startswith("GIT_")}
        self.git("init", "--bare", str(self.remote))
        self.git("init", "--initial-branch=master")
        for key, value in {
            "user.name": "Release fixture", "user.email": "fixture@example.invalid",
            "commit.gpgsign": "false", "tag.gpgsign": "false",
            "core.hooksPath": str(self.root / "no-hooks"),
        }.items():
            self.git("config", key, value)
        self.git("remote", "add", "origin", str(self.remote))
        self.first = self.commit()
        self.git("push", "--set-upstream", "origin", "master")
        self.env = self.git_env | {
            "GITHUB_REF": "refs/heads/master", "GITHUB_SHA": self.first,
            "GITHUB_RUN_ID": "1234", "GITHUB_OUTPUT": str(self.root / "output"),
            "GITHUB_STEP_SUMMARY": str(self.root / "summary"),
        }

    def git(self, *args):
        return subprocess.run(["git", *args], cwd=self.work, env=self.git_env, check=True,
                              text=True, capture_output=True).stdout.strip()

    def commit(self):
        file = self.work / "changes"
        file.write_text(file.read_text() + "change\n" if file.exists() else "change\n")
        self.git("add", "changes")
        self.git("commit", "--message", "Fixture change")
        return self.git("rev-parse", "HEAD")

    def baseline(self):
        # Numeric ordering, prereleases, and the movable nightly tag must not
        # alter which stable version the dropdown increments.
        for tag in ("v2.9.9", "v2.10.3", "v99.0.0-rc.1", "nightly", "v03.0.0"):
            self.git("tag", tag)
        self.git("push", "--tags", "origin")
        sha = self.commit()
        self.git("push", "origin", "master")
        self.env["GITHUB_SHA"] = sha
        return sha

    def release(self, bump="patch", commit="", **env):
        return subprocess.run([sys.executable, str(SCRIPT), "--bump", bump,
                               "--commit", commit], cwd=self.work,
                              env=self.env | env, text=True, capture_output=True)

    def assert_reserved(self, bump, expected):
        sha = self.baseline()
        result = self.release(bump)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.git("--git-dir", str(self.remote), "rev-parse",
                                  expected + "^{commit}"), sha)
        outputs = (self.root / "output").read_text()
        self.assertIn("tag=" + expected + "\n", outputs)
        self.assertIn("previous_tag=v2.10.3\n", outputs)
        self.assertIn("sha=" + sha + "\n", outputs)

    def test_patch(self):
        self.assert_reserved("patch", "v2.10.4")

    def test_minor(self):
        self.assert_reserved("minor", "v2.11.0")

    def test_major(self):
        self.assert_reserved("major", "v3.0.0")

    def test_first_release(self):
        result = self.release()
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.git("rev-parse", "v0.0.1^{commit}"), self.first)

    def test_explicit_merge_commit_survives_later_master_changes(self):
        target = self.baseline()
        newer = self.commit()
        self.git("push", "origin", "master")
        self.env["GITHUB_SHA"] = newer
        result = self.release("minor", target)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.git("rev-parse", "v2.11.0^{commit}"), target)

    def test_rerun_reuses_original_tag_even_after_later_release(self):
        self.baseline()
        first = self.release()
        self.assertEqual(first.returncode, 0, first.stderr)
        self.git("tag", "v2.11.0")
        self.git("push", "--tags", "origin")
        rerun = self.release()
        self.assertEqual(rerun.returncode, 0, rerun.stderr)
        self.assertIn("Reserved v2.10.4", rerun.stdout)
        changed_input = self.release("major")
        self.assertNotEqual(changed_input.returncode, 0)

    def test_duplicate_dispatch_does_not_bump_same_commit(self):
        self.baseline()
        self.assertEqual(self.release().returncode, 0)
        duplicate = self.release(GITHUB_RUN_ID="5678")
        self.assertNotEqual(duplicate.returncode, 0)
        self.assertIn("already has release tag", duplicate.stderr)
        self.assertNotIn("v2.10.5", self.git("tag", "--list"))

    def test_unmerged_commit_is_rejected(self):
        self.git("switch", "--orphan", "unmerged")
        (self.work / "outside").write_text("unmerged content\n")
        self.git("add", "outside")
        outside = self.commit()
        self.git("switch", "master")
        result = self.release(commit=outside)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("must already be merged", result.stderr)
        self.assertEqual(self.git("tag", "--list"), "")

    def test_commit_before_latest_stable_release_is_rejected(self):
        later = self.commit()
        self.git("tag", "v1.0.0", later)
        self.git("push", "origin", "master", "--tags")
        result = self.release(commit=self.first)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("predates or excludes", result.stderr)

    def test_invalid_inputs_and_wrong_workflow_branch_make_no_tag(self):
        for bump, commit, ref in (
            ("patch; touch injected", "", "refs/heads/master"),
            ("patch", "master", "refs/heads/master"),
            ("patch", "", "refs/heads/feature"),
        ):
            with self.subTest(bump=bump, commit=commit, ref=ref):
                result = self.release(bump, commit, GITHUB_REF=ref)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.git("tag", "--list"), "")
        self.assertFalse((self.work / "injected").exists())

    def test_concurrent_tag_collision_never_overwrites_remote(self):
        self.baseline()
        self.git("--git-dir", str(self.remote), "tag", "v2.10.4", self.first)
        result = self.release()
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.git("--git-dir", str(self.remote), "rev-parse",
                                  "v2.10.4^{commit}"), self.first)


if __name__ == "__main__":
    unittest.main()

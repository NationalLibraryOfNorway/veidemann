#!/usr/bin/env python3
"""Run review probes through Go overlays without editing service sources."""

import argparse
import json
import os
from pathlib import Path
import subprocess
import time


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--browser", action="store_true", help="Run real Chromium/Docker probes"
    )
    parser.add_argument("--run", help="Optional Go test name regular expression")
    args = parser.parse_args()
    review = Path(__file__).resolve().parent
    root = review.parents[1]

    temporary = Path(os.environ.get("TMPDIR", "/tmp")) / (
        f"veidemann-review-{os.getpid()}-{time.time_ns()}"
    )
    temporary.mkdir()
    try:
        replacements = {}
        for target, source in [
            ("recorderproxy/context/review_probe_test.go", "context_probes.go.txt"),
            ("recorderproxy/recorderproxy/review_probe_test.go", "harvest_probes.go.txt"),
            ("browser-controller/server/review_probe_test.go", "browser_probes.go.txt"),
        ]:
            replacements[str(root / target)] = str(review / source)

        overlay = temporary / "overlay.json"
        overlay.write_text(json.dumps({"Replace": replacements}))
        command = ["go", "test", "-overlay=" + str(overlay)]
        if args.browser:
            command += ["-tags=integration", "./server"]
            module = root / "browser-controller"
            default_run = "^TestReviewBrowserCases$"
        else:
            command += ["./context", "./recorderproxy"]
            module = root / "recorderproxy"
            default_run = "^TestReview"
        command += ["-run", args.run or default_run, "-count=1", "-timeout=3m", "-v"]
        print("Review probes assert expected behavior; failures reproduce findings.", flush=True)
        return subprocess.run(command, cwd=module, check=False).returncode
    finally:
        (temporary / "overlay.json").unlink(missing_ok=True)
        temporary.rmdir()


if __name__ == "__main__":
    raise SystemExit(main())

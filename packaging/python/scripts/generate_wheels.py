# Copyright 2020-2026 Buf Technologies, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import hashlib
import shutil
import subprocess
import sys
import urllib.request
from importlib.metadata import version as pkg_version
from pathlib import Path

GITHUB_RELEASES_BASE = "https://github.com/bufbuild/buf/releases/download"

# Maps buf platform suffix to Python wheel platform tag.
# buf binary names on GitHub releases follow the pattern:
#   buf-{PLATFORM} or buf-{PLATFORM}.exe
# Check https://go.dev/wiki/MinimumRequirements#operating-systems for
# minimum OS versions, especially macOS.
PLATFORMS = [
    ("Darwin-arm64", "macosx_11_0_arm64"),
    ("Darwin-x86_64", "macosx_11_0_x86_64"),
    ("Linux-aarch64", "manylinux_2_17_aarch64.manylinux2014_aarch64.musllinux_1_1_aarch64"),
    ("Linux-armv7", "manylinux_2_17_armv7l.manylinux2014_armv7l.musllinux_1_1_armv7l"),
    ("Linux-ppc64le", "manylinux_2_17_ppc64le.manylinux2014_ppc64le.musllinux_1_1_ppc64le"),
    ("Linux-riscv64", "manylinux_2_17_riscv64.musllinux_1_2_riscv64"),
    ("Linux-s390x", "manylinux_2_17_s390x.manylinux2014_s390x.musllinux_1_1_s390x"),
    ("Linux-x86_64", "manylinux_2_17_x86_64.manylinux2014_x86_64.musllinux_1_1_x86_64"),
    ("Windows-arm64", "win_arm64"),
    ("Windows-x86_64", "win_amd64"),
]


def fetch_checksums(version: str) -> dict[str, str]:
    url = f"{GITHUB_RELEASES_BASE}/v{version}/sha256.txt"
    with urllib.request.urlopen(url) as response:
        content = response.read().decode()
    result = {}
    for line in content.splitlines():
        if "  " in line:
            sha256, filename = line.split("  ", 1)
            result[filename] = sha256
    return result


def verify_checksum(path: Path, expected: str) -> None:
    actual = hashlib.sha256(path.read_bytes()).hexdigest()
    if actual != expected:
        msg = f"checksum mismatch for {path.name}: expected {expected}, got {actual}"
        raise ValueError(msg)


def download(url: str, dest: Path) -> None:
    with urllib.request.urlopen(url) as response, dest.open("wb") as f:
        shutil.copyfileobj(response, f)


def main() -> None:
    base_dir = Path(__file__).parent.parent

    version = pkg_version("buf-bin")

    print(f"Generating wheels for buf v{version}")
    checksums = fetch_checksums(version)

    bin_dir = base_dir / "out" / "bin"

    for buf_platform, wheel_platform in PLATFORMS:
        print(f"\nBuilding wheel for {buf_platform} ({wheel_platform})")

        shutil.rmtree(bin_dir, ignore_errors=True)
        bin_dir.mkdir(parents=True)

        try:
            ext = ".exe" if buf_platform.startswith("Windows") else ""
            filename = f"buf-{buf_platform}{ext}"
            dest = bin_dir / f"buf{ext}"
            print(f"  Downloading {GITHUB_RELEASES_BASE}/v{version}/{filename}")
            download(f"{GITHUB_RELEASES_BASE}/v{version}/{filename}", dest)
            verify_checksum(dest, checksums[filename])
            if not ext:
                dest.chmod(0o755)

            subprocess.run(
                ["uv", "build", "--wheel"],
                check=True,
                cwd=base_dir,
            )

            dist_dir = base_dir / "dist"
            built_wheel = next(dist_dir.glob("*-py3-none-any.whl"))

            subprocess.run(
                [
                    sys.executable,
                    "-m",
                    "wheel",
                    "tags",
                    "--remove",
                    "--platform-tag",
                    wheel_platform,
                    str(built_wheel),
                ],
                check=True,
            )
        finally:
            shutil.rmtree(bin_dir, ignore_errors=True)

    bin_dir.mkdir(parents=True)
    (bin_dir / ".gitkeep").touch()

    print("\nDone. Wheels written to dist/")


if __name__ == "__main__":
    main()

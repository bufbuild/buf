// Copyright 2020-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import os from "os";
import path from "path";

export const knownWindowsPackages: Record<string, string> = {
  "win32 arm64 LE": "@bufbuild/buf-win32-arm64",
  "win32 x64 LE": "@bufbuild/buf-win32-x64",
};

export const knownUnixlikePackages: Record<string, string> = {
  "darwin arm64 LE": "@bufbuild/buf-darwin-arm64",
  "darwin x64 LE": "@bufbuild/buf-darwin-x64",
  "linux arm64 LE": "@bufbuild/buf-linux-arm64",
  "linux x64 LE": "@bufbuild/buf-linux-x64",
};

export function pkgAndSubpathForCurrentPlatform(binKey = "buf"): {
  pkg: string;
  subpath: string;
} {
  let pkg: string;
  let subpath: string;
  let platformKey = `${process.platform} ${os.arch()} ${os.endianness()}`;

  if (platformKey in knownWindowsPackages) {
    pkg = knownWindowsPackages[platformKey];
    subpath = `bin/${binKey}.exe`;
  } else if (platformKey in knownUnixlikePackages) {
    pkg = knownUnixlikePackages[platformKey];
    subpath = `bin/${binKey}`;
  } else {
    throw new Error(`Unsupported platform: ${platformKey}`);
  }

  return { pkg, subpath };
}

export function generateBinPath(binKey: string): { binPath: string } {
  const { pkg, subpath } = pkgAndSubpathForCurrentPlatform(binKey);
  return {
    binPath: require.resolve(`${pkg}/${subpath}`),
  };
}
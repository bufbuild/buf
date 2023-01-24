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

/**
 * This file is largely based off of https://github.com/evanw/esbuild/blob/3c83a84d01e22664923b543998b5c03c0c5d8654/lib/npm/node-install.ts
 * with some modifications to only patch binary paths.
 */

import { pkgAndSubpathForCurrentPlatform } from "./platform";

import child_process = require("child_process");

// Input defined in prepack
declare const CURRENT_VERSION: string;
// Input defined in prepack
declare const ALL_BINS: Record<string, string>;

function validateBinaryVersion(...command: string[]): void {
  command.push("--version");
  const stdout = child_process
    .execFileSync(command.shift()!, command, {
      // Without this, this install script strangely crashes with the error
      // "EACCES: permission denied, write" but only on Ubuntu Linux when node is
      // installed from the Snap Store. This is not a problem when you download
      // the official version of node. The problem appears to be that stderr
      // (i.e. file descriptor 2) isn't writable?
      //
      // More info:
      // - https://snapcraft.io/ (what the Snap Store is)
      // - https://nodejs.org/dist/ (download the official version of node)
      // - https://github.com/evanw/esbuild/issues/1711#issuecomment-1027554035
      //
      stdio: "pipe",
    })
    .toString()
    .trim();
  if (stdout !== CURRENT_VERSION) {
    throw new Error(
      `Expected ${JSON.stringify(CURRENT_VERSION)} but got ${JSON.stringify(
        stdout
      )}`
    );
  }
}

async function checkAndPreparePackage(binKey: string) {
  const { pkg, subpath } = pkgAndSubpathForCurrentPlatform(binKey);

  let binPath: string;
  try {
    // First check for the binary package from our "optionalDependencies". This
    // package should have been installed alongside this package at install time.
    binPath = require.resolve(`${pkg}/${subpath}`);
  } catch (e) {
    throw new Error(`[buf] Failed to find package "${pkg}" on the file system

This can happen if you use the "--no-optional" flag. The "optionalDependencies"
package.json feature is used by buf to install the correct binary executable
for your current platform.
`);
  }
  return {
    toPath: binPath,
  };
}

async function checkAllBinaries() {
  for (const [binKey] of Object.entries(ALL_BINS)) {
    const { toPath } = await checkAndPreparePackage(binKey);
    if (binKey === "buf") {
      // We can only verify the version of the primary buf binary
      validateBinaryVersion(toPath);
    }
  }
}

checkAllBinaries();

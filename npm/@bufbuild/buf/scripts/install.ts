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
import child_process from "child_process";
import path from "path";
import fs from "fs";

// Input defined in prepack
declare const CURRENT_VERSION: string;
// Input defined in prepack
declare const ALL_BINS: Record<string, string>;

function downloadedBinPath(pkg: string, subpath: string): string {
  const rootLibDir = path.dirname(require.resolve('@bufbuild/buf'))
  return path.join(rootLibDir, `downloaded-${pkg.replace('/', '-')}-${path.basename(subpath)}`)
}

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
    console.warn(`[buf] Failed to find package "${pkg}" on the file system

This can happen if you use the "--no-optional" flag. The "optionalDependencies"
package.json feature is used by buf to install the correct binary executable
for your current platform. Going to try installing from npm directly.
`);
    binPath = downloadedBinPath(pkg, subpath);
    console.error(`[buf] Trying to install package "${pkg}" using npm`);
    installUsingNPM(pkg, subpath, binPath);
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

function installUsingNPM(pkg: string, subpath: string, binPath: string): void {
  // Erase "npm_config_global" so that "npm install --global @bufbuild/buf" works.
  // Otherwise this nested "npm install" will also be global, and the install
  // will deadlock waiting for the global installation lock.
  const env = { ...process.env, npm_config_global: undefined };

  // Create a temporary directory inside the "@bufbuild/buf" package with an empty
  // "package.json" file. We'll use this to run "npm install" in.
  const libDir = path.dirname(require.resolve("@bufbuild/buf"));
  const installDir = path.join(libDir, "npm-install");
  fs.mkdirSync(installDir);
  try {
    fs.writeFileSync(path.join(installDir, "package.json"), "{}");

    // Run "npm install" in the temporary directory which should download the
    // desired package. Try to avoid unnecessary log output. This uses the "npm"
    // command instead of a HTTP request so that it hopefully works in situations
    // where HTTP requests are blocked but the "npm" command still works due to,
    // for example, a custom configured npm registry and special firewall rules.
    child_process.execSync(
      `npm install --loglevel=error --prefer-offline --no-audit --progress=false ${pkg}@${CURRENT_VERSION}`,
      { cwd: installDir, stdio: "pipe", env }
    );

    // Move the downloaded binary executable into place. The destination path
    // is the same one that the JavaScript API code uses so it will be able to
    // find the binary executable here later.
    const installedBinPath = path.join(
      installDir,
      "node_modules",
      pkg,
      subpath
    );
    fs.renameSync(installedBinPath, binPath);
  } finally {
    // Try to clean up afterward so we don't unnecessarily waste file system
    // space. Leaving nested "node_modules" directories can also be problematic
    // for certain tools that scan over the file tree and expect it to have a
    // certain structure.
    try {
      removeRecursive(installDir);
    } catch {
      // Removing a file or directory can randomly break on Windows, returning
      // EBUSY for an arbitrary length of time. I think this happens when some
      // other program has that file or directory open (e.g. an anti-virus
      // program). This is fine on Unix because the OS just unlinks the entry
      // but keeps the reference around until it's unused. There's nothing we
      // can do in this case so we just leave the directory there.
    }
  }
}

function removeRecursive(dir: string): void {
  for (const entry of fs.readdirSync(dir)) {
    const entryPath = path.join(dir, entry);
    let stats;
    try {
      stats = fs.lstatSync(entryPath);
    } catch {
      continue; // Guard against https://github.com/nodejs/node/issues/4760
    }
    if (stats.isDirectory()) removeRecursive(entryPath);
    else fs.unlinkSync(entryPath);
  }
  fs.rmdirSync(dir);
}

checkAllBinaries();

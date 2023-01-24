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

const path = require("path");
const fs = require("fs");

/** @type {import("esbuild")} */
const esbuild = require("esbuild");

const binDir = path.join(__dirname, "bin");

const pkgInfo = require("./package.json");

async function prepare() {
  fs.mkdirSync(binDir, { recursive: true });

  // Create the postinstall script that will be run after installation.
  await esbuild.build({
    entryPoints: [path.join(__dirname, "scripts", "install.ts")],
    outfile: path.join(__dirname, "install.js"),
    target: "node12",
    bundle: true,
    platform: "node",
    logLevel: "warning",
    external: ["@bufbuild/buf"],
    define: {
        CURRENT_VERSION: JSON.stringify(pkgInfo.version),
        ALL_BINS: JSON.stringify(pkgInfo.bin),
    },
  });

  for (const [binKey, platformBinPath] of Object.entries(pkgInfo.bin)) {
    // Create the postinstall script that will be run after installation.
    await esbuild.build({
      entryPoints: [path.join(__dirname, "scripts", "bin.ts")],
      outfile: platformBinPath,
      target: "node12",
      bundle: true,
      platform: "node",
      logLevel: "warning",
      external: ["@bufbuild/buf"],
      define: {
        BIN_KEY: JSON.stringify(binKey),
        CURRENT_VERSION: JSON.stringify(pkgInfo.version),
      },
    });
    // Prepend the shebang to the resulting bin file.
    prependSehbang(platformBinPath);
  }
}

function prependSehbang(file) {
  const data = fs.readFileSync(file)
  const fd = fs.openSync(file, 'w+')
  const insert = Buffer.from("#!/usr/bin/env node\n")
  fs.writeSync(fd, insert, 0, insert.length, 0)
  fs.writeSync(fd, data, 0, data.length, insert.length)
  fs.close(fd, (err) => {
    if (err) throw err;
  });
}

prepare();
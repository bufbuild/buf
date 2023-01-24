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

// This script acts as a proxy for an arbitrary binary.
import { generateBinPath } from "./platform";

// Input defined in prepack
declare const BIN_KEY: string;

const { binPath } = generateBinPath(BIN_KEY);

require("child_process").execFileSync(binPath, process.argv.slice(2), {
  stdio: "inherit",
});

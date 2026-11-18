#!/usr/bin/env node
// npm 으로 깐 tidetop: 이 컴퓨터에 맞는 실행 파일(@tidepark/tidetop-<운영체제>-<칩>)을 찾아 띄운다.
// The npm launcher: finds the binary for this platform and runs it.
"use strict";
const { spawnSync } = require("node:child_process");

const name = `@tidepark/tidetop-${process.platform}-${process.arch}`;
const file = process.platform === "win32" ? "tidetop.exe" : "tidetop";
let bin;
try {
  bin = require.resolve(`${name}/bin/${file}`);
} catch {
  console.error(
    `tidetop: no binary for ${process.platform}-${process.arch} (${name} is not installed).\n` +
      "Reinstall without --no-optional, or download from https://github.com/tidepark/tidetop/releases",
  );
  process.exit(1);
}
const run = spawnSync(bin, process.argv.slice(2), { stdio: "inherit" });
if (run.error) {
  console.error(`tidetop: ${run.error.message}`);
  process.exit(1);
}
if (run.signal) process.kill(process.pid, run.signal);
process.exit(run.status ?? 1);

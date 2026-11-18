// GoReleaser 가 만든 실행 파일로 npm 꾸러미를 만든다.
// 쓰는 법: node npm/build.mjs <dist 폴더> <판 번호>   → npm/dist/ 에 꾸러미 일곱
//   - @tidepark/tidetop-<운영체제>-<칩> 여섯: 실행 파일 하나씩
//   - tidetop: 위 여섯을 선택 의존성으로 걸고, 맞는 것을 찾아 띄우는 짧은 JavaScript
import { chmodSync, copyFileSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const root = resolve(here, "..");
const [distArg, version] = process.argv.slice(2);
if (!distArg || !/^\d+\.\d+\.\d+(-[\w.]+)?$/.test(version ?? "")) {
  console.error("쓰는 법: node npm/build.mjs <dist 폴더> <판 번호 예: 0.1.0>");
  process.exit(1);
}
const dist = resolve(distArg);
const out = join(here, "dist");
rmSync(out, { recursive: true, force: true });

const OS = { darwin: "darwin", linux: "linux", windows: "win32" };
const ARCH = { amd64: "x64", arm64: "arm64" };
const common = {
  version,
  license: "MIT",
  homepage: "https://github.com/tidepark/tidetop",
  repository: { type: "git", url: "git+https://github.com/tidepark/tidetop.git" },
  author: "Tidepark <hello@tidepark.com>",
};

const artifacts = JSON.parse(readFileSync(join(dist, "artifacts.json"), "utf8")).filter((a) => a.type === "Binary");
const optional = {};
for (const a of artifacts) {
  const os = OS[a.goos];
  const cpu = ARCH[a.goarch];
  if (!os || !cpu) continue;
  const name = `@tidepark/tidetop-${os}-${cpu}`;
  const dir = join(out, `tidetop-${os}-${cpu}`);
  const file = os === "win32" ? "tidetop.exe" : "tidetop";
  mkdirSync(join(dir, "bin"), { recursive: true });
  copyFileSync(resolve(root, a.path), join(dir, "bin", file));
  chmodSync(join(dir, "bin", file), 0o755);
  writeFileSync(
    join(dir, "package.json"),
    JSON.stringify({ name, description: `tidetop binary for ${os}-${cpu}`, ...common, os: [os], cpu: [cpu], files: ["bin"] }, null, 2) + "\n",
  );
  copyFileSync(join(root, "LICENSE"), join(dir, "LICENSE"));
  optional[name] = version;
}
if (Object.keys(optional).length !== 6) {
  console.error(`실행 파일이 여섯이 아니다: ${Object.keys(optional).join(", ")}`);
  process.exit(1);
}

const main = join(out, "tidetop");
mkdirSync(join(main, "bin"), { recursive: true });
copyFileSync(join(here, "tidetop", "bin", "tidetop.js"), join(main, "bin", "tidetop.js"));
chmodSync(join(main, "bin", "tidetop.js"), 0o755);
for (const f of ["LICENSE", "README.md"]) copyFileSync(join(root, f), join(main, f));
writeFileSync(
  join(main, "package.json"),
  JSON.stringify(
    {
      name: "tidetop",
      description: "A tidepic scene in your terminal: your computer's headroom as a scene, not numbers",
      ...common,
      keywords: ["terminal", "cli", "monitor", "cpu", "tidepic"],
      bin: { tidetop: "bin/tidetop.js" },
      files: ["bin", "README.md", "LICENSE"],
      engines: { node: ">=16" },
      optionalDependencies: optional,
    },
    null,
    2,
  ) + "\n",
);
console.log(`npm/dist: tidetop ${version} + ${Object.keys(optional).length} platform packages`);

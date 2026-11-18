// npm/dist 의 꾸러미를 npm 에 올린다. 운영체제별 꾸러미를 먼저, 본 꾸러미를 맨 나중에 올린다.
// 쓰는 법: node npm/publish.mjs [--dry-run]
import { execFileSync } from "node:child_process";
import { readdirSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const out = join(dirname(fileURLToPath(import.meta.url)), "dist");
const extra = process.argv.includes("--dry-run") ? ["--dry-run"] : [];
const dirs = readdirSync(out).sort((a, b) => (a === "tidetop") - (b === "tidetop"));
for (const d of dirs) {
  console.log(`npm publish ${d}`);
  execFileSync("npm", ["publish", join(out, d), "--access", "public", ...extra], { stdio: "inherit" });
}

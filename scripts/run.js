#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const { execFileSync } = require("child_process");

const isWindows = process.platform === "win32";
const bin = path.join(__dirname, "..", "bin", "luma-cli" + (isWindows ? ".exe" : ""));
const installScript = path.join(__dirname, "install.js");
const args = process.argv.slice(2);

if (args[0] === "install") {
  execFileSync(process.execPath, [installScript], {
    stdio: "inherit",
    env: { ...process.env, LUMA_CLI_RUN: "1" },
  });
  process.exit(0);
}

if (!fs.existsSync(bin)) {
  execFileSync(process.execPath, [installScript], {
    stdio: "inherit",
    env: { ...process.env, LUMA_CLI_RUN: "1" },
  });
}

try {
  execFileSync(bin, args, { stdio: "inherit" });
} catch (err) {
  process.exit(err.status || 1);
}

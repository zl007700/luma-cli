#!/usr/bin/env node

const crypto = require("crypto");
const fs = require("fs");
const os = require("os");
const path = require("path");
const { execFileSync } = require("child_process");

const pkg = require("../package.json");

const VERSION = pkg.version.replace(/-.*/, "");
const REPO = "zl007700/luma-cli";
const NAME = "luma-cli";

const PLATFORM_MAP = {
  darwin: "darwin",
  linux: "linux",
  win32: "windows",
};

const ARCH_MAP = {
  x64: "amd64",
  arm64: "arm64",
};

const platform = PLATFORM_MAP[process.platform];
const arch = ARCH_MAP[process.arch];
const isWindows = process.platform === "win32";
const ext = isWindows ? ".zip" : ".tar.gz";
const archiveName = `${NAME}-${VERSION}-${platform}-${arch}${ext}`;
const releaseURL = `https://github.com/${REPO}/releases/download/v${VERSION}/${archiveName}`;

const binDir = path.join(__dirname, "..", "bin");
const dest = path.join(binDir, NAME + (isWindows ? ".exe" : ""));

function printHelp() {
  console.log(`${NAME} installer`);
  console.log("");
  console.log(`Downloads ${archiveName} from:`);
  console.log(`  ${releaseURL}`);
}

function download(url, destPath) {
  const args = [
    "--fail",
    "--location",
    "--silent",
    "--show-error",
    "--connect-timeout",
    "10",
    "--max-time",
    "120",
    "--max-redirs",
    "3",
    "--output",
    destPath,
    url,
  ];
  if (isWindows) {
    args.unshift("--ssl-revoke-best-effort");
  }
  execFileSync("curl", args, { stdio: ["ignore", "ignore", "pipe"] });
}

function expectedChecksum(name) {
  const checksumsPath = path.join(__dirname, "..", "checksums.txt");
  if (!fs.existsSync(checksumsPath)) {
    console.warn("[WARN] checksums.txt not found, skipping checksum verification");
    return null;
  }
  const lines = fs.readFileSync(checksumsPath, "utf8").split(/\r?\n/);
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed) continue;
    const parts = trimmed.split(/\s+/);
    if (parts.length >= 2 && parts[parts.length - 1] === name) {
      return parts[0];
    }
  }
  throw new Error(`Checksum entry not found for ${name}`);
}

function verifyChecksum(filePath, expected) {
  if (!expected) return;
  const hash = crypto.createHash("sha256");
  const fd = fs.openSync(filePath, "r");
  try {
    const buf = Buffer.alloc(64 * 1024);
    let bytesRead;
    while ((bytesRead = fs.readSync(fd, buf, 0, buf.length, null)) > 0) {
      hash.update(buf.subarray(0, bytesRead));
    }
  } finally {
    fs.closeSync(fd);
  }
  const actual = hash.digest("hex");
  if (actual.toLowerCase() !== expected.toLowerCase()) {
    throw new Error(`Checksum mismatch for ${path.basename(filePath)}: expected ${expected}, got ${actual}`);
  }
}

function extractArchive(archivePath, tmpDir) {
  if (isWindows) {
    const ps = [
      "-NoProfile",
      "-ExecutionPolicy",
      "Bypass",
      "-Command",
      `$ErrorActionPreference='Stop'; Expand-Archive -LiteralPath '${archivePath.replace(/'/g, "''")}' -DestinationPath '${tmpDir.replace(/'/g, "''")}' -Force`,
    ];
    execFileSync("powershell.exe", ps, { stdio: "inherit" });
    return;
  }
  execFileSync("tar", ["-xzf", archivePath, "-C", tmpDir], { stdio: "ignore" });
}

function install() {
  if (!platform || !arch) {
    throw new Error(`Unsupported platform: ${process.platform}-${process.arch}`);
  }

  fs.mkdirSync(binDir, { recursive: true });
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), `${NAME}-`));
  const archivePath = path.join(tmpDir, archiveName);

  try {
    download(releaseURL, archivePath);
    verifyChecksum(archivePath, expectedChecksum(archiveName));
    extractArchive(archivePath, tmpDir);

    const binaryName = NAME + (isWindows ? ".exe" : "");
    const extracted = path.join(tmpDir, binaryName);
    if (!fs.existsSync(extracted)) {
      throw new Error(`Expected binary not found in archive: ${binaryName}`);
    }

    fs.copyFileSync(extracted, dest);
    fs.chmodSync(dest, 0o755);
    console.log(`${NAME} v${VERSION} installed successfully`);
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

if (require.main === module) {
  if (process.argv.includes("--help")) {
    printHelp();
    process.exit(0);
  }

  const isNpxPostinstall = process.env.npm_command === "exec" && !process.env.LUMA_CLI_RUN;
  if (isNpxPostinstall) {
    process.exit(0);
  }

  try {
    install();
  } catch (err) {
    console.error(`Failed to install ${NAME}: ${err.message || err}`);
    console.error("");
    console.error("You can also download the binary manually from:");
    console.error(`  https://github.com/${REPO}/releases/tag/v${VERSION}`);
    process.exit(1);
  }
}

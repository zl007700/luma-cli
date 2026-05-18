#!/usr/bin/env node

const crypto = require("crypto");
const fs = require("fs");
const http = require("http");
const https = require("https");
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
const packageRoot = path.join(__dirname, "..");

function printHelp() {
  console.log(`${NAME} installer`);
  console.log("");
  console.log(`Downloads ${archiveName} from:`);
  console.log(`  ${releaseURL}`);
}

function download(url, destPath, redirects = 0) {
  if (redirects > 3) {
    return Promise.reject(new Error(`Too many redirects while downloading ${url}`));
  }

  const client = url.startsWith("https:") ? https : http;
  return new Promise((resolve, reject) => {
    const req = client.get(url, { timeout: 120000 }, (res) => {
      const status = res.statusCode || 0;
      const location = res.headers.location;
      if (status >= 300 && status < 400 && location) {
        res.resume();
        const nextURL = new URL(location, url).toString();
        download(nextURL, destPath, redirects + 1).then(resolve, reject);
        return;
      }

      if (status < 200 || status >= 300) {
        res.resume();
        reject(new Error(`download failed: HTTP ${status}`));
        return;
      }

      const out = fs.createWriteStream(destPath);
      res.pipe(out);
      out.on("finish", () => out.close(resolve));
      out.on("error", reject);
    });
    req.on("timeout", () => req.destroy(new Error("download timed out")));
    req.on("error", reject);
  });
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

function firstExisting(paths) {
  for (const candidate of paths) {
    if (candidate && fs.existsSync(candidate)) return candidate;
  }
  return null;
}

function windowsTool(name) {
  const root = process.env.SystemRoot || process.env.windir || "C:\\Windows";
  if (name === "powershell.exe") {
    return firstExisting([
      path.join(root, "System32", "WindowsPowerShell", "v1.0", "powershell.exe"),
      path.join(root, "Sysnative", "WindowsPowerShell", "v1.0", "powershell.exe"),
      "powershell.exe",
    ]);
  }
  if (name === "tar.exe") {
    return firstExisting([
      path.join(root, "System32", "tar.exe"),
      path.join(root, "Sysnative", "tar.exe"),
      "tar.exe",
    ]);
  }
  return name;
}

function unixTool(name) {
  if (name === "tar") {
    return firstExisting([
      "/usr/bin/tar",
      "/bin/tar",
      "/usr/local/bin/tar",
      "/opt/homebrew/bin/tar",
      "tar",
    ]);
  }
  return name;
}

function splitPathList(value) {
  return (value || "")
    .split(path.delimiter)
    .map((item) => item.trim())
    .filter(Boolean);
}

function samePath(a, b) {
  const left = path.resolve(a).replace(/[\\\/]+$/, "");
  const right = path.resolve(b).replace(/[\\\/]+$/, "");
  return isWindows ? left.toLowerCase() === right.toLowerCase() : left === right;
}

function pathContains(dir, value) {
  return splitPathList(value).some((entry) => samePath(entry, dir));
}

function npmGlobalPrefix() {
  if (process.env.npm_config_prefix) return process.env.npm_config_prefix;
  return path.resolve(__dirname, "..", "..", "..", "..");
}

function writeFileIfChanged(filePath, content, mode) {
  if (fs.existsSync(filePath) && fs.readFileSync(filePath, "utf8") === content) return;
  fs.writeFileSync(filePath, content, "utf8");
  if (mode) fs.chmodSync(filePath, mode);
}

function canWriteDirectory(dir) {
  try {
    fs.mkdirSync(dir, { recursive: true });
    const probe = path.join(dir, `.${NAME}-${process.pid}.tmp`);
    fs.writeFileSync(probe, "");
    fs.rmSync(probe, { force: true });
    return true;
  } catch (_) {
    return false;
  }
}

function addWindowsUserPath(dir) {
  if (!isWindows) return false;
  const powershell = windowsTool("powershell.exe");
  if (!powershell) return false;

  const script =
    "$ErrorActionPreference='Stop';" +
    "$dir=$env:LUMA_CLI_PATH_DIR;" +
    "$path=[Environment]::GetEnvironmentVariable('Path','User');" +
    "$parts=@($path -split ';' | Where-Object { $_ -and $_.Trim() -ne '' });" +
    "$exists=$false;" +
    "foreach($p in $parts){ if($p.TrimEnd('\\') -ieq $dir.TrimEnd('\\')){ $exists=$true } }" +
    "if(-not $exists){" +
    "  $new=($dir+';'+($parts -join ';')).TrimEnd(';');" +
    "  [Environment]::SetEnvironmentVariable('Path',$new,'User')" +
    "}";

  execFileSync(
    powershell,
    ["-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script],
    {
      stdio: "ignore",
      env: { ...process.env, LUMA_CLI_PATH_DIR: dir },
    }
  );
  return true;
}

function userWritablePathDirs() {
  const home = os.homedir();
  const disallowed = [
    `${path.sep}.codex${path.sep}tmp${path.sep}`,
    `${path.sep}Temp${path.sep}`,
    `${path.sep}tmp${path.sep}`,
  ].map((item) => (isWindows ? item.toLowerCase() : item));

  return splitPathList(process.env.PATH)
    .filter((entry) => {
      const resolved = path.resolve(entry);
      const comparable = isWindows ? resolved.toLowerCase() : resolved;
      const homeComparable = isWindows ? home.toLowerCase() : home;
      if (!comparable.startsWith(homeComparable)) return false;
      return !disallowed.some((part) => comparable.includes(part));
    })
    .filter((entry, index, all) => all.findIndex((candidate) => samePath(candidate, entry)) === index);
}

function createPathShim(targetDir) {
  const runScript = path.join(packageRoot, "scripts", "run.js");
  if (isWindows) {
    const cmdPath = path.join(targetDir, `${NAME}.cmd`);
    const psPath = path.join(targetDir, `${NAME}.ps1`);
    writeFileIfChanged(
      cmdPath,
      `@ECHO off\r\n"${process.execPath}" "${runScript}" %*\r\n`,
      0o755
    );
    writeFileIfChanged(
      psPath,
      `& "${process.execPath}" "${runScript}" @args\r\nexit $LASTEXITCODE\r\n`,
      0o755
    );
    return cmdPath;
  }

  const shimPath = path.join(targetDir, NAME);
  writeFileIfChanged(
    shimPath,
    `#!/usr/bin/env sh\nexec "${process.execPath}" "${runScript}" "$@"\n`,
    0o755
  );
  return shimPath;
}

function ensureCommandReachable() {
  const prefix = npmGlobalPrefix();
  const commandDir = isWindows ? prefix : path.join(prefix, "bin");
  const currentPath = process.env.PATH || "";
  if (pathContains(commandDir, currentPath)) return;

  if (isWindows) {
    try {
      addWindowsUserPath(commandDir);
      console.log(`${NAME}: added ${commandDir} to the user PATH for new terminals`);
    } catch (err) {
      console.warn(`${NAME}: could not update the user PATH automatically: ${err.message || err}`);
    }
  }

  for (const candidate of userWritablePathDirs()) {
    if (!canWriteDirectory(candidate)) continue;
    try {
      const shim = createPathShim(candidate);
      console.log(`${NAME}: command shim available at ${shim}`);
      return;
    } catch (_) {
      // Try the next writable PATH entry.
    }
  }

  console.warn(`${NAME}: ${commandDir} is not in PATH. Restart your terminal after installation.`);
}

function extractArchive(archivePath, tmpDir) {
  if (isWindows) {
    const powershell = windowsTool("powershell.exe");
    if (powershell) {
      try {
        const psEnv = {
          ...process.env,
          LUMA_CLI_ARCHIVE: archivePath,
          LUMA_CLI_DEST: tmpDir,
        };
        const ps = [
          "-NoProfile",
          "-ExecutionPolicy",
          "Bypass",
          "-Command",
          "$ErrorActionPreference='Stop'; Expand-Archive -LiteralPath $env:LUMA_CLI_ARCHIVE -DestinationPath $env:LUMA_CLI_DEST -Force",
        ];
        execFileSync(powershell, ps, { stdio: "inherit", env: psEnv });
        return;
      } catch (err) {
        console.warn(`[WARN] PowerShell extraction failed: ${err.message || err}`);
      }
    }

    const tar = windowsTool("tar.exe");
    if (tar) {
      execFileSync(tar, ["-xf", archivePath, "-C", tmpDir], { stdio: "inherit" });
      return;
    }

    throw new Error("No supported archive extractor found. Install PowerShell or ensure C:\\Windows\\System32 is in PATH.");
  }

  const tar = unixTool("tar");
  if (!tar) {
    throw new Error("No supported archive extractor found. Install tar or ensure it is in PATH.");
  }
  execFileSync(tar, ["-xzf", archivePath, "-C", tmpDir], { stdio: "ignore" });
}

async function install() {
  if (!platform || !arch) {
    throw new Error(`Unsupported platform: ${process.platform}-${process.arch}`);
  }

  fs.mkdirSync(binDir, { recursive: true });
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), `${NAME}-`));
  const archivePath = path.join(tmpDir, archiveName);

  try {
    await download(releaseURL, archivePath);
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
    ensureCommandReachable();
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

  install().catch((err) => {
    console.error(`Failed to install ${NAME}: ${err.message || err}`);
    console.error("");
    console.error("You can also download the binary manually from:");
    console.error(`  https://github.com/${REPO}/releases/tag/v${VERSION}`);
    process.exit(1);
  });
}

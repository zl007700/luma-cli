#!/usr/bin/env node
const fs = require("fs");
const os = require("os");
const path = require("path");
const http = require("http");
const { spawn } = require("child_process");

function arg(name, fallback = "") {
  const index = process.argv.indexOf(`--${name}`);
  return index >= 0 && index + 1 < process.argv.length ? process.argv[index + 1] : fallback;
}

function writeJSON(file, value) {
  fs.mkdirSync(path.dirname(path.resolve(file)), { recursive: true });
  fs.writeFileSync(file, JSON.stringify(value, null, 2), "utf8");
}

function browserCandidates() {
  if (process.env.LUMA_BROWSER_PATH) return [process.env.LUMA_BROWSER_PATH];
  if (process.platform === "win32") {
    return [
      "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
      "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe",
      "C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe",
      "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe"
    ];
  }
  if (process.platform === "darwin") {
    return [
      "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
      "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"
    ];
  }
  return ["/usr/bin/google-chrome", "/usr/bin/chromium", "/usr/bin/chromium-browser", "/usr/bin/microsoft-edge"];
}

function findBrowser() {
  return browserCandidates().find((candidate) => candidate && fs.existsSync(candidate)) || "";
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function waitForExit(child, timeoutMs = 3000) {
  if (!child || child.exitCode !== null) return Promise.resolve();
  return new Promise((resolve) => {
    const timer = setTimeout(resolve, timeoutMs);
    child.once("exit", () => {
      clearTimeout(timer);
      resolve();
    });
  });
}

async function removeProfileDir(profileDir) {
  for (let attempt = 0; attempt < 5; attempt += 1) {
    try {
      fs.rmSync(profileDir, { recursive: true, force: true });
      return;
    } catch (error) {
      if (!["EPERM", "EBUSY", "ENOTEMPTY"].includes(error.code) || attempt === 4) return;
      await sleep(200 * (attempt + 1));
    }
  }
}

async function waitForFile(file, timeoutMs) {
  const started = Date.now();
  while (Date.now() - started < timeoutMs) {
    if (fs.existsSync(file)) return;
    await sleep(100);
  }
  throw new Error("Browser debugging port was not created.");
}

function httpJSON(url) {
  return new Promise((resolve, reject) => {
    const request = http.get(url, (response) => {
      let body = "";
      response.setEncoding("utf8");
      response.on("data", (chunk) => { body += chunk; });
      response.on("end", () => {
        try {
          resolve(JSON.parse(body));
        } catch (error) {
          reject(new Error(`Invalid browser response: ${error.message}`));
        }
      });
    });
    request.on("error", reject);
  });
}

class CDP {
  constructor(url) {
    this.url = url;
    this.nextID = 1;
    this.pending = new Map();
    this.events = new Map();
  }

  async connect() {
    this.socket = new WebSocket(this.url);
    await new Promise((resolve, reject) => {
      const timer = setTimeout(() => reject(new Error("Browser connection timed out.")), 10000);
      this.socket.addEventListener("open", () => {
        clearTimeout(timer);
        resolve();
      });
      this.socket.addEventListener("error", () => {
        clearTimeout(timer);
        reject(new Error("Could not connect to browser."));
      });
    });
    this.socket.addEventListener("message", (event) => {
      const message = JSON.parse(String(event.data));
      if (message.id && this.pending.has(message.id)) {
        const { resolve, reject, timer } = this.pending.get(message.id);
        clearTimeout(timer);
        this.pending.delete(message.id);
        if (message.error) reject(new Error(message.error.message || "Browser command failed."));
        else resolve(message.result || {});
        return;
      }
      const listeners = this.events.get(message.method) || [];
      for (const listener of listeners) listener(message.params || {});
    });
  }

  send(method, params = {}, timeoutMs = 20000) {
    const id = this.nextID++;
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.pending.delete(id);
        reject(new Error(`${method} timed out.`));
      }, timeoutMs);
      this.pending.set(id, { resolve, reject, timer });
      this.socket.send(JSON.stringify({ id, method, params }));
    });
  }

  waitFor(method, timeoutMs = 20000) {
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        this.events.set(method, (this.events.get(method) || []).filter((item) => item !== listener));
        reject(new Error(`${method} timed out.`));
      }, timeoutMs);
      const listener = (params) => {
        clearTimeout(timer);
        this.events.set(method, (this.events.get(method) || []).filter((item) => item !== listener));
        resolve(params);
      };
      this.events.set(method, [...(this.events.get(method) || []), listener]);
    });
  }

  close() {
    if (this.socket && this.socket.readyState <= 1) this.socket.close();
  }
}

function keywordExpression(keywords) {
  return `(() => {
    const keywords = ${JSON.stringify(keywords)};
    const clean = (value) => String(value || "").replace(/\\s+/g, " ").trim().toLowerCase();
    const wanted = keywords.map(clean).filter(Boolean);
    const nodes = document.querySelectorAll("h1,h2,h3,p,li,td,th,article div,main div");
    let best = null;
    let bestNode = null;
    for (const node of nodes) {
      const text = clean(node.innerText);
      if (!text || text.length > 1600) continue;
      const score = wanted.reduce((total, keyword) => total + (text.includes(keyword) ? Math.max(2, keyword.length) : 0), 0);
      if (!score) continue;
      const rect = node.getBoundingClientRect();
      if (rect.width < 240 || rect.height < 18) continue;
      if (!best || score > best.score || (score === best.score && rect.height < best.rect.height)) {
        best = { score, text: text.slice(0, 500), rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height } };
        bestNode = node;
      }
    }
    if (!best) return null;
    if (bestNode) bestNode.setAttribute("data-luma-evidence", "true");
    window.scrollTo({ top: Math.max(0, window.scrollY + best.rect.y - 180), behavior: "instant" });
    return best;
  })()`;
}

const CLEAN_PAGE_EXPRESSION = `(() => {
  const viewportArea = Math.max(1, window.innerWidth * window.innerHeight);
  for (const element of document.querySelectorAll("body *")) {
    const style = getComputedStyle(element);
    const text = String(element.innerText || element.getAttribute("aria-label") || "").toLowerCase();
    const looksLikeModal = /(登录|登陆|注册|sign in|log in|login|subscribe|cookie|cookies|privacy|隐私|关注)/i.test(text);
    if (!["fixed", "sticky"].includes(style.position) && !looksLikeModal) continue;
    const rect = element.getBoundingClientRect();
    const area = Math.max(0, rect.width) * Math.max(0, rect.height);
    const centered = rect.width > window.innerWidth * 0.25 && rect.height > window.innerHeight * 0.15 &&
      rect.left < window.innerWidth * 0.75 && rect.right > window.innerWidth * 0.25;
    if (area > 0 && (area < viewportArea * 0.45 || (looksLikeModal && centered))) {
      element.style.setProperty("display", "none", "important");
    }
  }
  const evidence = document.querySelector('[data-luma-evidence="true"]');
  if (evidence) {
    evidence.style.setProperty("background", "#fff3a8", "important");
    evidence.style.setProperty("box-shadow", "0 0 0 10px #fff3a8", "important");
    evidence.style.setProperty("border-radius", "2px", "important");
  }
  return true;
})()`;

const EVIDENCE_CLIP_EXPRESSION = `(() => {
  const evidence = document.querySelector('[data-luma-evidence="true"]');
  if (!evidence) return null;
  const rect = evidence.getBoundingClientRect();
  if (rect.width < 80 || rect.height < 16) return null;
  const padX = 28;
  const padY = 36;
  const x = Math.max(0, Math.floor(rect.left - padX));
  const y = Math.max(0, Math.floor(rect.top - padY));
  const right = Math.min(window.innerWidth, Math.ceil(rect.right + padX));
  const bottom = Math.min(window.innerHeight, Math.ceil(rect.bottom + padY));
  const width = Math.max(320, right - x);
  const height = Math.max(180, bottom - y);
  if (width < window.innerWidth * 0.95 && height < window.innerHeight * 0.9) {
    return {
      x: Math.floor(window.scrollX + x),
      y: Math.floor(window.scrollY + y),
      width: Math.min(width, window.innerWidth - x),
      height: Math.min(height, window.innerHeight - y),
      scale: 1
    };
  }
  return null;
})()`;

async function evaluate(cdp, expression) {
  const result = await cdp.send("Runtime.evaluate", {
    expression,
    returnByValue: true,
    awaitPromise: true
  });
  return result.result ? result.result.value : undefined;
}

async function main() {
  const url = arg("url");
  const output = arg("output");
  const resultFile = arg("result", output ? `${output}.json` : "capture_result.json");
  const mode = arg("mode", "first-screen");
  const cropMode = arg("crop-mode", "evidence");
  const width = Math.max(640, Number(arg("width", "1280")));
  const height = Math.max(480, Number(arg("height", "720")));
  const timeoutMs = Math.max(5000, Number(arg("timeout-ms", "30000")));
  const waitMs = Math.max(0, Number(arg("wait-ms", "2500")));
  const keywords = arg("keywords", arg("keyword", ""))
    .split("|")
    .map((item) => item.trim())
    .filter(Boolean)
    .slice(0, 8);

  if (!url || !output) throw new Error("--url and --output are required");
  if (!["first-screen", "keyword"].includes(mode)) throw new Error("--mode must be first-screen or keyword");
  const parsed = new URL(url);
  if (!["http:", "https:"].includes(parsed.protocol)) throw new Error("Only HTTP(S) URLs are supported.");

  const browserPath = findBrowser();
  if (!browserPath) throw new Error("Chrome or Edge was not found. Set LUMA_BROWSER_PATH.");

  const profileDir = fs.mkdtempSync(path.join(os.tmpdir(), "luma-browser-"));
  const portFile = path.join(profileDir, "DevToolsActivePort");
  const browser = spawn(browserPath, [
    "--headless=new",
    "--disable-gpu",
    "--disable-extensions",
    "--disable-background-networking",
    "--disable-default-apps",
    "--disable-sync",
    "--hide-scrollbars",
    "--no-first-run",
    "--no-default-browser-check",
    "--remote-debugging-port=0",
    `--user-data-dir=${profileDir}`,
    "about:blank"
  ], { stdio: "ignore", windowsHide: true });

  let cdp;
  const result = {
    status: "failed",
    path: path.resolve(output),
    source_url: url,
    capture_mode: mode,
    fallback_used: false,
    width,
    height,
    title: "",
    matched_text: "",
    crop: null,
    error: null
  };

  try {
    await waitForFile(portFile, 10000);
    const [port] = fs.readFileSync(portFile, "utf8").trim().split(/\r?\n/);
    const targets = await httpJSON(`http://127.0.0.1:${port}/json`);
    const target = targets.find((item) => item.type === "page");
    if (!target || !target.webSocketDebuggerUrl) throw new Error("No browser page target was found.");

    cdp = new CDP(target.webSocketDebuggerUrl);
    await cdp.connect();
    await cdp.send("Page.enable");
    await cdp.send("Runtime.enable");
    await cdp.send("Emulation.setDeviceMetricsOverride", {
      width,
      height,
      deviceScaleFactor: 1,
      mobile: false
    });

    const loaded = cdp.waitFor("Page.loadEventFired", timeoutMs).catch(() => null);
    await cdp.send("Page.navigate", { url }, timeoutMs);
    await loaded;
    await sleep(waitMs);

    result.title = await evaluate(cdp, "document.title") || "";
    const bodyTextLength = await evaluate(cdp, "(document.body && document.body.innerText || '').trim().length") || 0;
    if (bodyTextLength < 20) throw new Error("Page rendered without readable content.");

    if (mode === "keyword" && keywords.length) {
      const match = await evaluate(cdp, keywordExpression(keywords));
      if (match) {
        result.matched_text = match.text || "";
        await sleep(400);
      } else {
        result.fallback_used = true;
        result.capture_mode = "first-screen";
        await evaluate(cdp, "window.scrollTo(0, 0)");
        await sleep(200);
      }
    }

    await evaluate(cdp, CLEAN_PAGE_EXPRESSION);
    await sleep(200);
    let clip = null;
    if (cropMode !== "viewport" && result.matched_text) {
      clip = await evaluate(cdp, EVIDENCE_CLIP_EXPRESSION);
      if (clip) result.crop = clip;
    }
    const screenshot = await cdp.send("Page.captureScreenshot", {
      format: "png",
      fromSurface: true,
      captureBeyondViewport: Boolean(clip),
      ...(clip ? { clip } : {})
    }, timeoutMs);
    const bytes = Buffer.from(screenshot.data || "", "base64");
    const validPNG = bytes.length > 1024 && bytes.subarray(0, 8).equals(Buffer.from("89504e470d0a1a0a", "hex"));
    if (!validPNG) throw new Error("Browser did not return a valid PNG screenshot.");

    fs.mkdirSync(path.dirname(path.resolve(output)), { recursive: true });
    fs.writeFileSync(output, bytes);
    result.status = "ready";
  } catch (error) {
    result.error = String(error.message || error);
  } finally {
    if (cdp) cdp.close();
    if (!browser.killed) browser.kill();
    await waitForExit(browser);
    await removeProfileDir(profileDir);
    writeJSON(resultFile, result);
  }

  process.stdout.write(`${JSON.stringify(result, null, 2)}\n`);
  if (result.status !== "ready") process.exitCode = 1;
}

main().catch((error) => {
  process.stderr.write(`${error.stack || error}\n`);
  process.exitCode = 1;
});

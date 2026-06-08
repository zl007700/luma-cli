#!/usr/bin/env node
const crypto = require("crypto");
const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");

function arg(name, fallback = "") {
  const index = process.argv.indexOf(`--${name}`);
  return index >= 0 && index + 1 < process.argv.length ? process.argv[index + 1] : fallback;
}

function readJSON(file) {
  return JSON.parse(fs.readFileSync(file, "utf8"));
}

function writeJSON(file, value) {
  fs.mkdirSync(path.dirname(path.resolve(file)), { recursive: true });
  fs.writeFileSync(file, JSON.stringify(value, null, 2), "utf8");
}

function signals(payload) {
  if (Array.isArray(payload.raw_signals)) return payload.raw_signals;
  if (payload.result && Array.isArray(payload.result.raw_signals)) return payload.result.raw_signals;
  if (payload.data?.result && Array.isArray(payload.data.result.raw_signals)) return payload.data.result.raw_signals;
  return [];
}

function dimensions(buffer) {
  if (buffer.length >= 24 && buffer.subarray(0, 8).equals(Buffer.from("89504e470d0a1a0a", "hex"))) {
    return { format: "png", width: buffer.readUInt32BE(16), height: buffer.readUInt32BE(20) };
  }
  if (buffer.length >= 10 && buffer.subarray(0, 3).toString("ascii") === "GIF") {
    return { format: "gif", width: buffer.readUInt16LE(6), height: buffer.readUInt16LE(8) };
  }
  if (buffer.length >= 30 && buffer.subarray(0, 4).toString("ascii") === "RIFF" && buffer.subarray(8, 12).toString("ascii") === "WEBP") {
    const type = buffer.subarray(12, 16).toString("ascii");
    if (type === "VP8X") {
      return {
        format: "webp",
        width: 1 + buffer.readUIntLE(24, 3),
        height: 1 + buffer.readUIntLE(27, 3)
      };
    }
  }
  if (buffer.length >= 4 && buffer[0] === 0xff && buffer[1] === 0xd8) {
    let offset = 2;
    while (offset + 9 < buffer.length) {
      if (buffer[offset] !== 0xff) {
        offset += 1;
        continue;
      }
      const marker = buffer[offset + 1];
      offset += 2;
      if (marker === 0xd8 || marker === 0xd9) continue;
      if (offset + 2 > buffer.length) break;
      const length = buffer.readUInt16BE(offset);
      if ([0xc0, 0xc1, 0xc2, 0xc3, 0xc5, 0xc6, 0xc7, 0xc9, 0xca, 0xcb, 0xcd, 0xce, 0xcf].includes(marker)) {
        return {
          format: "jpg",
          width: buffer.readUInt16BE(offset + 5),
          height: buffer.readUInt16BE(offset + 3)
        };
      }
      if (length < 2) break;
      offset += length;
    }
  }
  return null;
}

function safeURL(value) {
  try {
    const parsed = new URL(String(value || ""));
    return ["http:", "https:"].includes(parsed.protocol) ? parsed.toString() : "";
  } catch {
    return "";
  }
}

function fileExists(file) {
  try {
    return fs.statSync(file).isFile();
  } catch {
    return false;
  }
}

function resolveCLI(value) {
  if (value && value !== "luma-cli") return value;
  if (process.platform === "win32" && process.env.APPDATA) {
    const installed = path.join(
      process.env.APPDATA,
      "npm",
      "node_modules",
      "@lumageo",
      "luma-cli",
      "bin",
      "luma-cli.exe"
    );
    if (fileExists(installed)) return installed;
  }
  return value || "luma-cli";
}

function reviewImage(cli, imagePath, reviewPath, topic, claim, purpose) {
  const result = spawnSync(cli, [
    "material", "review", imagePath,
    "--topic", topic,
    "--claim", claim,
    "--purpose", purpose,
    "--output", reviewPath
  ], {
    cwd: process.cwd(),
    encoding: "utf8",
    windowsHide: true
  });
  if (result.error) throw result.error;
  if (result.status !== 0) {
    throw new Error(String(result.stderr || result.stdout || `material review exited ${result.status}`).trim());
  }
  if (!fileExists(reviewPath)) throw new Error("Material review did not create an output file.");
  return readJSON(reviewPath);
}

async function download(url, sourceURL, timeoutMs, maxBytes) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);
  try {
    const response = await fetch(url, {
      redirect: "follow",
      signal: controller.signal,
      headers: {
        "user-agent": "Mozilla/5.0 LumaMaterialCollector/1.0",
        accept: "image/avif,image/webp,image/png,image/jpeg,image/gif,*/*;q=0.8",
        ...(sourceURL ? { referer: sourceURL } : {})
      }
    });
    if (!response.ok) throw new Error(`HTTP ${response.status}`);
    const contentLength = Number(response.headers.get("content-length") || 0);
    if (contentLength > maxBytes) throw new Error("Image exceeds byte limit.");
    const buffer = Buffer.from(await response.arrayBuffer());
    if (buffer.length > maxBytes) throw new Error("Image exceeds byte limit.");
    return buffer;
  } finally {
    clearTimeout(timer);
  }
}

async function main() {
  const input = arg("input");
  const outputDir = arg("output-dir");
  const manifestPath = arg("manifest", path.join(outputDir || ".", "images_manifest.json"));
  const minWidth = Math.max(1, Number(arg("min-width", "800")));
  const minHeight = Math.max(1, Number(arg("min-height", "450")));
  const limit = Math.max(1, Number(arg("limit", "5")));
  const timeoutMs = Math.max(1000, Number(arg("timeout-ms", "15000")));
  const maxBytes = Math.max(1024, Number(arg("max-bytes", String(15 * 1024 * 1024))));
  const reviewTopic = arg("review-topic");
  const reviewClaim = arg("review-claim");
  const reviewPurpose = arg("review-purpose", "auxiliary");
  const cli = resolveCLI(arg("cli", "luma-cli"));
  const executeReviews = Boolean(reviewTopic || reviewClaim);
  if (!input || !outputDir) throw new Error("--input and --output-dir are required");
  if (executeReviews && (!reviewTopic || !reviewClaim)) {
    throw new Error("--review-topic and --review-claim must be provided together");
  }

  fs.mkdirSync(outputDir, { recursive: true });
  const candidates = signals(readJSON(input));
  const accepted = [];
  const rejected = [];
  const hashes = new Set();

  for (let index = 0; index < candidates.length && accepted.length < limit; index += 1) {
    const candidate = candidates[index];
    const imageURL = safeURL(candidate.image_url || candidate.contentUrl || candidate.thumbnail_url);
    const sourceURL = safeURL(candidate.source_url || candidate.hostPageUrl);
    if (!imageURL) {
      rejected.push({ index, reason: "missing_image_url" });
      continue;
    }
    if (/\/(?:empty|loading|placeholder|spacer)\.(?:png|jpe?g|gif|webp)(?:\?|$)/i.test(imageURL)) {
      rejected.push({ index, image_url: imageURL, reason: "placeholder_url" });
      continue;
    }

    try {
      const buffer = await download(imageURL, sourceURL, timeoutMs, maxBytes);
      if (buffer.length < 2048) throw new Error("Image file is too small.");
      const info = dimensions(buffer);
      if (!info) throw new Error("Unsupported or invalid image format.");
      if (info.width < minWidth || info.height < minHeight) {
        rejected.push({ index, image_url: imageURL, ...info, reason: "undersized" });
        continue;
      }
      const hash = crypto.createHash("sha256").update(buffer).digest("hex");
      if (hashes.has(hash)) {
        rejected.push({ index, image_url: imageURL, reason: "duplicate_content" });
        continue;
      }
      hashes.add(hash);
      const file = path.join(outputDir, `image_${String(accepted.length + 1).padStart(3, "0")}.${info.format}`);
      fs.writeFileSync(file, buffer);
      const asset = {
        asset_id: `downloaded_image_${String(accepted.length + 1).padStart(3, "0")}`,
        type: "image",
        evidence_role: "auxiliary_visual",
        status: "ready",
        path: path.resolve(file),
        image_url: imageURL,
        source_url: sourceURL,
        title: candidate.title || "",
        width: info.width,
        height: info.height,
        format: info.format,
        bytes: buffer.length,
        sha256: hash
      };
      if (executeReviews) {
        const reviewPath = `${file}.review.json`;
        try {
          const review = reviewImage(cli, path.resolve(file), path.resolve(reviewPath), reviewTopic, reviewClaim, reviewPurpose);
          asset.review_path = path.resolve(reviewPath);
          asset.review = review;
          if (review.usable !== true || review.decision !== "accept") {
            rejected.push({
              index,
              image_url: imageURL,
              path: path.resolve(file),
              reason: "vlm_review_rejected",
              review_path: path.resolve(reviewPath),
              review
            });
            continue;
          }
        } catch (error) {
          rejected.push({
            index,
            image_url: imageURL,
            path: path.resolve(file),
            reason: "vlm_review_failed",
            error: String(error.message || error)
          });
          continue;
        }
      } else {
        asset.review_status = "not_requested";
      }
      accepted.push(asset);
    } catch (error) {
      rejected.push({ index, image_url: imageURL, reason: "download_or_decode_failed", error: String(error.message || error) });
    }
  }

  const manifest = {
    schema_version: "0.1.0",
    input: path.resolve(input),
    review_context: executeReviews ? {
      topic: reviewTopic,
      claim: reviewClaim,
      purpose: reviewPurpose
    } : null,
    accepted,
    rejected,
    summary: {
      searched: candidates.length,
      accepted: accepted.length,
      rejected: rejected.length,
      vlm_review_enabled: executeReviews
    },
    generated_at: new Date().toISOString()
  };
  writeJSON(manifestPath, manifest);
  process.stdout.write(`${JSON.stringify({ manifest: path.resolve(manifestPath), ...manifest.summary }, null, 2)}\n`);
  if (!accepted.length) process.exitCode = 2;
}

main().catch((error) => {
  process.stderr.write(`${error.stack || error}\n`);
  process.exitCode = 1;
});

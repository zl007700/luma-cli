#!/usr/bin/env node
const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");

function arg(name, fallback = "") {
  const idx = process.argv.indexOf(`--${name}`);
  return idx >= 0 && idx + 1 < process.argv.length ? process.argv[idx + 1] : fallback;
}

function hasFlag(name) {
  return process.argv.includes(`--${name}`);
}

function readJSON(file) {
  return JSON.parse(fs.readFileSync(file, "utf8"));
}

function writeJSON(file, data) {
  fs.mkdirSync(path.dirname(path.resolve(file)), { recursive: true });
  fs.writeFileSync(file, JSON.stringify(data, null, 2), "utf8");
}

function copyFile(src, dst) {
  fs.mkdirSync(path.dirname(path.resolve(dst)), { recursive: true });
  fs.copyFileSync(src, dst);
}

function slug(value) {
  return String(value || "asset").replace(/[^\w\u4e00-\u9fa5-]+/g, "_").slice(0, 48);
}

function extname(file, fallback = ".png") {
  const ext = path.extname(String(file || "")).toLowerCase();
  return ext || fallback;
}

function makeAssetID(prefix, idx) {
  return `${prefix}_${String(idx + 1).padStart(3, "0")}`;
}

function fileExists(file) {
  try {
    return fs.statSync(file).isFile();
  } catch {
    return false;
  }
}

function listSignals(payload) {
  if (!payload || typeof payload !== "object") return [];
  if (Array.isArray(payload.raw_signals)) return payload.raw_signals;
  if (payload.result && Array.isArray(payload.result.raw_signals)) return payload.result.raw_signals;
  if (payload.data && payload.data.result && Array.isArray(payload.data.result.raw_signals)) {
    return payload.data.result.raw_signals;
  }
  return [];
}

function safeURL(value) {
  try {
    const parsed = new URL(String(value || ""));
    return ["http:", "https:"].includes(parsed.protocol) ? parsed : null;
  } catch {
    return null;
  }
}

function normalizeQuery(value) {
  return String(value || "").toLowerCase().replace(/\s+/g, " ").trim();
}

function queryTokens(value) {
  const text = normalizeQuery(value);
  const tokens = new Set(text.match(/[a-z0-9][a-z0-9._-]+/g) || []);
  for (const sequence of text.match(/[\u4e00-\u9fff]+/g) || []) {
    if (sequence.length <= 2) {
      tokens.add(sequence);
      continue;
    }
    for (let idx = 0; idx < sequence.length - 1; idx += 1) {
      tokens.add(sequence.slice(idx, idx + 2));
    }
  }
  for (const ignored of ["official", "report", "case", "study", "www", "https"]) tokens.delete(ignored);
  return [...tokens];
}

function relevanceScore(item, query) {
  const haystack = normalizeQuery(`${item.title || ""} ${item.summary || ""} ${item.url || ""}`);
  const tokens = queryTokens(query);
  if (!haystack || !tokens.length) return 0;
  const matched = tokens.filter((token) => haystack.includes(token));
  return matched.length;
}

function hostMatchesDomain(host, domain) {
  const normalized = String(domain || "")
    .trim()
    .toLowerCase()
    .replace(/^https?:\/\//, "")
    .replace(/\/.*$/, "");
  return Boolean(normalized && (host === normalized || host.endsWith(`.${normalized}`)));
}

function assessWebSource(item, task = {}) {
  const parsed = safeURL(item.url || item.source_url);
  if (!parsed) {
    return { source_tier: "unknown", source_kind: "invalid_url", source_host: "", source_reason: "invalid_url" };
  }
  const host = parsed.hostname.toLowerCase().replace(/^www\./, "");
  const pathname = parsed.pathname.toLowerCase();
  const preferredDomains = task.source_policy?.preferred_domains || task.preferred_domains || [];
  if (preferredDomains.some((domain) => hostMatchesDomain(host, domain))) {
    return {
      source_tier: "primary",
      source_kind: "preferred_domain",
      source_host: host,
      source_reason: "matches_task_preferred_domain"
    };
  }
  if (host.endsWith(".gov") || host.endsWith(".gov.cn") || host.endsWith(".edu") || host.endsWith(".edu.cn")) {
    return {
      source_tier: "institutional",
      source_kind: "government_or_education",
      source_host: host,
      source_reason: "government_or_education_domain"
    };
  }
  if (host === "arxiv.org" || host === "doi.org") {
    return {
      source_tier: "institutional",
      source_kind: "research_repository",
      source_host: host,
      source_reason: "research_repository"
    };
  }
  if (/(douyin|instagram|facebook|threads|twitter|tiktok)\./.test(host) || host === "x.com") {
    return {
      source_tier: "social",
      source_kind: "social_platform",
      source_host: host,
      source_reason: "social_platform"
    };
  }

  const query = normalizeQuery(task.query || "");
  const identityLabels = host
    .split(".")
    .filter((part) => part.length >= 4 && !["www", "docs", "blog", "news", "cloud"].includes(part));
  if (identityLabels.some((label) => query.includes(label))) {
    return {
      source_tier: "primary",
      source_kind: pathname.includes("/docs") ? "official_docs" : "official_site",
      source_host: host,
      source_reason: "domain_identity_matches_query"
    };
  }

  return {
    source_tier: "secondary",
    source_kind: pathname.includes("/news") ? "editorial_or_media" : "third_party_web",
    source_host: host,
    source_reason: "not_verified_as_primary"
  };
}

function sourceScore(item, task) {
  const query = task.query || "";
  const parsed = safeURL(item.url || item.source_url);
  if (!parsed) return -100;
  const host = parsed.hostname.toLowerCase();
  const pathname = parsed.pathname.toLowerCase();
  const assessment = assessWebSource(item, task);
  let score = 0;
  if (assessment.source_tier === "primary") score += 12;
  if (assessment.source_tier === "institutional") score += 10;
  if (assessment.source_tier === "secondary") score += 1;
  if (assessment.source_tier === "social") score -= 8;
  if (host.endsWith(".gov") || host.endsWith(".edu")) score += 8;
  if (/(^|\.)docs?\./.test(host) || pathname.includes("/docs") || pathname.includes("/documentation")) score += 7;
  if (pathname.includes("/blog") || pathname.includes("/news") || pathname.includes("/research")) score += 4;
  if (host.includes("github.com")) score += 3;
  if (["social", "social_account"].includes(String(item.source || "").toLowerCase())) score -= 5;
  if (/(instagram|facebook|threads|twitter|tiktok)\./.test(host) || host === "x.com") score -= 8;
  if (String(item.title || "").toLowerCase().includes("official")) score += 2;
  if (String(item.summary || "").length > 80) score += 1;
  score += relevanceScore(item, query) * 2;
  return score;
}

function rankWebSignals(signals, task) {
  const queries = taskQueries(task);
  const primaryQuery = task.query || queries[0] || "";
  return signals
    .filter((item) => safeURL(item.url))
    .map((item) => {
      const assessment = assessWebSource(item, task);
      const queryScores = queries.length
        ? queries.map((query) => ({ query, score: relevanceScore(item, query) }))
        : [{ query: primaryQuery, score: relevanceScore(item, primaryQuery) }];
      const best = queryScores.sort((a, b) => b.score - a.score)[0] || { query: primaryQuery, score: 0 };
      return {
        ...item,
        ...assessment,
        relevance_score: best.score,
        matched_query: best.query,
        source_score: sourceScore(item, task)
      };
    })
    .filter((item) => item.relevance_score > 0)
    .sort((a, b) => b.source_score - a.source_score)
    .slice(0, 5);
}

function payloadQueries(payload) {
  if (!payload || typeof payload !== "object") return [];
  if (Array.isArray(payload.queries)) return payload.queries;
  if (payload.result && Array.isArray(payload.result.queries)) return payload.result.queries;
  if (payload.data && payload.data.result && Array.isArray(payload.data.result.queries)) {
    return payload.data.result.queries;
  }
  return [];
}

function searchOutputMatchesTask(payload, task) {
  const queries = payloadQueries(payload).map(normalizeQuery).filter(Boolean);
  if (!queries.length) return true;
  return queries.includes(normalizeQuery(task.query || ""));
}

function rankImageSignals(signals, acceptance = {}) {
  const minWidth = Number(acceptance.min_width || 0);
  const minHeight = Number(acceptance.min_height || 0);
  const seen = new Set();
  return signals
    .filter((item) => {
      const imageURL = String(item.image_url || "");
      if (!safeURL(imageURL) || seen.has(imageURL)) return false;
      if (/\/empty\.(png|jpg|jpeg|webp)(\?|$)/i.test(imageURL)) return false;
      seen.add(imageURL);
      const width = Number(item.width || 0);
      const height = Number(item.height || 0);
      if (minWidth && width < minWidth) return false;
      if (minHeight && height < minHeight) return false;
      if (acceptance.require_source_url && !safeURL(item.source_url)) return false;
      return true;
    })
    .map((item) => {
      const width = Number(item.width || 0);
      const height = Number(item.height || 0);
      let score = 0;
      if (width >= 1200) score += 3;
      if (height >= 675) score += 2;
      if (width > height) score += 2;
      if (safeURL(item.source_url)) score += 1;
      return { ...item, visual_score: score };
    })
    .sort((a, b) => b.visual_score - a.visual_score)
    .slice(0, 5);
}

function resolveCLI(value) {
  if (value && value !== "luma-cli") return value;
  if (process.platform === "win32" && process.env.APPDATA) {
    const installedExe = path.join(
      process.env.APPDATA,
      "npm",
      "node_modules",
      "@lumageo",
      "luma-cli",
      "bin",
      "luma-cli.exe"
    );
    if (fileExists(installedExe)) return installedExe;
  }
  return value || "luma-cli";
}

function runCLI(cli, args) {
  const result = spawnSync(cli, args, {
    cwd: process.cwd(),
    encoding: "utf8",
    windowsHide: true
  });
  if (result.error) throw result.error;
  if (result.status !== 0) {
    throw new Error(String(result.stderr || result.stdout || `command exited ${result.status}`).trim());
  }
}

function runNodeScript(script, args, acceptedExitCodes = [0]) {
  const result = spawnSync(process.execPath, [script, ...args], {
    cwd: process.cwd(),
    encoding: "utf8",
    windowsHide: true
  });
  if (result.error) throw result.error;
  if (!acceptedExitCodes.includes(result.status)) {
    throw new Error(String(result.stderr || result.stdout || `script exited ${result.status}`).trim());
  }
  return result;
}

function claimByID(plan, claimID) {
  return (plan.core_claims || []).find((claim) => claim.claim_id === claimID) || null;
}

function keywordHints(claim, candidate) {
  const values = [
    candidate?.summary || "",
    claim?.claim || "",
    ...(claim?.existing_evidence || []),
    ...(candidate?.summary ? [] : [candidate?.title || ""])
  ];
  const ignored = new Set(["official", "report", "case", "study"]);
  const tokens = [];
  for (const value of values) {
    const clauses = String(value || "").split(/[，。；！？,.!?;：:、“”"'（）()]+/);
    for (const clause of clauses) {
      const clean = clause.replace(/\s+/g, " ").trim();
      const sequences = clean.match(/[\u4e00-\u9fffA-Za-z0-9][\u4e00-\u9fffA-Za-z0-9 ]{3,15}/g) || [];
      for (const sequence of sequences) {
      const token = sequence.trim();
      if (!ignored.has(token.toLowerCase()) && !tokens.includes(token)) tokens.push(token);
      if (tokens.length >= 6) return tokens;
      }
    }
  }
  return tokens;
}

function normalizedText(value) {
  return String(value || "").toLowerCase().replace(/[^\u4e00-\u9fffa-z0-9]+/g, "");
}

function captureMatchesCandidate(capture, candidate) {
  const matched = normalizedText(capture?.matched_text);
  const summary = normalizedText(candidate?.summary);
  if (!summary || !matched) return true;
  const chunks = [];
  for (let index = 0; index + 5 <= summary.length; index += 4) {
    chunks.push(summary.slice(index, index + 8));
  }
  return chunks.some((chunk) => chunk.length >= 5 && matched.includes(chunk));
}

function capturePaths(outDir, task, candidateIndex) {
  const base = `${task.task_id}_candidate_${String(candidateIndex + 1).padStart(2, "0")}`;
  return {
    image: path.join(outDir, `${base}.png`),
    capture: path.join(outDir, `${base}.capture.json`),
    review: path.join(outDir, `${base}.review.json`)
  };
}

function reusableCapture(paths, candidate, plan, claim) {
  if (!fileExists(paths.image) || !fileExists(paths.capture) || !fileExists(paths.review)) return null;
  try {
    const capture = readJSON(paths.capture);
    const review = readJSON(paths.review);
    if (
      capture.status !== "ready" ||
      capture.source_url !== candidate.url ||
      !captureMatchesCandidate(capture, candidate) ||
      review.topic !== plan.topic_title ||
      review.claim !== claim.claim ||
      review.purpose !== "evidence"
    ) {
      return null;
    }
    return { capture, review };
  } catch {
    return null;
  }
}

function executeCaptureCandidate(plan, task, candidate, candidateIndex, outDir, cli) {
  const claim = claimByID(plan, (task.claim_ids || [])[0]);
  if (!claim) throw new Error(`Claim not found for ${task.task_id}`);
  const paths = capturePaths(outDir, task, candidateIndex);
  let result = reusableCapture(paths, candidate, plan, claim);
  if (!result) {
    const captureScript = path.join(__dirname, "capture_webpage.js");
    const keywords = keywordHints(claim, candidate);
    runNodeScript(captureScript, [
      "--url", candidate.url,
      "--mode", keywords.length ? "keyword" : "first-screen",
      ...(keywords.length ? ["--keywords", keywords.join("|")] : []),
      "--output", paths.image,
      "--result", paths.capture
    ]);
    runCLI(cli, [
      "material", "review", path.resolve(paths.image),
      "--topic", plan.topic_title,
      "--claim", claim.claim,
      "--purpose", "evidence",
      "--output", path.resolve(paths.review)
    ]);
    result = { capture: readJSON(paths.capture), review: readJSON(paths.review) };
  }
  if (result.capture.status !== "ready") {
    return { accepted: false, reason: result.capture.error || "capture_failed", ...result };
  }
  if (result.review.usable !== true || result.review.decision !== "accept") {
    return { accepted: false, reason: result.review.reject_reason || "review_rejected", ...result };
  }
  const sourceAssessment = assessWebSource(candidate, task);
  return {
    accepted: true,
    capture: result.capture,
    review: result.review,
    asset: {
      asset_id: `capture_${task.task_id}_${String(candidateIndex + 1).padStart(2, "0")}`,
      task_id: `capture_${task.task_id}_${String(candidateIndex + 1).padStart(2, "0")}`,
      type: "website_screenshot",
      evidence_role: "fact_evidence",
      status: "ready",
      path: path.resolve(paths.image),
      source_url: candidate.url,
      source_title: candidate.title || "",
      ...sourceAssessment,
      claim_ids: task.claim_ids || [],
      review_path: path.resolve(paths.review),
      review: result.review,
      capture: result.capture
    }
  };
}

function imageManifestMatches(manifest, topic, claim, purpose) {
  const context = manifest?.review_context;
  return Boolean(
    context &&
    context.topic === topic &&
    context.claim === claim &&
    context.purpose === purpose
  );
}

function executeImageCollection(plan, task, resultPath, outDir, cli) {
  const imageDir = path.join(outDir, `${task.task_id}_images`);
  const manifestPath = path.join(imageDir, "images_manifest.json");
  const topic = plan.topic_title || "";
  const claim = plan.core_thesis || plan.topic_title || "";
  const purpose = "auxiliary";
  let manifest = null;
  if (fileExists(manifestPath)) {
    try {
      const existing = readJSON(manifestPath);
      if (imageManifestMatches(existing, topic, claim, purpose)) manifest = existing;
    } catch {
      manifest = null;
    }
  }
  if (!manifest) {
    const script = path.join(__dirname, "collect_images.js");
    runNodeScript(script, [
      "--input", path.resolve(resultPath),
      "--output-dir", path.resolve(imageDir),
      "--manifest", path.resolve(manifestPath),
      "--min-width", String(task.acceptance?.min_width || 800),
      "--min-height", String(task.acceptance?.min_height || 450),
      "--limit", "3",
      "--review-topic", topic,
      "--review-claim", claim,
      "--review-purpose", purpose,
      "--cli", cli
    ], [0, 2]);
    manifest = readJSON(manifestPath);
  }
  return { manifest, manifestPath };
}

function signalCardAsset(task, idx, outDir) {
  const id = makeAssetID("asset_signal", idx);
  const file = path.join(outDir, `${id}.json`);
  const spec = {
    component: "SignalCard",
    title: task.input?.title || "",
    author_name: task.input?.author_name || "",
    source: task.input?.source || "",
    url: task.input?.url || "",
    stats: { likes: task.input?.likes || 0 },
    disclaimer: "Market signal only; not objective proof.",
    purpose: task.purpose || ""
  };
  writeJSON(file, spec);
  return {
    asset_id: id,
    type: "signal_card",
    evidence_role: "market_signal",
    source: "generated",
    task_id: task.task_id,
    path: path.resolve(file),
    purpose: task.purpose || "",
    claim_ids: task.claim_ids || [],
    status: "ready",
    component_spec: spec
  };
}

function generatedVisualAsset(plan, visual, idx, outDir) {
  const id = makeAssetID("asset_visual", idx);
  const file = path.join(outDir, `${id}_${slug(visual.visual_component)}.json`);
  const spec = {
    component: visual.visual_component || "ChapterCard",
    section: visual.section || "",
    title: visual.claim || plan.topic_title || "",
    fallback: visual.fallback || "chapter_text_card",
    topic_title: plan.topic_title || ""
  };
  writeJSON(file, spec);
  return {
    asset_id: id,
    type: "generated_component",
    evidence_role: "visual_explanation",
    source: "generated",
    task_id: visual.segment_id,
    path: path.resolve(file),
    purpose: `visual support for ${visual.section || visual.segment_id}`,
    claim_ids: [],
    status: "ready",
    component_spec: spec
  };
}

function searchOutputPath(resultsDir, task) {
  const prefix = task.action === "image_search" ? "image_search" : "websearch";
  return path.join(resultsDir, `${prefix}_${task.task_id}.json`);
}

function taskQueries(task) {
  const queries = Array.isArray(task.queries) && task.queries.length ? task.queries : [task.query];
  return [...new Set(queries.map((query) => String(query || "").trim()).filter(Boolean))];
}

function searchOutputPaths(resultsDir, task) {
  const queries = taskQueries(task);
  if (queries.length <= 1) return [{ query: queries[0] || task.query || "", path: searchOutputPath(resultsDir, task) }];
  const prefix = task.action === "image_search" ? "image_search" : "websearch";
  return queries.map((query, index) => ({
    query,
    path: path.join(resultsDir, `${prefix}_${task.task_id}_q${String(index + 1).padStart(2, "0")}.json`)
  }));
}

function executeSearchTask(task, outputPath, cli, query = task.query) {
  if (task.action === "websearch") {
    runCLI(cli, [
      "content", "search", "websearch",
      "--query", query,
      "--date-range", task.date_range || "30d",
      "--num", String(task.max_results || 5),
      "--output", outputPath
    ]);
    return;
  }
  if (task.action === "image_search") {
    runCLI(cli, [
      "content", "search", "image",
      "--query", query,
      "--count", String(task.count || 8),
      "--output", outputPath
    ]);
  }
}

function mergedSearchPayload(payloads, paths) {
  const seen = new Set();
  const raw = [];
  for (const payload of payloads) {
    for (const item of listSignals(payload)) {
      const key = item.url || item.image_url || item.source_url || JSON.stringify(item).slice(0, 200);
      if (seen.has(key)) continue;
      seen.add(key);
      raw.push(item);
    }
  }
  return {
    raw_signals: raw,
    queries: paths.map((item) => item.query).filter(Boolean),
    source_files: paths.map((item) => path.resolve(item.path))
  };
}

function webCandidateAsset(task, ranked, idx, outputPath) {
  return {
    asset_id: makeAssetID("asset_web_candidates", idx),
    type: "web_evidence_candidates",
    evidence_role: "fact_evidence_candidate",
    source: "websearch",
    task_id: task.task_id,
    path: Array.isArray(outputPath) ? outputPath.map((item) => path.resolve(item)) : path.resolve(outputPath),
    claim_ids: task.claim_ids || [],
    status: ranked.length ? "candidate_found" : "empty",
    candidates: ranked.map((item) => ({
      title: item.title || "",
      summary: item.summary || "",
      url: item.url || "",
      published_at: item.published_at || "",
      source_score: item.source_score,
      source_tier: item.source_tier,
      source_kind: item.source_kind,
      source_host: item.source_host,
      source_reason: item.source_reason,
      matched_query: item.matched_query || ""
    }))
  };
}

function captureTask(task, candidate, idx) {
  return {
    task_id: `capture_${task.task_id}_${String(idx + 1).padStart(2, "0")}`,
    action: "capture_url",
    evidence_role: "fact_evidence",
    priority: task.priority || "medium",
    purpose: task.purpose || "Capture readable evidence for the claim.",
    url: candidate.url,
    title: candidate.title || "",
    claim_ids: task.claim_ids || [],
    source_task_id: task.task_id,
    capture_mode: "evidence_region",
    acceptance: task.acceptance || {
      must_match_claim_keywords: true,
      must_have_readable_page: true,
      capture_smallest_proving_region: true
    },
    reason: "Browser verification and screenshot are required before this claim is covered."
  };
}

function imageCandidateAssets(task, ranked, startIndex, outputPath) {
  return ranked.map((item, idx) => ({
    asset_id: makeAssetID("asset_image_candidate", startIndex + idx),
    type: "image_candidate",
    evidence_role: "auxiliary_visual",
    source: "image_search",
    task_id: task.task_id,
    path: path.resolve(outputPath),
    claim_ids: task.claim_ids || [],
    status: "candidate",
    title: item.title || "",
    image_url: item.image_url || "",
    thumbnail_url: item.thumbnail_url || "",
    source_url: item.source_url || "",
    source_name: item.source_name || "",
    width: item.width || null,
    height: item.height || null,
    visual_score: item.visual_score
  }));
}

function setCoverage(coverage, claimIDs, status) {
  const rank = {
    none: 0,
    generated_only: 1,
    fallback_only: 2,
    market_signal_only: 3,
    evidence_candidate_found: 4,
    covered_by_secondary_evidence: 5,
    covered_by_fact_evidence: 6
  };
  for (const claimID of claimIDs || []) {
    const current = coverage[claimID] || "none";
    if ((rank[status] || 0) > (rank[current] || 0)) coverage[claimID] = status;
  }
}

function coverageStatusForAsset(item) {
  if (["primary", "institutional"].includes(item.source_tier)) return "covered_by_fact_evidence";
  return "covered_by_secondary_evidence";
}

function mergeCapturedManifest(manifestPath, assets, coverage) {
  const completedTaskIDs = new Set();
  if (!manifestPath || !fileExists(manifestPath)) return completedTaskIDs;
  const manifest = readJSON(manifestPath);
  const captured = Array.isArray(manifest) ? manifest : manifest.assets || [];
  for (const item of captured) {
    assets.push(item);
    if (item.asset_id) completedTaskIDs.add(String(item.asset_id));
    if (item.task_id) completedTaskIDs.add(String(item.task_id));
    if (
      item.status === "ready" &&
      item.evidence_role === "fact_evidence" &&
      (item.path || item.local_path) &&
      item.source_url
    ) {
      setCoverage(coverage, item.claim_ids || [], coverageStatusForAsset(item));
    }
  }
  return completedTaskIDs;
}

function deliveryKind(asset) {
  if (asset.type === "website_screenshot") return "evidence_screenshot";
  if (asset.type === "image") return "auxiliary_image";
  if (asset.type === "signal_card") return "market_signal";
  if (asset.type === "generated_component") return "generated_component_spec";
  return "";
}

function buildDeliverables(plan, assets, deliveryDir) {
  const items = [];
  let visualIndex = 0;
  for (const asset of assets) {
    const kind = deliveryKind(asset);
    if (!kind || asset.status !== "ready") continue;
    const deliverable = {
      asset_id: asset.asset_id,
      kind,
      source_type: asset.type,
      evidence_role: asset.evidence_role || "",
      claim_ids: asset.claim_ids || [],
      purpose: asset.purpose || "",
      source_url: asset.source_url || "",
      source_title: asset.source_title || asset.title || "",
      source_tier: asset.source_tier || "",
      review_decision: asset.review?.decision || "",
      review_issues: asset.review?.issues || [],
      review_note: asset.review?.review_note || "",
      usable: asset.review ? asset.review.usable === true : true
    };
    if (["website_screenshot", "image"].includes(asset.type) && asset.path && fileExists(asset.path)) {
      const name = `${String(visualIndex + 1).padStart(2, "0")}_${asset.asset_id}${extname(asset.path)}`;
      const dst = path.join(deliveryDir, name);
      copyFile(asset.path, dst);
      deliverable.path = path.resolve(dst);
      deliverable.original_path = path.resolve(asset.path);
      visualIndex += 1;
    } else if (asset.path) {
      deliverable.path = path.resolve(asset.path);
    }
    items.push(deliverable);
  }
  const manifest = {
    schema_version: "0.1.0",
    topic_id: plan.topic_id,
    topic_title: plan.topic_title,
    summary: {
      total: items.length,
      visual_files: items.filter((item) => item.path && ["evidence_screenshot", "auxiliary_image"].includes(item.kind)).length,
      evidence_screenshots: items.filter((item) => item.kind === "evidence_screenshot").length,
      auxiliary_images: items.filter((item) => item.kind === "auxiliary_image").length,
      market_signals: items.filter((item) => item.kind === "market_signal").length,
      generated_component_specs: items.filter((item) => item.kind === "generated_component_spec").length
    },
    items,
    generated_at: new Date().toISOString()
  };
  const manifestPath = path.join(deliveryDir, "deliverables_manifest.json");
  writeJSON(manifestPath, manifest);
  return {
    directory: path.resolve(deliveryDir),
    manifest_path: path.resolve(manifestPath),
    ...manifest.summary,
    items
  };
}

function main() {
  const planPath = arg("plan") || arg("input");
  const output = arg("output", "05_material_assets.json");
  if (!planPath) throw new Error("--plan is required");

  const plan = readJSON(planPath);
  const outputDir = path.dirname(path.resolve(output));
  const outDir = arg("assets-dir", path.join(outputDir, "materials", plan.topic_id || "topic"));
  const deliveryDir = arg("deliverables-dir", path.join(outputDir, "final_assets"));
  const resultsDir = arg("results-dir", outDir);
  const executeCollection = hasFlag("execute-collection");
  const executeSearches = hasFlag("execute-searches") || executeCollection;
  const executeCaptures = hasFlag("execute-captures") || executeCollection;
  const executeImages = hasFlag("execute-images") || executeCollection;
  const cli = resolveCLI(arg("cli", "luma-cli"));
  fs.mkdirSync(outDir, { recursive: true });
  fs.mkdirSync(resultsDir, { recursive: true });

  const assets = [];
  const pendingTasks = [];
  const failedTasks = [];
  const coverage = {};

  for (const claim of plan.core_claims || []) {
    coverage[claim.claim_id] = claim.need_evidence ? "fallback_only" : "generated_only";
  }

  let webAssetIndex = 0;
  let imageAssetIndex = 0;
  for (const [idx, task] of (plan.collection_tasks || []).entries()) {
    if (task.action === "make_signal_card") {
      assets.push(signalCardAsset(task, idx, outDir));
      setCoverage(coverage, task.claim_ids, "market_signal_only");
      continue;
    }

    if (!["websearch", "image_search"].includes(task.action)) {
      pendingTasks.push({ ...task, reason: "Requires browser or agent execution." });
      continue;
    }

    const resultPaths = searchOutputPaths(resultsDir, task);
    const payloads = [];
    const missingPaths = [];
    for (const item of resultPaths) {
      let payload = null;
      let stale = false;
      if (fileExists(item.path)) {
        try {
          payload = readJSON(item.path);
          stale = !searchOutputMatchesTask(payload, { ...task, query: item.query });
        } catch {
          stale = true;
        }
      }
      if ((!payload || stale) && executeSearches) {
        try {
          executeSearchTask(task, item.path, cli, item.query);
          payload = readJSON(item.path);
          stale = false;
        } catch (error) {
          failedTasks.push({
            ...task,
            query: item.query,
            output_path: path.resolve(item.path),
            error: String(error.message || error),
            retryable: true
          });
          continue;
        }
      }
      if (!payload || stale) {
        missingPaths.push({ ...item, stale });
        continue;
      }
      payloads.push(payload);
    }

    if (!payloads.length) {
      pendingTasks.push({
        ...task,
        expected_output: resultPaths.map((item) => path.resolve(item.path)),
        reason: missingPaths.some((item) => item.stale)
          ? "Existing search output belongs to a different query. Re-run with --execute-searches."
          : "Search results are missing. Re-run collector with --execute-searches."
      });
      continue;
    }

    const payload = mergedSearchPayload(payloads, resultPaths);

    const signals = listSignals(payload);
    if (task.action === "websearch") {
      const ranked = rankWebSignals(signals, task);
      assets.push(webCandidateAsset(task, ranked, webAssetIndex++, resultPaths.map((item) => item.path)));
      if (ranked.length) {
        setCoverage(coverage, task.claim_ids, "evidence_candidate_found");
        if (executeCaptures) {
          const attempts = [];
          const accepted = [];
          const maxAccepted = Math.max(1, Number(task.max_accepted_captures || 1));
          const captureCandidates = ranked
            .filter((candidate) => candidate.source_score >= 0)
            .slice(0, Math.max(1, Number(task.max_capture_candidates || 3)));
          for (const [candidateIndex, candidate] of captureCandidates.entries()) {
            try {
              const result = executeCaptureCandidate(plan, task, candidate, candidateIndex, outDir, cli);
              attempts.push({
                url: candidate.url,
                title: candidate.title || "",
                accepted: result.accepted,
                reason: result.reason || "",
                review_path: result.asset?.review_path || path.resolve(capturePaths(outDir, task, candidateIndex).review)
              });
              if (result.accepted) {
                accepted.push(result);
                if (accepted.length >= maxAccepted) break;
              }
            } catch (error) {
              attempts.push({
                url: candidate.url,
                title: candidate.title || "",
                accepted: false,
                reason: String(error.message || error)
              });
            }
          }
          if (accepted.length) {
            for (const item of accepted) {
              assets.push(item.asset);
              setCoverage(coverage, task.claim_ids, coverageStatusForAsset(item.asset));
            }
          } else {
            failedTasks.push({
              ...task,
              action: "capture_and_review",
              error: "No web candidate passed capture and VLM evidence review.",
              attempts,
              retryable: true
            });
          }
        } else {
          pendingTasks.push(captureTask(task, ranked[0], 0));
        }
      } else {
        failedTasks.push({ ...task, error: "No usable web evidence candidate found.", retryable: true });
      }
      continue;
    }

    if (executeImages) {
      try {
        const mergedImagePath = path.join(resultsDir, `image_search_${task.task_id}_merged.json`);
        writeJSON(mergedImagePath, payload);
        const { manifest, manifestPath } = executeImageCollection(plan, task, mergedImagePath, outDir, cli);
        for (const item of manifest.accepted || []) {
          assets.push({
            ...item,
            task_id: task.task_id,
            source: "image_search",
            manifest_path: path.resolve(manifestPath)
          });
        }
        if (!(manifest.accepted || []).length) {
          failedTasks.push({
            ...task,
            error: "No downloaded image passed local checks and VLM review.",
            manifest_path: path.resolve(manifestPath),
            rejected_count: (manifest.rejected || []).length,
            retryable: true
          });
        }
      } catch (error) {
        failedTasks.push({
          ...task,
          error: String(error.message || error),
          retryable: true
        });
      }
    } else {
      const ranked = rankImageSignals(signals, task.acceptance || {});
      const mergedImagePath = path.join(resultsDir, `image_search_${task.task_id}_merged.json`);
      writeJSON(mergedImagePath, payload);
      const candidates = imageCandidateAssets(task, ranked, imageAssetIndex, mergedImagePath);
      imageAssetIndex += candidates.length;
      assets.push(...candidates);
      if (!ranked.length) {
        failedTasks.push({ ...task, error: "No image candidate passed metadata checks.", retryable: true });
      }
    }
  }

  for (const [idx, visual] of (plan.visual_plan || []).entries()) {
    assets.push(generatedVisualAsset(plan, visual, idx, outDir));
  }

  const completedCaptureTasks = mergeCapturedManifest(arg("captured-manifest"), assets, coverage);
  const remainingPendingTasks = pendingTasks.filter((task) => !completedCaptureTasks.has(String(task.task_id || "")));
  const deliverables = buildDeliverables(plan, assets, deliveryDir);

  const result = {
    schema_version: "0.2.0",
    topic_id: plan.topic_id,
    topic_title: plan.topic_title,
    plan_path: path.resolve(planPath),
    execution: {
      execute_searches: executeSearches,
      execute_captures: executeCaptures,
      execute_images: executeImages,
      execute_collection: executeCollection,
      cli,
      results_dir: path.resolve(resultsDir),
      precheck_reuses_existing_results: true
    },
    assets,
    deliverables,
    pending_tasks: remainingPendingTasks,
    failed_tasks: failedTasks,
    coverage,
    coverage_policy: {
      market_signal_only: "Shows attention or discussion; does not prove the claim.",
      evidence_candidate_found: "A source candidate exists, but browser verification/capture is pending.",
      covered_by_secondary_evidence: "A readable third-party source supports the claim; qualify attribution and avoid treating it as primary proof.",
      covered_by_fact_evidence: "A readable primary or institutional source is attached to the claim."
    },
    generated_at: new Date().toISOString()
  };
  writeJSON(output, result);
  console.log(JSON.stringify({
    output: path.resolve(output),
    assets: assets.length,
    pending_tasks: remainingPendingTasks.length,
    failed_tasks: failedTasks.length,
    coverage
  }, null, 2));
}

module.exports = {
  assessWebSource,
  coverageStatusForAsset,
  rankWebSignals
};

if (require.main === module) main();

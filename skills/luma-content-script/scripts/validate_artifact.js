#!/usr/bin/env node
const fs = require("fs");

function arg(name) {
  const index = process.argv.indexOf(`--${name}`);
  return index >= 0 ? process.argv[index + 1] : "";
}

function fail(message) {
  console.error(`validation failed: ${message}`);
  process.exit(1);
}

function readJSON(file) {
  if (!file) fail("--input is required");
  return JSON.parse(fs.readFileSync(file, "utf8").replace(/^\uFEFF/, ""));
}

function value(payload, key) {
  return payload?.input?.[key] ?? payload?.[key];
}

function validateDiscovery(payload) {
  const signals = Array.isArray(payload.raw_signals) ? payload.raw_signals : [];
  if (signals.length < 15) fail(`raw_signals has ${signals.length}; need at least 15`);
  if (!(Number(payload.counts?.social_raw) > 0)) fail("counts.social_raw must be greater than zero");
  if (!(Number(payload.counts?.web_raw) > 0)) fail("counts.web_raw must be greater than zero");
  return { unique_signals: signals.length, social_raw: payload.counts.social_raw, web_raw: payload.counts.web_raw };
}

function validatePlan(payload) {
  const plan = value(payload, "longform_plan") || payload.longform_plan || payload;
  for (const field of ["topic", "public_entry", "topic_reveal", "viewer_promise", "core_thesis", "stance", "audience_filter_turn"]) {
    if (!String(plan[field] || "").trim()) fail(`longform_plan.${field} is required`);
  }
  if (!Array.isArray(plan.outline) || plan.outline.length < 2) fail("longform_plan.outline needs at least two sections");
  const roles = new Set(["none", "example", "analogy", "supporting_evidence", "direct_proof"]);
  plan.outline.forEach((item, index) => {
    if (!String(item.section || "").trim() || !String(item.claim || "").trim()) fail(`outline[${index}] needs section and claim`);
    if (index < plan.outline.length - 1 && !String(item.bridge_to_next || "").trim()) fail(`outline[${index}].bridge_to_next is required`);
    if (!roles.has(item.evidence_role)) fail(`outline[${index}].evidence_role is invalid`);
  });
  return { sections: plan.outline.length };
}

function validateTopicSelection(payload) {
  const cards = Array.isArray(payload.topic_cards) ? payload.topic_cards : [];
  const selected = cards.filter((card) => card.status === "selected");
  if (selected.length !== 1) fail(`topic_cards must contain exactly one selected card; got ${selected.length}`);
  const card = selected[0];
  for (const field of ["topic_id", "title", "angle", "public_entry", "core_opinion", "audience_value", "why_selected"]) {
    if (!String(card[field] || "").trim()) fail(`selected topic.${field} is required`);
  }
  const signals = Array.isArray(card.evidence_signals) ? card.evidence_signals : [];
  if (signals.length < 3) fail("selected topic needs at least three evidence_signals");
  const sources = new Set(signals.map((signal) => String(signal.source || "").toLowerCase()));
  const hasSocial = [...sources].some((source) => source.includes("social") || source.includes("douyin"));
  const hasWeb = [...sources].some((source) => source.includes("web"));
  if (!hasSocial || !hasWeb) fail("evidence_signals must span social/Douyin and web sources");
  if (!Array.isArray(card.rejected_alternatives) || card.rejected_alternatives.length < 2) {
    fail("selected topic needs at least two rejected_alternatives");
  }
  return { topic_id: card.topic_id, evidence_signals: signals.length, rejected_alternatives: card.rejected_alternatives.length };
}

function validateMaterials(payload) {
  const assets = Array.isArray(payload.assets) ? payload.assets : [];
  const deliverables = Array.isArray(payload.deliverables?.items) ? payload.deliverables.items : [];
  if (!assets.length) fail("assets is empty");
  if (!deliverables.length) fail("deliverables.items is empty");
  const usable = deliverables.filter((item) => item.usable !== false);
  if (!usable.length) fail("no usable deliverables");
  return { assets: assets.length, deliverables: deliverables.length, usable: usable.length };
}

function validateScript(payload) {
  const script = value(payload, "script") || payload.script || payload;
  for (const field of ["topic_id", "title", "hook", "topic_reveal", "viewer_promise", "full_script"]) {
    if (!String(script[field] || "").trim()) fail(`script.${field} is required`);
  }
  if (!Array.isArray(script.sections) || script.sections.length < 2) fail("script.sections needs at least two sections");
  if (!script.full_script.includes(script.topic_reveal)) fail("full_script must contain topic_reveal");
  if (!script.full_script.includes(script.viewer_promise)) fail("full_script must contain viewer_promise");
  script.sections.forEach((item, index) => {
    if (!String(item.spoken_text || "").trim()) fail(`sections[${index}].spoken_text is required`);
    if (index < script.sections.length - 1 && !String(item.bridge_to_next || "").trim()) fail(`sections[${index}].bridge_to_next is required`);
    if (!Array.isArray(item.material_asset_ids)) fail(`sections[${index}].material_asset_ids must be an array`);
  });
  return { sections: script.sections.length, full_script_chars: script.full_script.length };
}

const validators = {
  discovery: validateDiscovery,
  "topic-selection": validateTopicSelection,
  plan: validatePlan,
  materials: validateMaterials,
  script: validateScript
};

const type = arg("type");
if (!validators[type]) fail("--type must be discovery, topic-selection, plan, materials, or script");
const result = validators[type](readJSON(arg("input")));
console.log(JSON.stringify({ ok: true, type, ...result }, null, 2));

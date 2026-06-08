#!/usr/bin/env node
const fs = require("fs");
const path = require("path");

function arg(name, fallback = "") {
  const index = process.argv.indexOf(`--${name}`);
  return index >= 0 && index + 1 < process.argv.length ? process.argv[index + 1] : fallback;
}

function readJSON(file, fallback = null) {
  if (!file) return fallback;
  return JSON.parse(fs.readFileSync(file, "utf8"));
}

function readText(file) {
  if (!file) return "";
  try {
    return fs.readFileSync(file, "utf8").trim();
  } catch {
    return "";
  }
}

function writeJSON(file, data) {
  fs.mkdirSync(path.dirname(path.resolve(file)), { recursive: true });
  fs.writeFileSync(file, JSON.stringify(data, null, 2), "utf8");
}

function compact(value, max = 500) {
  const text = String(value || "").replace(/\s+/g, " ").trim();
  return text.length > max ? `${text.slice(0, max - 1)}…` : text;
}

function list(value) {
  return Array.isArray(value) ? value.filter((item) => item !== null && item !== undefined && String(item).trim() !== "") : [];
}

function topicCards(review) {
  if (!review || typeof review !== "object") return [];
  if (Array.isArray(review.topic_cards)) return review.topic_cards;
  if (review.result && Array.isArray(review.result.topic_cards)) return review.result.topic_cards;
  if (review.data?.result && Array.isArray(review.data.result.topic_cards)) return review.data.result.topic_cards;
  return [];
}

function selectTopic(review, topicID) {
  const cards = topicCards(review);
  if (!cards.length) return null;
  if (topicID) return cards.find((card) => card.topic_id === topicID) || null;
  return cards.find((card) => card.status === "shortlisted") || cards[0];
}

function normalizeLongformPlan(payload, selectedTopic = null) {
  if (!payload || typeof payload !== "object") {
    return selectedTopic?.longform_plan || {};
  }
  if (payload.longform_plan && typeof payload.longform_plan === "object") return payload.longform_plan;
  if (payload.plan && typeof payload.plan === "object") return payload.plan;
  return selectedTopic?.longform_plan || {};
}

function claimMap(plan) {
  const map = new Map();
  for (const claim of plan?.core_claims || []) map.set(claim.claim_id, claim);
  return map;
}

function coverageConstraint(status) {
  switch (status) {
    case "covered_by_fact_evidence":
      return "May state as supported by attached primary or institutional evidence.";
    case "covered_by_secondary_evidence":
      return "Attribute or qualify; do not present as primary proof.";
    case "market_signal_only":
      return "Use only as evidence of discussion or attention, not factual truth.";
    case "evidence_candidate_found":
      return "A candidate exists but is not verified; avoid factual certainty.";
    case "fallback_only":
      return "No usable evidence; frame as opinion, question, or practical advice.";
    case "generated_only":
      return "Use as explanation or narrative structure, not proof.";
    default:
      return "No coverage status; keep wording cautious.";
  }
}

function buildClaimConstraints(plan, assets) {
  const claims = claimMap(plan);
  const coverage = assets?.coverage || {};
  return [...claims.values()].map((claim) => {
    const status = coverage[claim.claim_id] || (claim.need_evidence ? "fallback_only" : "generated_only");
    return {
      claim_id: claim.claim_id,
      section: claim.section || "",
      claim: claim.claim || "",
      claim_type: claim.claim_type || "",
      evidence_level: claim.evidence_level || "",
      coverage: status,
      writing_constraint: coverageConstraint(status)
    };
  });
}

function normalizeProfile(profile, extra) {
  if (!profile || typeof profile !== "object") return { extra };
  const source = profile.profile || profile;
  return {
    id: source.id || source.ID || "",
    name: source.name || source.display_name || "",
    identity: source.identity || "",
    audience: list(source.audience),
    stance: list(source.stance),
    avoid: list(source.avoid),
    style: source.style || source.tone || "",
    format_preferences: source.format_preferences || source.formats || null,
    extra
  };
}

function deliverableItems(deliverables) {
  return list(deliverables?.items).map((item) => ({
    asset_id: item.asset_id || "",
    kind: item.kind || "",
    path: item.path || "",
    claim_ids: list(item.claim_ids),
    source_url: item.source_url || "",
    source_title: item.source_title || "",
    source_tier: item.source_tier || "",
    review_decision: item.review_decision || "",
    review_issues: list(item.review_issues),
    review_note: item.review_note || "",
    usable: item.usable !== false
  }));
}

function sectionBriefs(plan, constraints, deliverables) {
  const byClaim = new Map(constraints.map((item) => [item.claim_id, item]));
  const deliverablesByClaim = new Map();
  for (const item of deliverableItems(deliverables)) {
    for (const claimID of item.claim_ids || []) {
      deliverablesByClaim.set(claimID, [...(deliverablesByClaim.get(claimID) || []), item.asset_id]);
    }
  }
  return (plan?.core_claims || []).map((claim, index) => {
    const constraint = byClaim.get(claim.claim_id);
    return {
      order: index + 1,
      section: claim.section || `section_${index + 1}`,
      claim_id: claim.claim_id,
      claim: claim.claim || "",
      writing_constraint: constraint?.writing_constraint || "",
      suggested_material_asset_ids: deliverablesByClaim.get(claim.claim_id) || [],
      visual_intent: (plan.visual_plan || []).find((visual) => visual.section === claim.section)?.visual_component || ""
    };
  });
}

function estimateTokens(text) {
  const value = String(text || "");
  let cjk = 0;
  let other = 0;
  for (const char of value) {
    if (/[\u3400-\u9fff]/.test(char)) cjk += 1;
    else other += 1;
  }
  return Math.ceil(cjk * 0.9 + other / 4);
}

function main() {
  const profilePath = arg("profile");
  const topicReviewPath = arg("topic-review") || arg("review");
  const longformPlanPath = arg("longform-plan") || arg("longform");
  const planReviewPath = arg("plan-review");
  const materialPlanPath = arg("material-plan") || arg("plan");
  const materialAssetsPath = arg("material-assets") || arg("assets");
  const deliverablesPath = arg("deliverables");
  const output = arg("output", "06_script_writer_payload.json");
  const topicID = arg("topic-id");
  const durationSec = Number(arg("duration-sec", "0")) || undefined;
  const platform = arg("platform", "short_video");

  if (!profilePath) throw new Error("--profile is required");
  if (!topicReviewPath) throw new Error("--topic-review is required");
  if (!materialPlanPath) throw new Error("--material-plan is required");
  if (!materialAssetsPath) throw new Error("--material-assets is required");
  if (!deliverablesPath) throw new Error("--deliverables is required");

  const profile = normalizeProfile(readJSON(profilePath), readText(arg("profile-extra")));
  const topicReview = readJSON(topicReviewPath);
  const selectedTopic = selectTopic(topicReview, topicID);
  if (!selectedTopic) throw new Error(`topic not found: ${topicID || "(default)"}`);
  const longformPlanPayload = readJSON(longformPlanPath, {});
  const longformPlan = normalizeLongformPlan(longformPlanPayload, selectedTopic);
  const planReview = readJSON(planReviewPath, {});
  const materialPlan = readJSON(materialPlanPath);
  const materialAssets = readJSON(materialAssetsPath);
  const deliverables = readJSON(deliverablesPath);
  const constraints = buildClaimConstraints(materialPlan, materialAssets);

  const payload = {
    schema_version: "0.1.0",
    ability: "script.write",
    input: {
      platform,
      target_duration_sec: durationSec || materialPlan.duration_sec || selectedTopic.format_fit?.duration_sec || 90,
      profile,
      topic: {
        topic_id: selectedTopic.topic_id || materialPlan.topic_id || topicID,
        title: selectedTopic.title || materialPlan.topic_title || "",
        public_entry: longformPlan.public_entry || selectedTopic.public_entry || "",
        angle: selectedTopic.angle || "",
        core_opinion: longformPlan.core_thesis || selectedTopic.core_opinion || materialPlan.core_thesis || "",
        common_misunderstanding: selectedTopic.common_misunderstanding || "",
        audience_value: selectedTopic.audience_value || "",
        format_fit: selectedTopic.format_fit || null,
        scores: selectedTopic.scores || null,
        risks: list(selectedTopic.risks)
      },
      longform_plan: longformPlan,
      plan_review: planReview?.result || planReview,
      material_context: {
        coverage: materialAssets.coverage || {},
        coverage_policy: materialAssets.coverage_policy || {},
        deliverables_summary: deliverables.summary || {},
        deliverables: deliverableItems(deliverables),
        failed_tasks: list(materialAssets.failed_tasks).map((task) => ({
          task_id: task.task_id,
          action: task.action,
          error: compact(task.error, 240),
          retryable: task.retryable === true
        }))
      },
      claim_constraints: constraints,
      section_briefs: sectionBriefs(materialPlan, constraints, deliverables),
      writing_requirements: {
        must_respect_profile: true,
        do_not_invent_facts: true,
        attribute_secondary_sources: true,
        market_signal_is_not_fact_proof: true,
        auxiliary_images_are_not_fact_proof: true,
        must_preserve_public_entry: true,
        public_entry_is_opening_direction: longformPlan.public_entry || selectedTopic.public_entry || "",
        output_structure: ["title", "hook", "sections", "full_script", "evidence_notes", "risk_notes", "estimated_duration_sec"]
      }
    },
    options: {
      language: arg("language", "zh-CN"),
      style: arg("style", ""),
      draft_count: Number(arg("draft-count", "1")) || 1
    },
    source_files: {
      profile: path.resolve(profilePath),
      profile_extra: arg("profile-extra") ? path.resolve(arg("profile-extra")) : "",
      topic_review: path.resolve(topicReviewPath),
      longform_plan: longformPlanPath ? path.resolve(longformPlanPath) : "",
      plan_review: planReviewPath ? path.resolve(planReviewPath) : "",
      material_plan: path.resolve(materialPlanPath),
      material_assets: path.resolve(materialAssetsPath),
      deliverables: path.resolve(deliverablesPath)
    },
    generated_at: new Date().toISOString()
  };

  const serializedInput = JSON.stringify(payload.input);
  const estimatedInputTokens = estimateTokens(serializedInput);
  const targetDuration = payload.input.target_duration_sec;
  const estimatedOutputTokens = Math.max(1200, Math.ceil(targetDuration * 7));
  payload.token_estimate = {
    method: "rough_local_estimate_cjk_aware",
    input_chars: serializedInput.length,
    estimated_input_tokens: estimatedInputTokens,
    estimated_output_tokens: estimatedOutputTokens,
    estimated_total_tokens: estimatedInputTokens + estimatedOutputTokens,
    note: "Use backend model usage for billing; this estimate is for preflight budgeting only."
  };

  writeJSON(output, payload);
  console.log(JSON.stringify({
    output: path.resolve(output),
    topic_id: payload.input.topic.topic_id,
    claims: payload.input.claim_constraints.length,
    deliverables: payload.input.material_context.deliverables.length,
    target_duration_sec: payload.input.target_duration_sec,
    estimated_input_tokens: payload.token_estimate.estimated_input_tokens,
    estimated_output_tokens: payload.token_estimate.estimated_output_tokens
  }, null, 2));
}

main();

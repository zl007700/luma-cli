#!/usr/bin/env node
const fs = require("fs");
const path = require("path");

function arg(name, fallback = "") {
  const idx = process.argv.indexOf(`--${name}`);
  return idx >= 0 && idx + 1 < process.argv.length ? process.argv[idx + 1] : fallback;
}

function intArg(name, fallback) {
  const value = Number.parseInt(arg(name, String(fallback)), 10);
  return Number.isFinite(value) ? value : fallback;
}

function readJSON(file) {
  return JSON.parse(fs.readFileSync(file, "utf8"));
}

function writeJSON(file, data) {
  fs.mkdirSync(path.dirname(path.resolve(file)), { recursive: true });
  fs.writeFileSync(file, JSON.stringify(data, null, 2), "utf8");
}

function compact(text, max = 120) {
  const value = String(text || "").replace(/\s+/g, " ").trim();
  return value.length > max ? value.slice(0, max) : value;
}

function unique(items) {
  return [...new Set(items.map((item) => String(item || "").trim()).filter(Boolean))];
}

function selectTopic(review, topicID) {
  const cards = review.topic_cards || [];
  if (topicID) {
    const card = cards.find((item) => item.topic_id === topicID);
    if (!card) throw new Error(`topic not found: ${topicID}`);
    return card;
  }
  return cards.find((item) => item.status === "selected") || cards.find((item) => item.status === "shortlisted") || cards[0];
}

function approvedLongformPlan(payload) {
  if (!payload || typeof payload !== "object") return null;
  return payload.input?.longform_plan || payload.longform_plan || payload.plan || null;
}

function applyApprovedPlan(topic, plan) {
  if (!plan) return topic;
  const outline = Array.isArray(plan.outline) ? plan.outline : [];
  return {
    ...topic,
    title: plan.topic || topic.title,
    public_entry: plan.public_entry || topic.public_entry,
    core_opinion: plan.core_thesis || topic.core_opinion,
    longform_plan: {
      ...plan,
      logic_chain: outline.map((item) => ({
        section: item.section,
        claim: item.claim,
        evidence_role: item.evidence_role,
        bridge_to_next: item.bridge_to_next,
        evidence: []
      }))
    },
    format_fit: {
      ...(topic.format_fit || {}),
      duration_sec: Number(plan.target_duration_sec) || topic.format_fit?.duration_sec
    }
  };
}

function sectionHas(section, values) {
  const text = `${section.section || ""} ${section.visual_need || ""}`.toLowerCase();
  return values.some((value) => text.includes(value.toLowerCase()));
}

function classifyClaim(section, idx) {
  if (sectionHas(section, ["\u5f00\u5934", "\u7ed3\u8bba", "hook"])) return "mixed";
  if (sectionHas(section, ["\u884c\u52a8", "action", "\u5efa\u8bae"])) return "recommendation";
  if (sectionHas(section, ["\u5f71\u54cd", "impact", "\u8bef\u533a", "opinion", "\u5224\u65ad"])) return "opinion";
  if (sectionHas(section, ["\u80cc\u666f", "\u73b0\u8c61", "background", "event", "\u53d1\u5e03"])) return "factual";
  if (Array.isArray(section.evidence) && section.evidence.length > 0) return idx <= 1 ? "factual" : "mixed";
  return idx === 0 ? "mixed" : "opinion";
}

function evidenceLevel(claimType, idx) {
  if (claimType === "factual") return "required";
  if (claimType === "mixed" || idx === 0) return "supporting";
  return "none";
}

function buildClaims(topic) {
  const sections = topic.longform_plan && Array.isArray(topic.longform_plan.logic_chain)
    ? topic.longform_plan.logic_chain
    : [];
  const signalTitles = unique((topic.evidence_signals || []).map((signal) => signal.title));
  const sourceSections = sections.length
    ? sections
    : [
        { section: "开头结论", claim: topic.title, evidence: signalTitles },
        { section: "现象背景", claim: topic.angle || signalTitles[0] || topic.title, evidence: signalTitles },
        { section: "核心判断", claim: topic.core_opinion || topic.title, evidence: signalTitles },
        { section: "常见误区", claim: topic.common_misunderstanding || "", evidence: [] },
        { section: "对受众的影响", claim: topic.audience_value || "", evidence: [] }
      ].filter((section) => section.claim);

  return sourceSections.map((section, idx) => {
    const claimType = classifyClaim(section, idx);
    const level = evidenceLevel(claimType, idx);
    return {
      claim_id: `claim_${String(idx + 1).padStart(3, "0")}`,
      section: section.section || `section_${idx + 1}`,
      claim: compact(section.claim || topic.core_opinion || topic.title, 220),
      claim_type: claimType,
      evidence_level: level,
      need_evidence: level !== "none",
      risk_level: level === "required" ? "high" : level === "supporting" ? "medium" : "low",
      existing_evidence: unique(section.evidence || []),
      material_requirements: level === "required"
        ? ["authoritative_web_source", "readable_capture"]
        : level === "supporting"
          ? ["market_signal_or_authoritative_source"]
          : ["generated_visual"]
    };
  });
}

function evidenceTaskForSignal(signal, claimID, idx) {
  return {
    task_id: `mat_signal_${String(idx + 1).padStart(3, "0")}`,
    action: "make_signal_card",
    evidence_role: "market_signal",
    priority: idx < 2 ? "high" : "medium",
    purpose: "Show that the market is discussing this angle. Do not use it as objective proof.",
    claim_ids: [claimID],
    input: {
      source: signal.source || "",
      title: signal.title || "",
      author_name: signal.author_name || "",
      url: signal.url || "",
      likes: signal.likes || 0,
      published_at: signal.published_at || ""
    }
  };
}

function buildSearchQuery(topic, claim) {
  const factualSubject = compact(topic.title || topic.theme || "", 72);
  const claimText = compact(claim.claim || "", 72);
  if (claim.claim_type === "factual") {
    const evidenceTitle = claim.existing_evidence?.[0] || "";
    const matchingSignal = (topic.evidence_signals || []).find((signal) => {
      const title = String(signal.title || "");
      return evidenceTitle && (title.includes(evidenceTitle) || evidenceTitle.includes(title));
    });
    return unique([
      compact(evidenceTitle || claimText || factualSubject, 72),
      matchingSignal?.author_name || "",
      "official"
    ]).join(" ");
  }
  return unique([compact(topic.theme || factualSubject, 60), claimText, "report case study"]).join(" ");
}

function asciiTerms(text) {
  return String(text || "")
    .match(/[A-Za-z][A-Za-z0-9 -]{2,}/g)?.map((item) => item.trim()).filter(Boolean) || [];
}

function buildSearchQueries(topic, claim) {
  const title = compact(topic.title || topic.theme || "", 72);
  const theme = compact(topic.theme || "", 72);
  const claimText = compact(claim.claim || "", 72);
  const thesis = compact(topic.core_opinion || topic.longform_plan?.core_thesis || "", 72);
  const evidenceTitle = claim.existing_evidence?.[0] || "";
  const matchingSignal = (topic.evidence_signals || []).find((signal) => {
    const signalTitle = String(signal.title || "");
    return evidenceTitle && (signalTitle.includes(evidenceTitle) || evidenceTitle.includes(signalTitle));
  });
  const author = matchingSignal?.author_name || "";
  const base = buildSearchQuery(topic, claim);
  const englishHints = asciiTerms(`${title} ${theme} ${claimText} ${thesis}`).join(" ");
  return unique([
    base,
    unique([evidenceTitle, author, "official"]).join(" "),
    unique([claimText, "案例", "报道"]).join(" "),
    unique([theme || title, "AI", "业务流程", "落地", "案例"]).join(" "),
    "企业 AI 落地 业务流程 案例",
    "AI 工具 业务流程 落地 失败",
    "AI adoption business process workflow failure",
    "enterprise AI workflow implementation case study",
    englishHints && !/^ai(?: ai)*$/i.test(englishHints) ? `${englishHints} AI workflow case study` : ""
  ]).filter((query) => query.length >= 6 && query !== "official").slice(0, 10);
}

function webSearchTask(topic, claim, idx) {
  const queries = buildSearchQueries(topic, claim);
  return {
    task_id: `mat_web_${String(idx + 1).padStart(3, "0")}`,
    action: "websearch",
    evidence_role: "fact_evidence_candidate",
    priority: claim.evidence_level === "required" ? "high" : "medium",
    purpose: "Find a source that can support or constrain this claim.",
    query: queries[0],
    queries,
    date_range: "7d",
    max_results: 10,
    recall_strategy: "expanded_query_merge",
    max_capture_candidates: 5,
    max_accepted_captures: 3,
    claim_ids: [claim.claim_id],
    source_policy: {
      prefer: ["official_site", "official_docs", "official_blog", "primary_research", "reputable_media"],
      reject: ["content_farm", "unsourced_repost", "search_snippet_only"]
    },
    acceptance: {
      must_match_claim_keywords: true,
      must_have_readable_page: true,
      capture_smallest_proving_region: true
    },
    next_action: "capture_best_result"
  };
}

function imageSearchTask(topic) {
  const title = compact(topic.title || topic.theme || topic.core_opinion, 90);
  const queries = unique([
    title,
    "business workflow automation diagram",
    "AI workflow diagram",
    "enterprise AI process automation",
    "AI agent workflow",
    "business process automation dashboard",
    "企业 AI 业务流程 图解",
    "AI 落地 业务流程 图"
  ]);
  return {
    task_id: "mat_image_001",
    action: "image_search",
    evidence_role: "auxiliary_visual",
    priority: "low",
    purpose: "Find one clean auxiliary visual. Never use it as factual proof unless the source is official.",
    query: queries[0],
    queries,
    count: 12,
    claim_ids: [],
    cost_estimate_credits: 8,
    acceptance: {
      min_width: 800,
      min_height: 450,
      prefer_landscape: true,
      require_source_url: true,
      reject_watermark_heavy: true
    }
  };
}

function visualForSection(section, idx) {
  let component = "ChapterCard";
  if (sectionHas(section, ["\u8bef\u533a", "compare", "vs"])) component = "CompareTwoSides";
  if (sectionHas(section, ["\u6838\u5fc3", "\u6d41\u7a0b", "process", "logic"])) component = "ProcessFlowCard";
  if (sectionHas(section, ["\u884c\u52a8", "action", "\u5efa\u8bae"])) component = "ActionListCard";
  if (sectionHas(section, ["\u5f71\u54cd", "impact"])) component = "ImpactListCard";
  if (idx === 0) component = "HookBigText";
  return {
    segment_id: `visual_${String(idx + 1).padStart(3, "0")}`,
    section: section.section || `section_${idx + 1}`,
    claim: section.claim || "",
    visual_component: component,
    asset_need: ["ProcessFlowCard", "CompareTwoSides", "ImpactListCard"].includes(component)
      ? "generated_component"
      : "text_or_evidence_card",
    fallback: "chapter_text_card"
  };
}

function main() {
  const reviewPath = arg("review") || arg("input");
  const longformPlanPath = arg("longform-plan");
  const topicID = arg("topic-id");
  const output = arg("output", "04_material_plan.json");
  const maxWebQueries = Math.max(0, intArg("max-web-queries", 2));
  const maxImageQueries = Math.max(0, intArg("max-image-queries", 1));
  if (!reviewPath) throw new Error("--review is required");

  const review = readJSON(reviewPath);
  let topic = selectTopic(review, topicID);
  if (!topic) throw new Error("no topic card found");
  if (longformPlanPath) {
    topic = applyApprovedPlan(topic, approvedLongformPlan(readJSON(longformPlanPath)));
  }

  const claims = buildClaims(topic);
  const evidence = topic.evidence_signals || [];
  const collectionTasks = [];
  evidence.slice(0, 6).forEach((signal, idx) => {
    const claim = claims[Math.min(idx, claims.length - 1)];
    collectionTasks.push(evidenceTaskForSignal(signal, claim.claim_id, idx));
  });

  const evidenceClaims = claims
    .filter((claim) => claim.need_evidence)
    .sort((a, b) => {
      const rank = { required: 2, supporting: 1, none: 0 };
      return rank[b.evidence_level] - rank[a.evidence_level];
    })
    .slice(0, maxWebQueries);
  evidenceClaims.forEach((claim, idx) => collectionTasks.push(webSearchTask(topic, claim, idx)));

  const wantsAuxiliaryImage = (topic.material_hypothesis || []).length > 0 || claims.length >= 4;
  if (maxImageQueries > 0 && wantsAuxiliaryImage) {
    collectionTasks.push(imageSearchTask(topic));
  }

  const logic = topic.longform_plan && Array.isArray(topic.longform_plan.logic_chain)
    ? topic.longform_plan.logic_chain
    : claims.map((claim) => ({ section: claim.section, claim: claim.claim }));

  const plan = {
    schema_version: "0.2.0",
    topic_id: topic.topic_id,
    topic_title: topic.title,
    source_topic_card: topic,
    video_format: topic.format_fit?.recommended || "short_explainer",
    duration_sec: topic.format_fit?.duration_sec || 90,
    core_thesis: topic.longform_plan?.core_thesis || topic.core_opinion || topic.title,
    core_claims: claims,
    visual_plan: logic.map(visualForSection),
    collection_tasks: collectionTasks,
    collection_budget: {
      max_web_queries: maxWebQueries,
      max_image_queries: maxImageQueries,
      estimated_image_search_credits: maxImageQueries * 8,
      browser_captures: Math.min(maxWebQueries, 3)
    },
    evidence_policy: {
      fact_coverage_requires: ["readable_capture", "source_url", "claim_match"],
      market_signal_never_counts_as_fact_proof: true,
      generated_components_never_count_as_fact_proof: true,
      image_search_is_auxiliary_unless_official_source: true
    },
    fallback_strategy: {
      if_no_authoritative_web: ["capture a relevant search result page", "downgrade factual wording to opinion"],
      if_social_screenshot_unavailable: ["use generated signal card from title, author, and stats"],
      if_image_search_unusable: ["use generated component or digital-human shot"],
      if_no_real_assets: ["HookBigText", "CompareTwoSides", "ProcessFlowCard", "ActionListCard"]
    },
    generated_at: new Date().toISOString()
  };
  writeJSON(output, plan);
  console.log(JSON.stringify({
    output: path.resolve(output),
    topic_id: plan.topic_id,
    tasks: plan.collection_tasks.length,
    budget: plan.collection_budget
  }, null, 2));
}

main();

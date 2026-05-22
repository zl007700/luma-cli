#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

const repoRoot = path.resolve(__dirname, "..");
const skillsRoot = path.join(repoRoot, "skills");
const outRoot = path.resolve(process.argv[2] || path.join(repoRoot, "dist", "skills-site"));
const wellKnownRoot = path.join(outRoot, ".well-known", "agent-skills");

function parseFrontmatter(content) {
  const match = content.match(/^---\r?\n([\s\S]*?)\r?\n---/);
  if (!match) {
    return {};
  }
  const data = {};
  for (const line of match[1].split(/\r?\n/)) {
    const kv = line.match(/^([A-Za-z0-9_-]+):\s*(.*)$/);
    if (!kv) {
      continue;
    }
    let value = kv[2].trim();
    if ((value.startsWith('"') && value.endsWith('"')) || (value.startsWith("'") && value.endsWith("'"))) {
      value = value.slice(1, -1);
    }
    data[kv[1]] = value;
  }
  return data;
}

function copyFile(src, dst) {
  fs.mkdirSync(path.dirname(dst), { recursive: true });
  fs.copyFileSync(src, dst);
}

fs.rmSync(outRoot, { recursive: true, force: true });
fs.mkdirSync(wellKnownRoot, { recursive: true });

const entries = [];
for (const dirent of fs.readdirSync(skillsRoot, { withFileTypes: true })) {
  if (!dirent.isDirectory()) {
    continue;
  }
  const skillDir = path.join(skillsRoot, dirent.name);
  const skillPath = path.join(skillDir, "SKILL.md");
  if (!fs.existsSync(skillPath)) {
    continue;
  }
  const content = fs.readFileSync(skillPath, "utf8");
  const meta = parseFrontmatter(content);
  const name = meta.name || dirent.name;
  const description = meta.description;
  if (!description) {
    throw new Error(`Missing description in ${skillPath}`);
  }
  const targetDir = path.join(wellKnownRoot, name);
  copyFile(skillPath, path.join(targetDir, "SKILL.md"));
  entries.push({
    name,
    description,
    files: ["SKILL.md"],
  });
}

entries.sort((a, b) => a.name.localeCompare(b.name));
fs.writeFileSync(
  path.join(wellKnownRoot, "index.json"),
  JSON.stringify({ skills: entries }, null, 2) + "\n",
  "utf8"
);

console.log(`Built ${entries.length} skills at ${wellKnownRoot}`);
console.log("Host the output directory at: https://pikgeo.com/skills/luma");
console.log("Then test with: npx -y skills add https://pikgeo.com/skills/luma -l");

#!/usr/bin/env node
// scripts/check-pages-links.mjs
//
// Link integrity checker for docs/index.html and the docs/sections/*.md
// sources. Verifies:
//
//   1. Every relative href points to a file that exists in docs/.
//   2. Every external anchor link (https://...) reaches the host (DNS
//      resolves) and HTTP HEAD returns 2xx or 3xx. Network errors are
//      treated as a soft warning when --soft is set, otherwise as a
//      hard failure (exit 1).
//   3. Every in-page anchor (#main, #hero, ...) has a matching id in
//      the same document.
//
// Designed to run in CI (.github/workflows/pages-freshness.yml) on
// every PR that touches docs/**.
//
// Usage:
//   node scripts/check-pages-links.mjs              # strict (default)
//   node scripts/check-pages-links.mjs --soft       # warn-only
//
// Exit codes:
//   0 — every link resolved.
//   1 — at least one link failed (relative missing, anchor missing, or
//       remote non-2xx/3xx when --soft is not set).
//   2 — usage error.

import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import dns from 'node:dns/promises';

const here = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(here, '..');
const docsRoot = path.join(repoRoot, 'docs');

const args = new Set(process.argv.slice(2));
const soft = args.has('--soft');

// Allowlist of external URLs that are intentionally not HEAD-checked
// because they are private (auth-gated), preconnect hints, or simply
// return 404 to HEAD with cookies disabled. These are still rendered
// in the page; they are not link-rot.
const allowExternal = new Set([
  // Google Fonts preconnect targets. The browser never navigates to
  // the bare host; it only opens a connection before fetching the
  // font CSS. HEAD returns 404.
  'https://fonts.googleapis.com',
  'https://fonts.gstatic.com',
  // Container registry URL. The package is not yet published but the
  // link is canonical and will resolve on the next release. HEAD to
  // GitHub returns 404 because the GitHub Packages page requires
  // browser session cookies.
  'https://github.com/fgjcarlos/nrcc/pkgs/container/nrcc',
]);

// Allowlist of relative paths that are part of a multi-commit PR and
// may not yet exist in the working tree at the time of W2. Each entry
// must be deleted once the linked file ships in the same PR.
// (Empty — every linked file is committed in the same work-unit that
// introduces the link.)
const allowMissingRelative = new Set([]);

if (args.has('--help') || args.has('-h')) {
  console.log(
    'Usage: node scripts/check-pages-links.mjs [--soft]',
  );
  process.exit(0);
}

function fail(message, code = 1) {
  console.error('\u2717', message);
  process.exitCode = code;
}

function warn(message) {
  console.warn('!', message);
}

function ok(message) {
  console.log('\u2713', message);
}

function isExternal(href) {
  return /^[a-z][a-z0-9+.-]*:\/\//i.test(href) || href.startsWith('//');
}

function isMailOrTel(href) {
  return /^(mailto:|tel:|javascript:|data:|#)/.test(href);
}

async function collectLinks(file) {
  const text = await fs.readFile(file, 'utf8');
  const anchors = [];
  const idRegex = /id\s*=\s*"([^"]+)"/g;
  for (const match of text.matchAll(idRegex)) {
    anchors.push(match[1]);
  }
  // Two separate passes:
  //   * href="..."  -> navigation / clickable links. We resolve external
  //     URLs and relative paths.
  //   * src="..."   -> asset references. Only relative paths are
  //     resolved (preconnect/dns-prefetch src are network hints, not
  //     real navigations, and HEAD-ing them returns 404 anyway).
  const hrefRegex = /\bhref\s*=\s*"([^"]+)"/g;
  const srcRegex = /\bsrc\s*=\s*"([^"]+)"/g;
  const links = [];
  for (const match of text.matchAll(hrefRegex)) {
    const value = match[1];
    if (!value) continue;
    if (value.startsWith('#')) {
      links.push({ value, kind: 'anchor', from: file });
      continue;
    }
    if (isMailOrTel(value)) {
      continue;
    }
    if (isExternal(value)) {
      links.push({ value, kind: 'external', from: file });
      continue;
    }
    const [pathPart] = value.split('#');
    links.push({ value: pathPart, kind: 'relative', from: file });
  }
  for (const match of text.matchAll(srcRegex)) {
    const value = match[1];
    if (!value) continue;
    if (value.startsWith('#') || isMailOrTel(value)) continue;
    if (isExternal(value)) continue;
    const [pathPart] = value.split('#');
    links.push({ value: pathPart, kind: 'relative', from: file });
  }
  return { links, anchors };
}

async function resolveRelative(file, target) {
  const fromDir = path.dirname(file);
  const absolute = path.resolve(fromDir, target);
  const relPath = path.relative(repoRoot, absolute);
  if (allowMissingRelative.has(relPath) || allowMissingRelative.has(target)) {
    warn(`allowing missing relative target (PR-bound, ships later): ${path.relative(repoRoot, file)} -> ${target}`);
    return true;
  }
  try {
    const stat = await fs.stat(absolute);
    if (stat.isDirectory()) {
      fail(`relative link points to a directory without index.html: ${path.relative(repoRoot, file)} -> ${target}`);
      return false;
    }
    return true;
  } catch {
    fail(`relative link target missing: ${path.relative(repoRoot, file)} -> ${target}`);
    return false;
  }
}

async function resolveAnchor(file, anchor) {
  // In-page anchors are validated against the same file's id set.
  const text = await fs.readFile(file, 'utf8');
  const ids = [...text.matchAll(/id\s*=\s*"([^"]+)"/g)].map((m) => m[1]);
  if (!ids.includes(anchor)) {
    fail(`in-page anchor #${anchor} not found in ${path.relative(repoRoot, file)}`);
    return false;
  }
  return true;
}

async function checkRemote(href) {
  if (allowExternal.has(href)) {
    warn(`skipping allowlisted external URL: ${href}`);
    return true;
  }
  let url;
  try {
    url = new URL(href);
  } catch {
    fail(`unparseable external URL: ${href}`);
    return false;
  }
  try {
    await dns.lookup(url.hostname);
  } catch (error) {
    const message = `DNS lookup failed for ${url.hostname}: ${error.code || error.message}`;
    if (soft) {
      warn(message);
      return true;
    }
    fail(message);
    return false;
  }
  try {
    const response = await fetch(url, { method: 'HEAD', redirect: 'follow' });
    if (response.status >= 200 && response.status < 400) {
      return true;
    }
    const message = `external link returned ${response.status}: ${href}`;
    if (soft) {
      warn(message);
      return true;
    }
    fail(message);
    return false;
  } catch (error) {
    const message = `external link fetch failed: ${href} (${error.message})`;
    if (soft) {
      warn(message);
      return true;
    }
    fail(message);
    return false;
  }
}

async function main() {
  const targets = [
    path.join(docsRoot, 'index.html'),
    ...(await fs.readdir(path.join(docsRoot, 'sections'))).map((name) =>
      path.join(docsRoot, 'sections', name),
    ),
  ];

  let total = 0;
  let relative = 0;
  let external = 0;
  let anchors = 0;

  for (const file of targets) {
    const stat = await fs.stat(file).catch(() => null);
    if (!stat || !stat.isFile()) {
      fail(`target not found: ${path.relative(repoRoot, file)}`);
      continue;
    }
    const { links } = await collectLinks(file);
    for (const link of links) {
      total += 1;
      if (link.kind === 'relative') {
        relative += 1;
        await resolveRelative(file, link.value);
      } else if (link.kind === 'external') {
        external += 1;
        await checkRemote(link.value);
      } else if (link.kind === 'anchor') {
        anchors += 1;
        await resolveAnchor(file, link.value.slice(1));
      }
    }
  }

  ok(`scanned ${targets.length} file(s): ${total} link(s) — relative=${relative}, external=${external}, anchors=${anchors}`);

  if (process.exitCode) {
    console.error('link integrity: FAILED');
  } else {
    ok('link integrity: PASSED');
  }
}

main().catch((error) => {
  console.error('check-pages-links crashed:', error);
  process.exit(2);
});
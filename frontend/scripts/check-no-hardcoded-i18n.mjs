import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const frontendRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const sourceRoot = path.join(frontendRoot, 'src');
const skippedDirectories = new Set(['locales', '__test__']);
const brands = new Set([
  'NRCC', 'Node-RED', 'daisyUI', 'JSON', 'JSX', 'ESLint', 'TypeScript',
  'Vite', 'Vitest', 'Playwright', 'Restish', 'OpenAPI', 'adminAuth',
  'httpNodeAuth', 'httpStaticAuth', 'functionGlobalContext',
  'credentialSecret', 'requireHttps', 'settings.js', 'package.json',
]);

function walk(directory) {
  const files = [];
  for (const entry of fs.readdirSync(directory, { withFileTypes: true })) {
    if (entry.isDirectory()) {
      if (!skippedDirectories.has(entry.name)) files.push(...walk(path.join(directory, entry.name)));
      continue;
    }
    if (entry.isFile() && /\.tsx$/.test(entry.name) && !/\.test\.tsx$/.test(entry.name)) files.push(path.join(directory, entry.name));
  }
  return files;
}

function extractJsxText(source) {
  const literals = [];
  const expression = />([^<>{}]+)</g;
  for (const match of source.matchAll(expression)) {
    const literal = match[1].replace(/\s+/g, ' ').trim();
    if (!literal) continue;
    const line = source.slice(0, match.index + 1).split('\n').length;
    literals.push({ line, literal });
  }
  return literals;
}

function isCodeLike(literal) {
  return /[`;]/.test(literal) || /\.[cm]?[jt]sx?\b/.test(literal) ||
    /\b\w+\s*\{/.test(literal) || /\b\w+\([^)]*\)/.test(literal);
}

function isTypeIdentifier(literal) {
  // Single PascalCase identifier (e.g. Promise, Component, ReactNode).
  // Avoids flagging arrow-then-generic patterns like '() => Promise<...>'
  // where the captured 'Promise' sits between the arrow and the generic.
  const trimmed = literal.trim();
  return /^[A-Z][A-Za-z0-9]*$/.test(trimmed) && trimmed.length >= 3;
}

function isViolation(literal) {
  if (!literal || /^\s*$/.test(literal) || /^[\p{P}\p{S}\s]+$/u.test(literal)) return false;
  if (brands.has(literal) || isCodeLike(literal) || isTypeIdentifier(literal)) return false;
  if (/^[a-z]+$/.test(literal) && literal.length <= 4) return false;
  const letters = (literal.match(/[A-Za-z]/g) ?? []).length;
  return letters >= 3 && literal.length >= 5;
}

const violations = [];
for (const file of walk(sourceRoot)) {
  const relative = path.relative(frontendRoot, file).split(path.sep).join('/');
  const source = fs.readFileSync(file, 'utf8');
  for (const { line, literal } of extractJsxText(source)) {
    if (isViolation(literal)) violations.push({ file: relative, line, literal });
  }
}

if (violations.length > 0) {
  const byFile = Map.groupBy(violations, ({ file }) => file);
  for (const [file, findings] of byFile) {
    console.error(`${file}:`);
    for (const finding of findings) console.error(`  L${finding.line}: ${JSON.stringify(finding.literal)}`);
  }
}
console.log(JSON.stringify(violations, null, 2));
process.exitCode = violations.length === 0 ? 0 : 1;

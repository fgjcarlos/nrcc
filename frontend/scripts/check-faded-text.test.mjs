// Run with: node scripts/check-faded-text.test.mjs
import assert from 'node:assert/strict'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const guard = path.join(path.dirname(fileURLToPath(import.meta.url)), 'check-faded-text.mjs')

function runOn(files) {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'faded-text-'))
  try {
    for (const [name, content] of Object.entries(files)) {
      fs.mkdirSync(path.dirname(path.join(dir, name)), { recursive: true })
      fs.writeFileSync(path.join(dir, name), content)
    }
    return spawnSync(process.execPath, [guard, dir], { encoding: 'utf8', cwd: dir })
  } finally {
    fs.rmSync(dir, { recursive: true, force: true })
  }
}

// Clean tree: tokens, /65 and above, escaped lines and test files all pass.
const clean = runOn({
  'Clean.tsx': [
    '<p className="text-ds-text-muted">a</p>',
    '<p className="text-base-content/65">b</p>',
    '<p className="text-base-content">c</p>',
    '<button className="disabled:text-base-content/40" // contrast-ok: disabled control',
    '<Icon className="text-base-content/35" aria-hidden="true" />{/* contrast-ok: decorative */}',
  ].join('\n'),
  'Thing.test.tsx': '<p className="text-base-content/40">test-only</p>',
  '__tests__/Fixture.tsx': '<p className="text-base-content/40">fixture</p>',
  'notes.ts': 'const c = "text-base-content/40"',
})
assert.equal(clean.status, 0, clean.stderr)
assert.match(clean.stdout, /No opacity-faded text below \/65 in 1 files\./)

// Faded tree: every sub-/65 use is reported with file:line, variants included.
const faded = runOn({
  'nested/Faded.tsx': [
    '<p className="text-xs text-base-content/50">a</p>',
    '<input className="placeholder:text-base-content/45" />',
    '<p className="text-base-content/70">ok</p>',
    '<p className="text-base-content/60">b</p> // contrast-ok:',
  ].join('\n'),
})
assert.equal(faded.status, 1)
const reported = faded.stderr.split('\n').filter((l) => l.startsWith('  '))
assert.deepEqual(reported, [
  `  ${path.join('nested', 'Faded.tsx')}:1: text-base-content/50`,
  `  ${path.join('nested', 'Faded.tsx')}:2: text-base-content/45`,
  `  ${path.join('nested', 'Faded.tsx')}:4: text-base-content/60`,
])

console.log('check-faded-text: all assertions passed')

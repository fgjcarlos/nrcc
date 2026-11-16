// Fails when readable text is faded with an opacity utility below /65.
// `text-base-content/NN` mixes escape check-contrast-tokens (it only checks
// token pairs), and /60 already drops to 4.42:1 on the light sunken ground.
// Use text-ds-text-muted or text-ds-text-secondary instead (DESIGN.md).
//
// Usage: node scripts/check-faded-text.mjs [srcDir]   (default: ../src)
// Escape: put `// contrast-ok: <reason>` or `{/* contrast-ok: <reason> */}`
// on the same line, for disabled states or aria-hidden decorative icons.

import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const MIN_OPACITY = 65
const FADED = /(?<![\w-])text-base-content\/(\d{1,3})(?![\w.])/g
const ESCAPE = /(?:\/\/|\/\*)\s*contrast-ok:\s*\S/
const TEST_FILE = /\.(test|spec)\.tsx$/

const srcDir = path.resolve(
  process.argv[2] ?? path.join(path.dirname(fileURLToPath(import.meta.url)), '..', 'src'),
)

function* walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const full = path.join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name !== '__tests__' && entry.name !== 'node_modules') yield* walk(full)
    } else if (entry.name.endsWith('.tsx') && !TEST_FILE.test(entry.name)) {
      yield full
    }
  }
}

const findings = []
let scanned = 0
for (const file of walk(srcDir)) {
  scanned++
  fs.readFileSync(file, 'utf8').split('\n').forEach((line, i) => {
    if (ESCAPE.test(line)) return
    for (const match of line.matchAll(FADED)) {
      if (Number(match[1]) < MIN_OPACITY) {
        findings.push(`${path.relative(process.cwd(), file)}:${i + 1}: ${match[0]}`)
      }
    }
  })
}

if (findings.length > 0) {
  console.error(
    `Opacity-faded text below /${MIN_OPACITY} (use text-ds-text-muted or text-ds-text-secondary, ` +
      'or add `contrast-ok: <reason>` on the line):',
  )
  for (const finding of findings) console.error(`  ${finding}`)
  process.exit(1)
}
console.log(`No opacity-faded text below /${MIN_OPACITY} in ${scanned} files.`)

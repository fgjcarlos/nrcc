# Skill Registry — Node-RED Control Center

Generated: 2026-04-09

## Available Skills

### User-level Skills (`~/.config/opencode/skills/`)

| Name | Trigger / Description |
|------|-----------------------|
| `sdd-init` | Initialize SDD context in a project. Trigger: "sdd init", "iniciar sdd", "openspec init". |
| `sdd-explore` | Explore and investigate ideas before committing to a change. |
| `sdd-propose` | Create a change proposal with intent, scope, and approach. |
| `sdd-spec` | Write specifications with requirements and scenarios. |
| `sdd-design` | Create technical design document with architecture decisions. |
| `sdd-tasks` | Break down a change into an implementation task checklist. |
| `sdd-apply` | Implement tasks from the change, writing actual code. |
| `sdd-verify` | Validate that implementation matches specs, design, and tasks. |
| `sdd-archive` | Sync delta specs to main specs and archive a completed change. |
| `skill-registry` | Create or update the skill registry for the current project. |
| `skill-creator` | Creates new AI agent skills following the Agent Skills spec. |
| `go-testing` | Go testing patterns. Trigger: writing Go tests, teatest, adding test coverage. |

### Project-level Skills (`.agents/skills/`)

| Name | Trigger / Description |
|------|-----------------------|
| `accessibility` | WCAG 2.2 audit and improvements. Trigger: "improve accessibility", "a11y audit", "WCAG compliance", "screen reader support", "keyboard navigation". |
| `seo` | SEO optimization. Trigger: "improve SEO", "fix meta tags", "add structured data", "sitemap optimization". |

## Project Conventions

No root-level `AGENTS.md`, `CLAUDE.md`, `.cursorrules`, or `GEMINI.md` found.

Project is a Go+React monorepo with a single binary distribution model. See `TECHNICAL_OVERVIEW.md` for full architecture context.

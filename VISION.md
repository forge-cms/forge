# Forge — Vision

This document captures the long-term vision for Forge and Forge Cloud.

Last updated: 2026-03-18

---

## The core thesis

Most CMS tooling is built for humans to operate. AI assistants are bolted on
as an afterthought — a chat interface over a system that was never designed
for machine interaction.

Forge is built differently. From day one, every architectural decision has
been made with four audiences in mind: the developer writing code, the AI
assistant helping build it, the human visiting the resulting site, and the
AI agent consuming its content.

The result is a framework where AI is not an add-on. It is a first-class
participant in every layer — content creation, content delivery, and
content management.

---

## The vision in one sentence

A user tells their AI assistant: *"Make me a blog with these specs. The first
post should be about my experience today."* Ten minutes later, the blog is live.

No code. No deployment pipeline. No configuration files. Just a conversation.

---

## Authored AI — the methodology

Forge is the product. Authored AI is the methodology it was built with —
and the methodology it enables.

**Authored AI** describes AI-partnered development where the human is the
author of all architectural decisions and governance, and AI is the
implementation partner. The human directs. AI implements. The human owns
the output — not because they wrote every line, but because they made every
decision that mattered.

This is the natural evolution beyond vibe coding. Vibe coding is fast and
powerful — and it will collapse under its own speed if nobody is making
the architectural decisions. Authored AI is what happens when you bring
governance to AI-partnered development.

The key insight: AI has no memory between sessions. If architectural
decisions are not persisted — in DECISIONS.md, in amendment protocols, in
copilot instructions — the reasoning exists only in chat history that will
be closed. The AI cannot help you maintain what it cannot remember building.

**Authored AI in practice:**
- Every architectural decision is documented before implementation
- An amendment protocol governs cross-file changes
- AI instructions encode the rules and constraints
- The human reviews and approves before any commit
- The output is attributable — every decision has a named author

Forge embodies this methodology. Its API naming, file structure, and
architecture documentation are designed to be unambiguous — so any AI
assistant, given the codebase and the docs, produces idiomatic output on
the first try. Not because the AI is clever, but because the framework
leaves no room for ambiguity.

---

## Why this is achievable with Forge

**Content lifecycle as a first-class concept.** `forge.Node` enforces
Draft → Scheduled → Published → Archived for every content type. An AI
assistant creating content operates within the same lifecycle rules as a
human editor. There is no special mode, no bypass, no unsafe shortcut.

**Structured schema from struct tags.** A `BlogPost` content type already
defines its own schema via Go struct tags and the `Head()` method. Forge
derives an MCP resource schema from this automatically — no separate schema
definition, no drift between code and documentation.

**Role system the AI respects.** `forge.Auth` and the role hierarchy
(Guest → Author → Editor → Admin) apply equally to human requests and MCP
tool calls. An AI assistant operating as an Author cannot delete another
author's published content. The rules are the same.

**Validation the AI cannot bypass.** `forge.Validate` and `Validate() error`
run on every save, regardless of who or what initiated the save. An AI
creating a post that violates validation rules gets the same 422 response
a human would.

**AI-readable output already built in.** `llms.txt`, `llms-full.txt`, AIDoc
endpoints, and gzip-compressed AI responses are part of v1.0.0. A Forge
site is already optimised for AI consumption before MCP is added.

**Forge is a structured contract layer between humans and AI.** The role
system, lifecycle enforcement, and validation rules apply equally to human
requests and AI agent calls. You can give an AI assistant precisely the
access level it needs — and know with certainty that it cannot exceed it.
Not because you wrote special AI rules, but because the rules already exist
for everyone.

---

## MCP as the foundation

MCP (Model Context Protocol) is the technical layer that makes the ten-minute
blog vision real.

Forge's existing architecture maps cleanly onto MCP primitives:

| Forge concept | MCP concept |
|---|---|
| `forge.Node` + struct tags | Resource schema (auto-derived) |
| `forge.Module` operations | Tools (Create, Update, Publish, Delete) |
| `forge.Auth` / role system | Authentication (same rules, same roles) |
| `forge.Validate` | Tool input validation (same constraints) |
| Content lifecycle | Resource state machine (same states) |

MCP is not a new system sitting beside Forge. It is a thin transport layer
over semantics that already exist. The schema is already defined. The rules
are already enforced. MCP exposes them to AI assistants over a structured
protocol.

### What an AI assistant can do via Forge MCP

**Content operations:**
- Create, update, publish, archive, and delete content
- Schedule posts for future publication
- Query content by status, tag, or date range

**Site management:**
- Inspect and update redirect rules
- Check SEO status of published content
- Query sitemap coverage

### The ten-minute blog — step by step

```
User → AI assistant:
  "Create a blog about my travels. First post: my day in Copenhagen today."

AI assistant → Forge MCP:
  tool: create_content
  args: { type: "Post", title: "A day in Copenhagen",
          body: "...", status: "published" }

Result: blog is live, one published post.
Total time: under 10 minutes.
```

The AI assistant does not write code. It calls well-defined tools over a
structured protocol, operating within the same constraints as any other
authenticated user of the system.

---

## Roadmap

### Phase 1 — MCP core (M10, v2.0.0)

Implement the MCP server in Forge. This is the technical prerequisite for
everything that follows.

- MCP server transport: stdio (local tools) and SSE (remote, authenticated)
- Auto-derive resource schema from `forge.Node` struct tags
- Expose module CRUD operations as typed MCP tools
- Apply existing role system and validation to all MCP calls
- Separate `forge-mcp` package to preserve zero-dependency core

### Phase 2 — Production foundation

Close the remaining gaps between the open source framework and a hosted
offering.

- `forge-pgx` integration tests against a real database
- Shared template partials
- `forge:head` public helper
- `forge.New` MustConfig enforcement
- `forge.AppSchema{}` and `forge.OGDefaults{}`
- Webhooks
- `forge.Analytics` middleware
- `forge-admin`: web-based admin UI

### Phase 3 — Forge Cloud private beta

Invitation-only. Manually provisioned instances to start.

- Site provisioning
- Content model definition via admin UI (no-code)
- Forge Cloud MCP layer
- Custom domain and SSL

### Phase 4 — Forge Cloud general availability

- Automated multi-tenant provisioning
- Team management and billing
- Commercial license (AGPL exemption) introduced

---

## Licensing

### Current — AGPL v3

`forge` and `forge-mcp` are licensed under the GNU Affero General
Public License v3 (AGPL).

AGPL means: the source code is open and free to use, modify, and distribute.
If you use Forge to provide a hosted service to others, you must release your
modifications under the same license.

For individual developers, open source projects, and companies building their
own sites with Forge: AGPL imposes no meaningful restriction. You can use
Forge freely.

### forge-admin — closed source

`forge-admin` is a commercial component of Forge Cloud. It is not
part of the open source framework and is not available separately.

### Future — Commercial license (AGPL exemption)

When Forge Cloud launches commercially, a commercial license will be available
for organisations that want to use Forge as the basis of a hosted service
without the AGPL obligation.

This is the standard open core model. The framework stays open. The
commercial license is for those who want to build on top of it as a service.

### On the MIT → AGPL transition

Forge launched under MIT. No external contributors exist as of March 2026,
so relicensing to AGPL requires no coordination. The CLA signed by future
contributors grants forge-cms the right to issue commercial licenses without
requiring individual consent.

---

## What this is not

Forge Cloud is not a competitor to Vercel, Netlify, or Railway. Those are
general-purpose deployment platforms. Forge Cloud is a content platform
where AI is a first-class participant in every layer.

The differentiation is not infrastructure. It is the AI-first content model,
the MCP integration, and the structured access that allows an AI assistant
to manage a site as naturally as a human editor would.

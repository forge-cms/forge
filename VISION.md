# Forge — Vision

This document captures the long-term vision for Forge and Forge Cloud.
It is an internal planning document, not intended for public distribution.

Last updated: 2026-03-11

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

## Why this is achievable with Forge

This is not a distant aspiration. Every architectural decision in Forge v1.0.0
points directly toward it.

**Content lifecycle as a first-class concept.** `forge.Node` enforces
Draft → Scheduled → Published → Archived for every content type. An AI
assistant creating content operates within the same lifecycle rules as a
human editor. There is no special mode, no bypass, no unsafe shortcut.

**Structured schema from struct tags.** A `BlogPost` content type already
defines its own schema via Go struct tags and the `Head()` method. Forge can
derive an MCP resource schema from this automatically — no separate schema
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
endpoints, and gzip-compressed AI responses are already part of v1.0.0.
A Forge site is already optimised for AI consumption before MCP is added.

---

## Forge Cloud

Forge Cloud is the hosted offering built on top of the open source framework.

### What it is not

Forge Cloud is not a generic hosting platform. It does not accept arbitrary
Go code. It does not run user-supplied binaries. This would require sandboxing,
build pipelines, and security isolation at a level that is a different product
entirely.

### What it is

Forge Cloud provisions and manages Forge instances. A user defines their
content model — the types, fields, and relationships that make up their site —
and Forge Cloud generates and hosts the corresponding Forge application.

The user never writes Go code. The user never deploys. Forge Cloud owns
the infrastructure, the upgrades, and the backups.

```
User defines:    BlogPost { Title, Body, Tags, CoverImage }
Cloud generates: a running Forge instance with that content model
User receives:   Admin UI + REST API + MCP endpoint + llms.txt + sitemap
```

Because every Forge Cloud instance is a genuine Forge application, it
inherits everything from the framework: lifecycle enforcement, role-based
access, validation, redirects, scheduled publishing, AI endpoints — all
of it, without the user thinking about any of it.

### How humans use Forge Cloud

For most users, the admin UI is the primary interface. Content editors,
marketers, and site owners log in, create and manage content, review
drafts, and publish — exactly as they would in any other CMS. The AI
assistant is an additional capability, not a replacement for the human
interface.

A typical Forge Cloud site has:
- An admin UI for day-to-day content management by humans
- A REST API for developers and integrations
- An MCP endpoint for AI assistants
- Public-facing HTML (or headless, if preferred)

All four interfaces operate on the same content model, the same lifecycle
rules, and the same role system. There is no separate "AI mode" — it is
one system with multiple surfaces.

### The escape hatch

A developer who outgrows the no-code layer can eject to raw Go code at any
time. Forge Cloud exports a complete, idiomatic Forge application that the
developer can take, host themselves, and extend freely. The AGPL-licensed
core means the code is always readable and auditable.

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
- Query content by status, tag, date range, or full-text search

**Site management:**
- Inspect and update redirect rules
- Check SEO status of published content
- Review cookie declarations and compliance manifests
- Query sitemap coverage

**Forge Cloud operations (hosted):**
- Provision a new site with a given content model
- Add a new content type to an existing site
- Configure domain and SSL

### The ten-minute blog — step by step

```
User → AI assistant:
  "Create a blog about my travels. First post: my day in Copenhagen today."

AI assistant → Forge Cloud MCP:
  tool: provision_site
  args: { name: "My Travel Blog", subdomain: "travel" }

  tool: define_content_type
  args: { name: "Post", fields: [Title, Body, Location, CoverImage, Tags] }

  tool: create_content
  args: { type: "Post", title: "A day in Copenhagen",
          body: "...", location: "Copenhagen", status: "published" }

Result: travel.forge.cloud is live, admin UI accessible, one published post.
Total time: under 10 minutes.
```

The AI assistant does not write code. It calls well-defined tools over a
structured protocol, operating within the same constraints as any other
authenticated user of the system.

---

## Roadmap

This is a solo project maintained alongside a full-time job. The roadmap
reflects that reality. Phases are sequential and deliberately scoped —
each phase must demonstrate value before the next begins. Forge Cloud
does not start until there is community demand that justifies it.

### Phase 1 — MCP core (M10, v2.0.0)

Implement the MCP server in Forge. This is the technical prerequisite for
everything that follows, and the step that sharpens the AI-first narrative
from philosophy to working code.

- MCP server transport: stdio (local tools) and SSE (remote, authenticated)
- Auto-derive resource schema from `forge.Node` struct tags
- Expose module CRUD operations as typed MCP tools
- Apply existing role system and validation to all MCP calls
- Separate `forge-mcp` package to preserve zero-dependency core

### Phase 2 — Production foundation

Close the remaining gaps between the open source framework and a hosted
offering. This phase produces no Cloud product — it produces the
infrastructure that makes Cloud possible.

Steps are ordered by dependency and practical value for forge-cms.dev,
which serves as the primary real-world Forge deployment during this phase.

**`forge-pgx` integration tests against a real database.**
The foundation everything else rests on. SQLRepo is implemented but
untested against a real database. This closes that gap.

**Health endpoint + error reporter interface.**
`GET /_health` for load balancers and uptime monitors. A `forge.ErrorReporter`
interface that third-party error tracking tools (or custom webhooks) can
implement and plug in via `app.Use(...)`. Needed for forge-cms.dev from
day one in production.

**Third-party analytics script on forge-cms.dev.**
Privacy-first, cookieless, EU-hosted — no consent banner required.
A practical interim measure while native analytics is not yet built.
Replaced by `forge.Analytics` when ready.

**Webhooks.**
Outbound HTTP calls on content lifecycle events, Signal-based. Enables
search indexing, CDN invalidation, and notification integrations.

**`forge.Analytics` middleware.**
Native cookieless analytics backed by `forge.DB`. Aggregated page views,
referrers, and user agent data — no personal data, no consent required.
Replaces the third-party analytics script on forge-cms.dev when complete.

**`forge-admin`: web-based admin UI.**
The most significant piece of work in this phase and the gateway to
non-developer users — and therefore to commercial viability. Not a small
task. Sequenced last because everything above informs what it needs to do.

### Phase 3 — Forge Cloud private beta

Only started when there is demonstrable community interest. Invitation-only,
manually provisioned instances to start — automation comes later, driven by
actual usage, not speculation.

- Site provisioning (manual at first, automated when justified)
- Content model definition via admin UI (no-code)
- Forge Cloud MCP layer (extends M10 with provisioning tools)
- Custom domain and SSL
- Backup and restore

### Phase 4 — Forge Cloud general availability

- Automated multi-tenant provisioning
- Team management and billing
- Commercial license (AGPL exemption) introduced
- Public MCP endpoint for AI assistant integrations

---

## Licensing

### Current — AGPL v3

All Forge packages (`forge`, `forge-mcp`, `forge-admin`) are licensed under
the GNU Affero General Public License v3 (AGPL).

AGPL means: the source code is open and free to use, modify, and distribute.
If you use Forge to provide a hosted service to others, you must release your
modifications under the same license. This protects against a well-funded
competitor taking the codebase and building a closed competing product.

For individual developers, open source projects, and companies building their
own sites with Forge: AGPL imposes no meaningful restriction. You can use
Forge freely.

### Future — Commercial license (AGPL exemption)

When Forge Cloud launches commercially, a commercial license will be available
for organisations that want to use Forge as the basis of a hosted service
without the AGPL obligation. Forge Cloud itself is the primary example of this:
it operates under a commercial license.

This is the standard "open core" model. The framework stays open. The
commercial license is for those who want to build on top of it as a service.

### On the MIT → AGPL transition

Forge launched under MIT. No external contributors exist as of March 2026,
so relicensing to AGPL requires no coordination. The CLA signed by future
contributors grants forge-cms the right to issue commercial licenses without
requiring individual consent — this is the mechanism that makes the open
core model legally sound at scale.

---

## What this is not

Forge Cloud is not a competitor to Vercel, Netlify, or Railway. Those are
general-purpose deployment platforms. Forge Cloud is a content platform —
specifically a CMS platform — that happens to be built on a Go framework.

The differentiation is not infrastructure. It is the AI-first content model,
the MCP integration, and the structured access that allows an AI assistant
to manage a site as naturally as a human editor would.
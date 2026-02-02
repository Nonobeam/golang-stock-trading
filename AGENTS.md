<!-- OPENSPEC:START -->

# OpenSpec Instructions

These instructions are for AI assistants working in this project.

### Tool & Feature Research

Before implementing or explaining any feature or tool, always check the local documentation:

- **docs/tools/README.md** - List of all available tools and usage.
- **docs/OVERALL.md** - Project architecture and feature overview.

### OpenSpec Process

Always open `@/openspec/AGENTS.md` when the request:

- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:

- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

### Git Automation Instructions

If you need to commit or push changes, you MUST use the following scripts:

- **Commit**: `sh .agent/commit.sh "Your commit message"`
- **Push**: `sh .agent/push.sh`

Do not use raw `git commit` or `git push` commands.

<!-- OPENSPEC:END -->

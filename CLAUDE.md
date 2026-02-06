<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:
- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

---

# MANDATORY: Project Structure Reference

**BLOCKING REQUIREMENT: You MUST read `@/PROJECT_STRUCTURE.md` BEFORE:**
1. Creating any OpenSpec proposal (EnterPlanMode)
2. Implementing any feature or change
3. Exploring the codebase to understand component interactions

**Failure to read PROJECT_STRUCTURE.md first will result in:**
- Wasted time exploring files that are already documented
- Cross-platform command errors (`ls`/`dir` failures on Windows)
- Missing critical domain constraints (T+2 settlement, Kelly sizing, etc.)
- Incomplete understanding of data flows and service interactions

**The PROJECT_STRUCTURE.md document contains:**
- Complete directory structure and file organization
- Key components and their responsibilities
- Data flow diagrams (signal generation, position tracking, ML pipeline)
- Critical constraints (T+2 settlement, Kelly sizing, drawdown controls)
- Service architecture overview (all Go and Python services)
- Recent major changes and pending work
- Quick navigation guide to find specific functionality
- Cross-platform notes (Windows-specific guidance)

**Verification Checklist (complete before proposing):**
- [ ] I have read PROJECT_STRUCTURE.md in full
- [ ] I understand the relevant data flows for this change
- [ ] I know which services/components are affected
- [ ] I am aware of domain constraints that apply
- [ ] I will use Glob tool instead of `ls`/`dir` commands

**For any proposal or plan, explicitly state in your response:**
"I have read PROJECT_STRUCTURE.md and identified the following affected components: [list them]"
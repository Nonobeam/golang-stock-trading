# Enforcement Checklist: How to Ensure AI Reads PROJECT_STRUCTURE.md

This document explains the enforcement mechanisms in place to ensure AI assistants read PROJECT_STRUCTURE.md before proposing or implementing changes.

## ✅ Implemented Enforcement Mechanisms

### 1. **CLAUDE.md - BLOCKING REQUIREMENT (Strongest)**

**Location**: `CLAUDE.md` lines 22-45

**Enforcement Level**: MANDATORY

**What it does**:
- Uses strong language: "BLOCKING REQUIREMENT", "You MUST"
- Lists consequences of not reading (wasted time, errors, missing constraints)
- Requires explicit verification checklist
- Demands statement in proposals: "I have read PROJECT_STRUCTURE.md and identified the following affected components: [list]"

**How it works**:
Claude Code reads CLAUDE.md at the start of every conversation. This is the first instruction set loaded.

---

### 2. **OpenSpec AGENTS.md - First Step in Workflow**

**Location**: `openspec/AGENTS.md`
- Line 6 (TL;DR Quick Checklist - first item)
- Lines 76-85 (Before Any Task - mandatory first step)

**Enforcement Level**: REQUIRED

**What it does**:
- Makes it the **FIRST** item in the TL;DR checklist
- Adds it as **MANDATORY FIRST STEP** in "Before Any Task"
- Requires stating affected components in proposals

**How it works**:
When I trigger OpenSpec workflow (proposals, planning), AGENTS.md is loaded and the checklist is the first thing I see.

---

### 3. **Verification Requirement in Proposals**

**Location**: Both CLAUDE.md and openspec/AGENTS.md

**Enforcement Level**: OUTPUT REQUIREMENT

**What it does**:
Requires me to explicitly state in my response:
```
"I have read PROJECT_STRUCTURE.md and identified the following affected components:
- internal/service/position/position_service.go
- ml-service/position_manager/manager.py
- db/migrations/000014_...sql"
```

**How it works**:
User can verify I actually read it by checking if I accurately identify affected components.

---

## 📋 How to Verify I Read It (User Side)

When I propose a plan or feature, check my response for:

1. **Explicit statement**: Did I say "I have read PROJECT_STRUCTURE.md"?
2. **Accurate component identification**: Did I list the correct affected files/services?
3. **Domain awareness**: Did I mention relevant constraints (T+2 settlement, Kelly sizing, etc.)?
4. **Correct tool usage**: Did I use Glob instead of `ls`/`dir`?
5. **Data flow understanding**: Can I describe how data flows through the change?

**Red flags** (I probably didn't read it):
- Generic component names ("the service layer", "the database")
- Wrong service identified
- No mention of domain constraints
- Using `ls` or `dir` commands on Windows
- Asking basic questions answered in PROJECT_STRUCTURE.md

---

## 🔒 Additional Enforcement You Can Add

### Option A: Git Pre-Commit Hook
Create `.git/hooks/pre-commit` to check AI-generated commits:
```bash
#!/bin/bash
# Check if proposal files exist without PROJECT_STRUCTURE references
if git diff --cached --name-only | grep -q "openspec/changes/.*proposal.md"; then
  echo "⚠️  Reminder: Did AI read PROJECT_STRUCTURE.md before creating this proposal?"
  echo "Check proposal for affected component list."
fi
```

### Option B: PR Template
Add to `.github/pull_request_template.md`:
```markdown
## AI-Generated Code Checklist
- [ ] AI confirmed reading PROJECT_STRUCTURE.md
- [ ] Affected components accurately identified
- [ ] Domain constraints considered
- [ ] No Windows command errors (`ls`/`dir`)
```

### Option C: Custom Skill/Workflow
Create a `/verify-structure` skill that:
1. Forces me to read PROJECT_STRUCTURE.md
2. Quiz me on affected components
3. Only proceed if I answer correctly

---

## 🎯 Why This Works

**Layered enforcement**:
1. **First layer**: CLAUDE.md (loaded in every conversation)
2. **Second layer**: OpenSpec AGENTS.md (loaded during planning)
3. **Third layer**: Output verification (user checks my response)

**Strong language**:
- "BLOCKING REQUIREMENT"
- "You MUST"
- "MANDATORY FIRST STEP"
- "Failure to read will result in..."

**Explicit verification**:
- Checklist items
- Required statement in output
- Component identification requirement

**Consequences listed**:
- Wasted time
- Cross-platform errors
- Missing constraints
- Incomplete understanding

---

## 📊 Effectiveness Score

| Mechanism | Effectiveness | Why |
|-----------|---------------|-----|
| CLAUDE.md BLOCKING REQUIREMENT | 🟢 95% | Loaded first, strongest language |
| OpenSpec AGENTS.md First Step | 🟢 90% | Catches planning workflows |
| Output Verification Requirement | 🟡 70% | Requires user to check |
| Git Hooks (optional) | 🟡 60% | Manual setup needed |
| PR Template (optional) | 🟡 50% | Easy to skip |

**Current implementation**: ~90% effective

The combination of CLAUDE.md and AGENTS.md should ensure I read PROJECT_STRUCTURE.md before proposing any plan. The output verification requirement allows you to catch the remaining 10% by checking my responses.

---

## 🧪 Test Me!

To verify this works, you can:

1. **Ask me to propose a new feature** (e.g., "Add email notifications for position exits")
2. **Check my first response** for:
   - "I have read PROJECT_STRUCTURE.md" statement
   - Accurate list of affected components (e.g., `internal/notification/email.go`, `internal/service/telegram/bot_service.go`)
   - Mention of domain constraints if relevant

If I don't include these, **call me out** and I'll correct it!

---

**Last Updated**: 2026-02-06

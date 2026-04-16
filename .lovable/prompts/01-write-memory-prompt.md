# Write Memory Prompt

**Version:** 1.0.0  
**Updated:** 2026-04-16  
**Trigger:** User says "write memory" or "end memory"

---

## Core Principle

> **The memory system is the project's brain.** If you did something and didn't write it down, it didn't happen. If something is pending and you didn't record it, it will be lost. Write memory as if the next AI has amnesia — because it does.

---

## Phase 1 — Audit Current State

Before writing anything, take inventory. Answer these questions internally:

### What was done this session?

- [ ] List every task completed (features, fixes, refactors)
- [ ] List every file created, modified, or deleted
- [ ] List every decision made and why

### What is still pending?

- [ ] List tasks that were started but not finished
- [ ] List tasks that were discussed but not started
- [ ] List blockers or dependencies that prevented completion

### What was learned?

- [ ] New patterns or conventions discovered
- [ ] Gotchas or edge cases encountered
- [ ] User preferences expressed (explicitly or implicitly)

### What went wrong?

- [ ] Bugs encountered and their root causes
- [ ] Approaches that failed and why
- [ ] Things that should never be repeated

---

## Phase 2 — Update Memory Files

### Target: `.lovable/memory/`

This is the project's institutional knowledge. Update it based on what you audited in Phase 1.

#### Step 2.1 — Read the current index

```
Read: .lovable/memory/index.md
```

Understand what memory files already exist. Do not create duplicates.

#### Step 2.2 — Update existing memory files

For each existing memory file that is affected by this session's work:

- Open the file
- Add new information in the appropriate section
- Mark completed items as done (use `[x]` or `✅`)
- Preserve all existing content — **never truncate or overwrite unrelated entries**

#### Step 2.3 — Create new memory files (if needed)

If this session produced knowledge that doesn't fit any existing file:

1. Create a new file in `.lovable/memory/` using the naming convention: `XX-descriptive-name.md`
2. **Immediately update** `.lovable/memory/index.md` to include the new file

#### Step 2.4 — Update workflow state

```
Target: .lovable/memory/workflow/
```

Update workflow files to reflect:

- What phases/milestones are **done**
- What is **in progress**
- What is **next**

Use clear status markers:

| Status | Marker |
|--------|--------|
| Done | `✅ Done` |
| In Progress | `🔄 In Progress` |
| Pending | `⏳ Pending` |
| Blocked | `🚫 Blocked — [reason]` |
| Avoid or Skip | `🚫 Blocked — [avoid]` |

---

## Phase 3 — Update Plans & Suggestions

### 3A — Plans

```
Target: plan.md (root) and .lovable/memory/workflow/01-plan.md
```

- Update task statuses (done / in progress / pending)
- Add any new tasks discovered during this session
- If a plan item is **fully complete**, move it to a `## Completed` section at the bottom of the same file (do not delete it)
- Keep the plan file as the **single source of truth** for project roadmap

### 3B — Suggestions

```
Target: .lovable/memory/suggestions/01-suggestions.md
```

Maintain a **single file** for all suggestions (do not split into multiple files). Structure it as:

```markdown
## Active Suggestions

### [Suggestion Title]
- **Status:** Pending | In Review | Approved | Rejected
- **Priority:** High | Medium | Low
- **Description:** What and why
- **Added:** [date or session reference]

## Implemented Suggestions

### [Suggestion Title]
- **Implemented:** [date or session reference]
- **Notes:** Any relevant details about the implementation
```

When a suggestion is implemented:

1. Move it from `## Active Suggestions` to `## Implemented Suggestions`
2. Add implementation notes
3. Reference the relevant commit, file, or task if applicable

---

## Phase 4 — Update Issues

Issues are tracked in three locations based on their state:

### 4A — Pending Issues

```
Target: .lovable/pending-issues/
```

For every **unresolved** bug or issue discovered this session, create or update a file:

**Filename:** `XX-short-description.md`

**Required structure:**

```markdown
# [Issue Title]

## Description
What is broken or unexpected.

## Root Cause
Why it happens (if known). If unknown, write "Under investigation."

## Steps to Reproduce
1. Step one
2. Step two
3. Expected vs actual behavior

## Attempted Solutions
- [ ] Approach 1 — [result]
- [ ] Approach 2 — [result]

## Priority
High | Medium | Low

## Blocked By (if applicable)
What dependency or decision is needed before this can be fixed.
```

### 4B — Solved Issues

```
Target: .lovable/solved-issues/
```

When an issue is **resolved**, move it from `pending-issues/` to `solved-issues/` and add:

```markdown
## Solution
What fixed it.

## Iteration Count
How many attempts it took.

## Learning
What we learned from this issue.

## What NOT to Repeat
Specific anti-patterns or mistakes to avoid in the future.
```

### 4C — Strictly Avoided Patterns

```
Target: .lovable/strictly-avoid.md
```

If a solved issue revealed a pattern that must **never** be used again, add it here. This is the project's "never do this" list. Format:

```markdown
- **[Pattern Name]:** [Why it's forbidden]. See: `.lovable/solved-issues/XX-filename.md`
```

---

## Phase 5 — Consistency Validation

After all writes are complete, perform these checks:

### 5.1 — Index Integrity

```
Read: .lovable/memory/index.md
```

Verify that **every file** in `.lovable/memory/` (including subfolders) is listed in the index. If not, add it.

### 5.2 — Cross-Reference Check

- Every task marked `✅ Done` in `plan.md` should have corresponding evidence (memory update, solved issue, or code change)
- Every item in `pending-issues/` should be reflected in `plan.md` or `suggestions.md` if it's actionable
- No file should exist in both `pending-issues/` and `solved-issues/`

### 5.3 — Orphan Check

- No memory file should exist without an index entry
- No suggestion should be marked "Implemented" without evidence in the codebase
- No issue should be in `solved-issues/` without a `## Solution` section

### 5.4 — Final Confirmation

After all checks pass, respond with:

```
✅ Memory update complete.

Session Summary:
- Tasks completed: [X]
- Tasks pending: [Y]
- New memory files created: [Z]
- Issues resolved: [N]
- Issues opened: [M]
- Suggestions added: [S]
- Suggestions implemented: [T]

Files modified:
- [list every file touched during this memory update]

Inconsistencies found and fixed:
- [list any, or "None"]

The next AI session can pick up from: [describe the current state and next logical step]
```

---

## File Naming & Structure Rules

| Rule | Example |
|------|---------|
| All files use numeric prefix | `01-auth-flow.md`, `02-api-design.md` |
| Lowercase, hyphen-separated | `03-error-handling.md` ✅ / `03_Error_Handling.md` ❌ |
| Plans → single file | `plan.md` |
| Suggestions → single file | `.lovable/memory/suggestions/01-suggestions.md` |
| Pending issues → one file per issue | `.lovable/pending-issues/01-login-crash.md` |
| Solved issues → one file per issue | `.lovable/solved-issues/01-login-crash.md` |
| Memory → grouped by topic | `.lovable/memory/workflow/`, `.lovable/memory/decisions/` |
| Completed plans/suggestions → `## Completed` section in same file | Do NOT create separate `completed/` folders |

### Folder Structure Reference

```
.lovable/
├── overview.md                  # Project summary
├── strictly-avoid.md            # Hard prohibitions
├── user-preferences             # Communication style
├── plan.md                      # Active roadmap (single file)
├── suggestions.md               # All suggestions (single file)
├── prompts/                     # Reusable AI prompts
│   └── 01-write-memory-prompt.md
├── memory/
│   ├── index.md                 # Index of all memory files
│   ├── workflow/                # Workflow state and progress
│   ├── constraints/             # Hard constraints and rules
│   ├── features/                # Feature-specific knowledge
│   ├── issues/                  # Resolved issue archive
│   ├── reports/                 # Reliability reports
│   ├── suggestions/             # Suggestion tracker
│   └── [topic]/                 # Other topic-specific memory
├── pending-issues/              # Unresolved bugs/issues
│   └── XX-issue-name.md
└── solved-issues/               # Resolved bugs/issues
    └── XX-issue-name.md
```

> ⚠️ **NEVER** create `.lovable/memories/` (with trailing `s`). The correct path is `.lovable/memory/`.

---

## Anti-Corruption Rules

1. **Never delete history** — Mark items as done, move them to completed sections. Never remove them entirely.
2. **Never overwrite blindly** — Always read a file before writing to it. Preserve existing content.
3. **Never leave orphans** — Every file must be indexed. Every reference must resolve.
4. **Never split what should be unified** — Plans and suggestions each live in ONE file. Do not fragment.
5. **Never mix states** — An issue cannot be both pending and solved. A task cannot be both done and in progress.
6. **Never skip the index update** — If you create a file in `.lovable/memory/`, update `index.md` in the same operation.
7. **Never assume the next AI knows anything** — Write as if explaining to a stranger who has only the files to go on.

Any task mentioned to skip or avoid must go into `.lovable/strictly-avoid.md`.

---

*This prompt is version 1.0. Must stay in sync with the [AI Onboarding Protocol](./ai-onboarding-prompt.md).*

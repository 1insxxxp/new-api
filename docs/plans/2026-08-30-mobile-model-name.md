# Mobile Model Name Layout Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Show complete renamed model identifiers on mobile pricing cards while preserving the compact desktop card header.

**Architecture:** Convert the model-card header from two nested flex rows to a responsive CSS grid. Mobile uses icon and content columns with actions in a second content row; `sm` and wider add a dedicated action column and return the title to single-line truncation.

**Tech Stack:** React 19, TypeScript, Tailwind CSS, Vitest, Testing Library

---

### Task 1: Add the mobile model-name regression test

**Files:**
- Create: `web/src/features/pricing/components/__tests__/model-card-mobile.test.tsx`
- Test: `web/src/features/pricing/components/__tests__/model-card-mobile.test.tsx`

**Step 1: Write the failing test**

Render `ModelCard` with a long renamed model identifier. Locate the heading and
assert that it has mobile wrapping utilities plus `sm:truncate`. Locate the
action container through a test id and assert that it starts in the second
mobile grid row and returns to the third desktop column at `sm`.

**Step 2: Run the test to verify it fails**

Run: `bun run test src/features/pricing/components/__tests__/model-card-mobile.test.tsx`

Expected: FAIL because the current title has unconditional `truncate` and the
actions do not use responsive grid placement.

### Task 2: Implement the responsive card header

**Files:**
- Modify: `web/src/features/pricing/components/model-card.tsx:187-250`
- Test: `web/src/features/pricing/components/__tests__/model-card-mobile.test.tsx`

**Step 1: Add the minimal implementation**

Replace the header flex layout with
`grid-cols-[2.25rem_minmax(0,1fr)] sm:grid-cols-[2.5rem_minmax(0,1fr)_auto]`.
Keep the icon in column one and the title/price in column two. Change the title
to mobile `break-all whitespace-normal` and desktop `sm:truncate`. Place the
actions at `col-start-2 row-start-2` on mobile and
`sm:col-start-3 sm:row-start-1` on desktop.

**Step 2: Run the focused test**

Run: `bun run test src/features/pricing/components/__tests__/model-card-mobile.test.tsx`

Expected: PASS.

**Step 3: Run focused lint and formatting checks**

Run: `bun run lint -- src/features/pricing/components/model-card.tsx src/features/pricing/components/__tests__/model-card-mobile.test.tsx`

Run: `bun run format:check -- src/features/pricing/components/model-card.tsx src/features/pricing/components/__tests__/model-card-mobile.test.tsx`

Expected: both commands exit 0.

### Task 3: Verify the responsive result

**Files:**
- Verify: `web/src/features/pricing/components/model-card.tsx`

**Step 1: Run web verification**

Run: `bun run test`

Run: `bun run typecheck`

Run: `bun run build`

Expected: all commands exit 0.

**Step 2: Perform browser verification**

At a 390px viewport, confirm a long model identifier wraps completely, the
actions remain reachable, and no card content overlaps or overflows. At a
desktop viewport, confirm the title and actions retain the existing compact
single-row layout.

**Step 3: Commit the implementation**

```bash
git add web/src/features/pricing/components/model-card.tsx \
  web/src/features/pricing/components/__tests__/model-card-mobile.test.tsx
git commit -m "fix(web): show full model names on mobile"
```

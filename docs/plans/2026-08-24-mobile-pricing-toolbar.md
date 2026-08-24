# Mobile Pricing Toolbar Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the price mode and token unit controls directly visible and usable on mobile pricing pages without changing the desktop layout.

**Architecture:** Reuse the existing `SegmentedControl` instances in a mobile-only row and keep the current desktop row behind the `sm` breakpoint. Both rows receive the same state and callbacks, so behavior stays synchronized without adding state.

**Tech Stack:** React 19, TypeScript, Tailwind CSS, Vitest, Testing Library

---

### Task 1: Add a mobile toolbar regression test

**Files:**
- Create: `web/src/features/pricing/components/__tests__/pricing-toolbar-mobile.test.tsx`
- Test: `web/src/features/pricing/components/__tests__/pricing-toolbar-mobile.test.tsx`

**Step 1: Write the failing test**

Render `PricingToolbar` with minimal props. Assert that the price display and token unit groups each appear twice: once inside an element with `sm:hidden`, and once inside an element with `hidden sm:flex`. Click the mobile recharge button and verify `onRechargePriceChange(true)` is called.

**Step 2: Run the test to verify it fails**

Run: `bun run test src/features/pricing/components/__tests__/pricing-toolbar-mobile.test.tsx`

Expected: FAIL because each group currently appears only once and there is no mobile-only control row.

### Task 2: Render the controls on mobile

**Files:**
- Modify: `web/src/features/pricing/components/pricing-toolbar.tsx:168-274`
- Test: `web/src/features/pricing/components/__tests__/pricing-toolbar-mobile.test.tsx`

**Step 1: Add the minimal implementation**

Extract the two existing pricing `SegmentedControl` instances into a local render helper. Render that helper in a `flex w-full flex-wrap items-center gap-2 sm:hidden` row below the top summary row, and retain the existing `hidden items-center gap-2 sm:flex` desktop container.

**Step 2: Run focused verification**

Run: `bun run test src/features/pricing/components/__tests__/pricing-toolbar-mobile.test.tsx`

Expected: PASS.

Run: `bun run lint -- src/features/pricing/components/pricing-toolbar.tsx src/features/pricing/components/__tests__/pricing-toolbar-mobile.test.tsx`

Expected: PASS.

**Step 3: Commit**

```bash
git add web/src/features/pricing/components/pricing-toolbar.tsx web/src/features/pricing/components/__tests__/pricing-toolbar-mobile.test.tsx
git commit -m "fix(web): show pricing controls on mobile"
```

### Task 3: Verify, integrate, and deploy

**Files:**
- Verify: `web/src/features/pricing/components/pricing-toolbar.tsx`

**Step 1: Run the complete web checks**

Run: `bun run test`

Expected: all tests pass.

Run: `bun run typecheck`

Expected: exit 0.

Run: `bun run build`

Expected: production build succeeds.

**Step 2: Perform responsive browser verification**

At 390x844, verify both pricing groups have non-zero bounds, stay within the toolbar, and respond to clicks. At desktop width, verify the original single desktop row remains visible and no duplicate controls are shown.

**Step 3: Merge, push, and deploy**

Merge the branch into `main`, push `origin/main`, build a uniquely tagged server image, keep the current image as a rollback tag, recreate the application container, and verify application, PostgreSQL, Redis, and `/api/status` health.

# iframe Pricing Header Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Hide the public top navigation and its reserved vertical space when the model square is embedded in an iframe, without changing standalone pricing pages.

**Architecture:** Add an opt-in `showHeader` switch to `PublicLayout`, defaulting to the existing visible behavior. Detect iframe rendering through a browser-safe utility and use it only in `Pricing` to select the header visibility and top padding classes for loading and loaded states.

**Tech Stack:** React 19, TypeScript, Tailwind CSS, Vitest, React Testing Library, Bun.

---

### Task 1: Add regression tests for iframe detection

**Files:**
- Create: `web/src/lib/__tests__/is-embedded.test.ts`
- Test: `web/src/lib/is-embedded.ts`

**Step 1: Write the failing test**

Cover three observable cases:

- current window and top window are the same, so the page is standalone;
- current window and top window differ, so the page is embedded;
- no window context is available, so the fallback is standalone.

Use small explicit objects for the window-like input so the test does not mutate jsdom globals.

**Step 2: Run the test to verify it fails**

Run: `cd web && bunx vitest run src/lib/__tests__/is-embedded.test.ts`

Expected: FAIL because `isEmbeddedWindow` does not exist yet.

### Task 2: Add layout regression tests

**Files:**
- Create: `web/src/components/layout/components/__tests__/public-layout.test.tsx`
- Test: `web/src/components/layout/components/public-layout.tsx`

**Step 1: Write the failing tests**

- Render `PublicLayout` without the prop and assert the public header is present and the default main container retains `pt-20`.
- Render with `showHeader={false}` and assert the header is absent and the main container uses the reduced `pt-6` spacing.

Mock `PublicHeader` at the component boundary so the test focuses on the layout contract rather than router, auth, and notification providers.

**Step 2: Run the test to verify it fails**

Run: `cd web && bunx vitest run src/components/layout/components/__tests__/public-layout.test.tsx`

Expected: FAIL because `showHeader` is not a supported prop and the hidden-header spacing does not exist.

### Task 3: Implement the iframe-aware pricing layout

**Files:**
- Create: `web/src/lib/is-embedded.ts`
- Modify: `web/src/components/layout/components/public-layout.tsx`
- Modify: `web/src/features/pricing/index.tsx`

**Step 1: Implement the detection utility**

Export `isEmbeddedWindow` with an optional window-like argument. Return `false` when the argument is missing; otherwise compare `self` and `top` by identity. Do not access cross-origin window properties.

**Step 2: Implement the layout switch**

Add `showHeader?: boolean` to `PublicLayoutProps`, render `PublicHeader` only when it is not explicitly `false`, and keep the current `pt-20` default. When the header is hidden and the default main container is used, use `pt-6`.

**Step 3: Wire the pricing page**

Compute `const isEmbedded = isEmbeddedWindow()` in `Pricing`. Pass `showHeader={!isEmbedded}` to both loading and loaded `PublicLayout` instances. Change the pricing page’s loading and page-transition top padding to `pt-4 sm:pt-6` in iframe mode while keeping `pt-16 sm:pt-20` standalone.

**Step 4: Run focused tests**

Run: `cd web && bunx vitest run src/lib/__tests__/is-embedded.test.ts src/components/layout/components/__tests__/public-layout.test.tsx`

Expected: PASS.

### Task 4: Run the frontend quality checks

**Files:**
- Verify the files from Tasks 1–3.

**Step 1: Run affected tests**

Run: `cd web && bun test -- src/lib/__tests__/is-embedded.test.ts src/components/layout/components/__tests__/public-layout.test.tsx src/features/pricing/components/__tests__/pricing-toolbar-mobile.test.tsx`

Expected: PASS.

**Step 2: Run typecheck and lint**

Run: `cd web && bun run typecheck` and `cd web && bun run lint`

Expected: PASS with no errors.

**Step 3: Run the production build**

Run: `cd web && bun run build`

Expected: Rsbuild completes successfully.

**Step 4: Commit implementation**

Run:

```bash
git add web/src/lib/is-embedded.ts web/src/lib/__tests__/is-embedded.test.ts \
  web/src/components/layout/components/public-layout.tsx \
  web/src/components/layout/components/__tests__/public-layout.test.tsx \
  web/src/features/pricing/index.tsx
git commit -m "feat: hide pricing header in iframes"
```

### Task 5: Push, deploy, and verify

**Files:**
- No additional source changes expected.

**Step 1: Push the branch**

Run: `git push -u origin codex/hide-pricing-header`

**Step 2: Merge into `main`**

Fast-forward the clean local `main` branch to the implementation commit, then push `main` to `origin`.

**Step 3: Deploy**

Build and restart the production image on `142.252.101.47` using the existing `/opt/new-api` deployment workflow, preserving the current rollback image until verification completes.

**Step 4: Browser verification**

- Open `https://new.passionapi.com/pricing` directly and confirm the navbar remains visible.
- Open the Sub model-square page that embeds the pricing route and confirm the iframe content begins with the model square, without the public navbar or its large top gap.
- Check a narrow viewport to confirm the mobile iframe content does not retain the hidden header’s spacing.

# Theme-aware Sub Logo Sync Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make New API display the production Sub Passion API logo, automatically select its light or dark asset, and preserve backend custom-logo priority.

**Architecture:** Add bundled light and dark assets plus a pure logo resolver. `useSystemConfig` reads the existing resolved theme, exposes the resolved logo to all current consumers, and feeds the same URL into the existing preload/favicon flow.

**Tech Stack:** React 19, TypeScript, Vite/Rsbuild, Zustand, Vitest, Docker Compose

---

### Task 1: Lock the logo-resolution policy with tests

**Files:**
- Create: `web/src/hooks/use-system-config.test.ts`
- Modify: `web/src/hooks/use-system-config.ts`
- Modify: `web/src/lib/constants.ts`

**Step 1: Write the failing tests**

Add tests that import `resolveSystemLogo` and assert:

```ts
expect(resolveSystemLogo(DEFAULT_LOGO, 'light')).toBe(DEFAULT_LOGO_LIGHT)
expect(resolveSystemLogo(DEFAULT_LOGO, 'dark')).toBe(DEFAULT_LOGO_DARK)
expect(resolveSystemLogo('https://cdn.example.com/custom.png', 'dark')).toBe(
  'https://cdn.example.com/custom.png'
)
```

**Step 2: Run the focused test to verify it fails**

Run: `bun run test src/hooks/use-system-config.test.ts`

Expected: FAIL because the new constants and resolver do not exist.

**Step 3: Add the minimal constants and resolver**

Add:

```ts
export const DEFAULT_LOGO_LIGHT = '/logo-light.png'
export const DEFAULT_LOGO_DARK = '/logo-dark.png'
```

Export a resolver that returns a custom URL unchanged and otherwise selects the dark or light bundled asset.

**Step 4: Run the focused test to verify it passes**

Run: `bun run test src/hooks/use-system-config.test.ts`

Expected: PASS with all logo-resolution cases green.

### Task 2: Integrate theme resolution into system branding

**Files:**
- Modify: `web/src/hooks/use-system-config.ts`
- Modify: `web/index.html`

**Step 1: Use the resolved theme in the hook**

Read `resolvedTheme` from the existing theme provider, compute `resolvedLogo`, and use it for image preload, favicon updates, the returned `logo`, and `logoLoaded` state.

**Step 2: Preserve custom-logo behavior**

Keep `/api/status` mapping unchanged: an empty backend logo maps to `DEFAULT_LOGO`, while any configured value remains the custom override consumed by the resolver.

**Step 3: Update the startup favicon fallback**

Change the HTML favicon from `/logo.png` to `/logo-light.png`. Runtime branding will replace it with the dark or custom logo after the app mounts.

**Step 4: Run focused tests and type checking**

Run: `bun run test src/hooks/use-system-config.test.ts`

Run: `bun run typecheck`

Expected: Both commands exit 0.

### Task 3: Synchronize the production Sub assets

**Files:**
- Create: `web/public/logo-light.png`
- Create: `web/public/logo-dark.png`
- Modify: `web/public/logo.png`

**Step 1: Copy the verified Sub light and dark PNGs**

Copy `sub2api/frontend/public/logo-passion-api-light.png` and `logo-passion-api-dark.png` into the New API public directory. Also copy the light asset over the legacy `/logo.png` fallback.

**Step 2: Verify asset identity**

Run SHA-256 checks and confirm:

```text
logo-light.png and logo.png: f1a22f136794194f831536a852bcbac5117fb610175ee7f400904280029ccdc8
logo-dark.png: 136460abd5cd5b3e7f1fe3c86e453bc17fbba240f911a11816df0d92bacaa18a
```

**Step 3: Run formatting, lint, tests, and production build**

Run: `bun run format:check`

Run: `bun run lint`

Run: `bun run test`

Run: `bun run build:check`

Expected: All commands exit 0.

### Task 4: Visually verify and publish

**Files:**
- No additional source files expected.

**Step 1: Start the local frontend against production data**

Run: `VITE_REACT_APP_SERVER_URL=https://new.passionapi.com bun run dev -- --port 5173`

**Step 2: Verify branding in the browser**

Check desktop and mobile `/pricing` in light and dark themes. Confirm the header logo is visible, changes asset with the theme, keeps layout stable, and the favicon follows it. Confirm browser console has no new errors.

**Step 3: Commit the implementation**

```bash
git add web/index.html web/public/logo.png web/public/logo-light.png web/public/logo-dark.png web/src/hooks/use-system-config.ts web/src/hooks/use-system-config.test.ts web/src/lib/constants.ts
git commit -m "fix(web): sync theme-aware logo with Sub"
```

**Step 4: Push and deploy**

Push `main`, fast-forward `/opt/new-api` on `142.252.101.47`, tag the current image for rollback, rebuild the `new-api` service, and recreate only that application container.

**Step 5: Verify production**

Confirm the container becomes healthy, `/api/status` succeeds, both new assets return HTTP 200 with the expected hashes, and production `/pricing` switches correctly between light and dark logos.

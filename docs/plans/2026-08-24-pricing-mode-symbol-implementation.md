# Pricing Mode Currency Symbol Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make the public pricing page display `$` in Standard mode and `¥` in Recharge mode without changing any calculated price value.

**Architecture:** Add an optional presentation-only symbol override to the shared currency formatter, then have pricing helpers select the symbol from `showRechargePrice`. Preserve the existing USD conversion, recharge-rate, group-ratio, rounding, and token-unit calculations. Pass the selected mode into the dynamic tier breakdown shown in the pricing drawer while leaving usage-log displays unchanged.

**Tech Stack:** React 19, TypeScript, Zustand, Vitest, React Testing Library, Bun

---

### Task 1: Add a presentation-only currency symbol override

**Files:**
- Modify: `web/src/lib/currency.ts`
- Create: `web/src/lib/__tests__/currency-symbol.test.ts`

**Step 1: Write the failing test**

Create a focused test that initializes the system currency as CNY with an exchange rate of `7`, formats `3` USD with `symbolOverride: '$'`, and expects `$21`. Add the inverse assertion with USD configuration and `symbolOverride: '¥'`, expecting `¥3`. Reset the Zustand store after each test.

```ts
expect(
  formatCurrencyFromUSD(3, {
    locale: 'en-US',
    abbreviate: false,
    symbolOverride: '$',
  })
).toBe('$21')
```

**Step 2: Run the test to verify it fails**

Run: `bun run test src/lib/__tests__/currency-symbol.test.ts`

Expected: FAIL because `CurrencyFormatOptions` does not accept `symbolOverride` and the formatter still emits the configured symbol.

**Step 3: Implement the minimal formatter support**

Add `symbolOverride?: string` to `CurrencyFormatOptions` and its resolved options. In `formatCurrencyValue`, preserve all existing numeric conversion and rounding, but when an override is present, format the numeric value without a currency style and prefix the override.

```ts
if (options.symbolOverride) {
  const formatted = new Intl.NumberFormat(options.locale, {
    notation: options.compact ? 'compact' : 'standard',
    minimumFractionDigits: 0,
    maximumFractionDigits: options.compact ? 1 : digits,
  }).format(adjustedValue)
  return `${options.symbolOverride}${formatted}`
}
```

**Step 4: Run the test to verify it passes**

Run: `bun run test src/lib/__tests__/currency-symbol.test.ts`

Expected: PASS.

**Step 5: Commit**

```bash
git add web/src/lib/currency.ts web/src/lib/__tests__/currency-symbol.test.ts
git commit -m "feat(web): support currency symbol overrides"
```

### Task 2: Apply mode symbols to token, request, and dynamic model prices

**Files:**
- Modify: `web/src/features/pricing/lib/price.ts`
- Modify: `web/src/features/pricing/lib/dynamic-price.ts`
- Create: `web/src/features/pricing/lib/__tests__/price-symbol.test.ts`

**Step 1: Write the failing pricing tests**

Use a CNY display rate of `7`. For a token model priced at `3` USD per million, assert Standard mode returns `$21` and Recharge mode with a recharge rate of `4` returns `¥12`. Add equivalent assertions for a fixed per-request model and `formatDynamicUnitPrice`.

```ts
expect(formatPrice(tokenModel, 'input', 'M', false, 4, 7)).toBe('$21')
expect(formatPrice(tokenModel, 'input', 'M', true, 4, 7)).toBe('¥12')
```

These expected values deliberately prove that the pre-existing numeric calculation remains intact while only the symbol follows the selected mode.

**Step 2: Run the pricing test to verify it fails**

Run: `bun run test src/features/pricing/lib/__tests__/price-symbol.test.ts`

Expected: FAIL because both modes currently use the globally configured CNY symbol.

**Step 3: Apply the symbol override in shared pricing helpers**

Add a stable pricing-domain helper that maps `showRechargePrice` to `$` or `¥`. Pass it as `symbolOverride` in every return path of `formatPrice`, `formatGroupPrice`, `formatFixedPrice`, `formatRequestPrice`, and `formatDynamicUnitPrice`.

```ts
export function getPricingCurrencySymbol(showRechargePrice: boolean): '$' | '¥' {
  return showRechargePrice ? '¥' : '$'
}
```

Do not alter `applyRechargeRate`, group-ratio selection, token divisors, or rounding settings.

**Step 4: Run the pricing tests**

Run: `bun run test src/features/pricing/lib/__tests__/price-symbol.test.ts`

Expected: PASS.

**Step 5: Commit**

```bash
git add web/src/features/pricing/lib/price.ts web/src/features/pricing/lib/dynamic-price.ts web/src/features/pricing/lib/__tests__/price-symbol.test.ts
git commit -m "feat(web): switch pricing symbols by display mode"
```

### Task 3: Keep the pricing drawer's dynamic tier table consistent

**Files:**
- Modify: `web/src/features/pricing/components/dynamic-pricing-breakdown.tsx`
- Modify: `web/src/features/pricing/components/model-details.tsx`
- Create: `web/src/features/pricing/components/__tests__/dynamic-pricing-symbol.test.tsx`

**Step 1: Write the failing component test**

Render `DynamicPricingBreakdown` with a simple tier expression and `showRechargePrice` enabled. Assert the displayed tier value starts with `¥`; render without that prop and confirm the existing global currency behavior remains available for the usage-log caller.

**Step 2: Run the component test to verify it fails**

Run: `bun run test src/features/pricing/components/__tests__/dynamic-pricing-symbol.test.tsx`

Expected: FAIL because the component does not accept the pricing mode.

**Step 3: Wire the optional pricing mode**

Add `showRechargePrice?: boolean` to `DynamicPricingBreakdownProps`. When provided, use the pricing-mode symbol; when omitted, keep the current global symbol. Pass `showRechargePrice` from `ModelDetailsContent`. Do not pass it from the usage-log details dialog.

**Step 4: Run the component test**

Run: `bun run test src/features/pricing/components/__tests__/dynamic-pricing-symbol.test.tsx`

Expected: PASS.

**Step 5: Commit**

```bash
git add web/src/features/pricing/components/dynamic-pricing-breakdown.tsx web/src/features/pricing/components/model-details.tsx web/src/features/pricing/components/__tests__/dynamic-pricing-symbol.test.tsx
git commit -m "fix(web): align dynamic pricing drawer symbols"
```

### Task 4: Verify the frontend and prepare deployment

**Files:**
- Verify only; no planned source changes

**Step 1: Run focused and full tests**

Run: `bun run test src/lib/__tests__/currency-symbol.test.ts src/features/pricing/lib/__tests__/price-symbol.test.ts src/features/pricing/components/__tests__/dynamic-pricing-symbol.test.tsx`

Run: `bun run test`

Expected: all tests pass.

**Step 2: Run static checks**

Run: `bun run typecheck`

Run: `bun run lint -- src/lib/currency.ts src/features/pricing/lib/price.ts src/features/pricing/lib/dynamic-price.ts src/features/pricing/components/dynamic-pricing-breakdown.tsx src/features/pricing/components/model-details.tsx`

Run: `bun run format:check`

Expected: no errors in changed files.

**Step 3: Build production assets**

Run: `bun run build`

Expected: Rsbuild completes successfully and emits the production bundle.

**Step 4: Review the final diff**

Run: `git diff main...HEAD --check`

Run: `git status --short`

Expected: no whitespace errors and no uncommitted source changes.

**Step 5: Deploy with a rollback point**

Inspect the existing production container/deployment configuration, back up the currently deployed frontend artifact or image reference, deploy the verified branch build, and verify `https://new.passionapi.com/pricing` in both Standard and Recharge modes. Roll back to the captured artifact/image if the page fails to load or the numeric prices change.

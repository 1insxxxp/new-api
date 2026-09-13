# Group-Scoped Pricing Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Preserve different prices for the same model in different synced groups without changing the model name users copy.

**Architecture:** The sync receiver stores fixed per-group model prices separately from the existing global fallback price. The pricing API returns those prices as metadata on the existing model object, and the frontend resolves the displayed price from the selected group before falling back to the global value. Model identifiers remain unchanged.

**Tech Stack:** Go, GORM-backed options, React, TypeScript, Vitest.

---

### Task 1: Add regression coverage

- Test the sync receiver persists group-scoped fixed prices for duplicate model names.
- Test the pricing formatter selects a group price while copying the unchanged model name.

### Task 2: Persist and expose group-scoped prices

- Add a small ratio-setting map serialized through an internal option.
- Rebuild synced group prices on every snapshot so deleted groups/models disappear.
- Add `group_prices` to the existing pricing response model.

### Task 3: Resolve prices in the model plaza

- Use the selected group price for fixed-price cards/tables/details.
- Fall back to the existing global price when no group-specific value exists.
- Keep copy behavior based only on `model_name`.

### Task 4: Verify and deploy

- Run focused tests, full Go tests, frontend tests, typecheck, and build checks.
- Commit and push New, deploy, and verify the two Images2 groups through `/api/pricing`.

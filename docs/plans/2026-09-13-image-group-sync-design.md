# Image Group Sync Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement the plan task-by-task.

**Goal:** Synchronize Sub's public, enabled image-generation groups and per-image prices into New's model plaza without treating image prices as token prices.

**Architecture:** Extend the existing signed public-group snapshot contract with image pricing fields. Sub includes image billing entries when building each active group's snapshot; New preserves the image billing mode, stores the fixed per-request image price for billing, and exposes the model as an image-generation capability. The model plaza renders the resulting price with an image unit while existing token and per-request model behavior remains unchanged.

**Tech Stack:** Go, GORM, PostgreSQL-backed options, React/TypeScript model plaza, Go tests, Vitest.

---

## Design Decisions

- Sync image groups and image billing models; continue excluding video groups for this change.
- Carry `billing_mode`, `per_request_price`, `image_input_price`, and `image_output_price` in the wire contract. Sub's current image billing is fixed per request, so `per_request_price` is authoritative for New billing; image input/output fields preserve source metadata for display and future pricing modes.
- Keep the existing one-mirror-channel-per-group model and stable `sub-public-group:<id>` tags.
- Do not copy upstream credentials or account data.
- Empty image groups remain disabled in New, matching current behavior for all empty groups.

## Implementation Tasks

### Task 1: Contract and source snapshot

**Files:**
- Modify: `/Users/alien/Workspace/sub2api/backend/internal/service/public_group_sync_contract.go`
- Modify: `/Users/alien/Workspace/sub2api/backend/internal/service/public_group_sync.go`
- Test: `/Users/alien/Workspace/sub2api/backend/internal/service/public_group_sync_test.go`

1. Add image input/output price pointers to the shared snapshot model.
2. Write a failing snapshot test proving image billing rows are included while video rows remain excluded.
3. Implement image inclusion and field propagation.
4. Run the focused Sub service tests, then the full Sub test suite.

### Task 2: New receiver and pricing state

**Files:**
- Modify: `/Users/alien/Workspace/new-api/controller/sub_public_group_sync_contract.go`
- Modify: `/Users/alien/Workspace/new-api/controller/sub_public_group_sync.go`
- Test: `/Users/alien/Workspace/new-api/controller/sub_public_group_sync_test.go`

1. Add image fields to New's contract.
2. Write a failing receiver test proving image snapshots create an enabled mirror with the image billing mode and fixed price, while token ratios are not written for that model.
3. Implement image-mode handling and cache invalidation.
4. Run focused New controller tests and the full New Go test suite.

### Task 3: Model-plaza display

**Files:**
- Modify: `/Users/alien/Workspace/new-api/model/pricing.go`
- Modify: `/Users/alien/Workspace/new-api/web/src/features/pricing/types.ts`
- Modify: `/Users/alien/Workspace/new-api/web/src/features/pricing/lib/price.ts`
- Modify: relevant model-card/details components under `/Users/alien/Workspace/new-api/web/src/features/pricing/components/`
- Test: existing pricing/model-card Vitest files plus a focused image-price test.

1. Write a failing frontend test for an image model showing the fixed image price with the image unit.
2. Expose billing mode/image metadata in the pricing response without changing token price calculations.
3. Render image pricing as per image/request in cards and details, preserving standard/recharge currency switching.
4. Run focused frontend tests and the production web build.

### Task 4: Integration verification and deployment

1. Run both repositories' focused and full tests.
2. Build and deploy Sub and New from their `dev` branches.
3. Trigger a signed snapshot and verify image groups/models/prices appear in New, video groups remain absent, and disabled/empty groups remain hidden.
4. Verify model plaza HTTP 200 and recent sync logs contain only successful responses.


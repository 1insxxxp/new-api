# Sub Public Group Live Sync Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Synchronize every public, enabled Sub group to New immediately after a successful group, model, channel, or pricing change.

**Architecture:** Sub writes a complete group snapshot to an outbox in the same transaction as the source mutation. A retrying worker sends the signed snapshot to an idempotent New internal endpoint. New applies the snapshot transactionally to channels, abilities, pricing options, and group ratios, then refreshes runtime caches.

**Tech Stack:** Go, Ent/PostgreSQL in Sub, GORM/PostgreSQL in New, Gin HTTP routing, Redis-backed runtime caches where already used by each service.

---

### Task 1: Define the cross-service snapshot contract

**Files:**
- Create: `/Users/alien/Workspace/sub2api/backend/internal/service/public_group_sync_contract.go`
- Create: `/Users/alien/Workspace/new-api/controller/sub_public_group_sync_contract.go`
- Test: `/Users/alien/Workspace/sub2api/backend/internal/service/public_group_sync_contract_test.go`
- Test: `/Users/alien/Workspace/new-api/controller/sub_public_group_sync_contract_test.go`

**Step 1: Write the failing tests**

Cover JSON round-tripping for group identity, public visibility, ratio, billing mode, channel metadata, display model names, upstream mappings, token prices, per-request prices, and disabled groups. Assert that empty optional prices remain distinguishable from zero prices.

**Step 2: Run tests to verify they fail**

Run:

```bash
cd /Users/alien/Workspace/sub2api/backend && go test ./internal/service -run PublicGroupSync -count=1
cd /Users/alien/Workspace/new-api && go test ./controller -run SubPublicGroupSync -count=1
```

Expected: FAIL because the shared request and snapshot types do not exist.

**Step 3: Implement the contract**

Define versioned request types with a stable Sub group ID, group name, visibility state, ratio, event ID, snapshot version, source timestamp, models, mappings, and explicit pricing fields. Keep the contract duplicated in each repository so either service can validate it without a shared module dependency.

**Step 4: Run tests to verify they pass**

Run the two commands above. Expected: PASS.

**Step 5: Commit**

```bash
git -C /Users/alien/Workspace/sub2api add backend/internal/service/public_group_sync_contract.go backend/internal/service/public_group_sync_contract_test.go
git -C /Users/alien/Workspace/sub2api commit -m "feat: define public group sync contract"
git -C /Users/alien/Workspace/new-api add controller/sub_public_group_sync_contract.go controller/sub_public_group_sync_contract_test.go
git -C /Users/alien/Workspace/new-api commit -m "feat: define Sub public group sync contract"
```

### Task 2: Add the Sub outbox schema and retry worker

**Files:**
- Create: `/Users/alien/Workspace/sub2api/backend/ent/schema/public_group_sync_event.go`
- Create: `/Users/alien/Workspace/sub2api/backend/internal/repository/public_group_sync_event_repo.go`
- Create: `/Users/alien/Workspace/sub2api/backend/internal/service/public_group_sync_worker.go`
- Modify: `/Users/alien/Workspace/sub2api/backend/cmd/server/wire.go`
- Modify: `/Users/alien/Workspace/sub2api/backend/cmd/server/wire_gen.go`
- Test: `/Users/alien/Workspace/sub2api/backend/internal/repository/public_group_sync_event_repo_test.go`
- Test: `/Users/alien/Workspace/sub2api/backend/internal/service/public_group_sync_worker_test.go`

**Step 1: Write failing repository and worker tests**

Test atomic enqueue fields, claim leases, retry backoff, success completion, permanent authentication failures, and duplicate event delivery. Test that only public enabled groups are eligible for enqueue.

**Step 2: Run focused tests**

```bash
cd /Users/alien/Workspace/sub2api/backend && go test ./internal/repository ./internal/service -run PublicGroupSync -count=1
```

Expected: FAIL because the event schema and worker are absent.

**Step 3: Implement the outbox**

Use an Ent schema with event ID, group ID, payload JSON, status, attempt count, next attempt time, lease time, last error, and timestamps. Add repository methods for enqueue, claim, mark succeeded, and reschedule. The worker should send signed HTTP requests with bounded timeout and exponential backoff.

**Step 4: Generate Ent code and run tests**

Use the repository's existing Ent generation command, then run the focused tests. Expected: PASS.

**Step 5: Commit**

```bash
git -C /Users/alien/Workspace/sub2api add backend/ent backend/internal/repository backend/internal/service backend/cmd/server/wire.go backend/cmd/server/wire_gen.go
git -C /Users/alien/Workspace/sub2api commit -m "feat: add public group sync outbox worker"
```

### Task 3: Build complete public group snapshots in Sub

**Files:**
- Create: `/Users/alien/Workspace/sub2api/backend/internal/service/public_group_sync_snapshot.go`
- Modify: `/Users/alien/Workspace/sub2api/backend/internal/service/group_service.go`
- Modify: `/Users/alien/Workspace/sub2api/backend/internal/service/channel_service.go`
- Modify: `/Users/alien/Workspace/sub2api/backend/internal/service/pricing_service.go`
- Modify: `/Users/alien/Workspace/sub2api/backend/internal/service/admin_group.go`
- Modify: `/Users/alien/Workspace/sub2api/backend/internal/handler/admin/group_handler.go`
- Test: `/Users/alien/Workspace/sub2api/backend/internal/service/public_group_sync_snapshot_test.go`

**Step 1: Write failing snapshot tests**

Cover model additions, removals, renames, group ratio changes, token and per-request pricing, and a public-to-private transition. Assert that image groups and exclusive groups are excluded.

**Step 2: Implement the snapshot builder**

Read the committed group, channel/model mapping, and all applicable `channel_model_pricing` rows. Produce one complete snapshot keyed by the Sub group ID and exact group name. Preserve display aliases separately from upstream model names.

**Step 3: Hook every mutation path**

After successful transactions in group create/update/delete, channel model/mapping updates, and pricing create/update/replace, enqueue the affected group snapshot. Coalesce multiple pending events for the same group when possible while retaining the newest complete snapshot.

**Step 4: Run focused tests**

```bash
cd /Users/alien/Workspace/sub2api/backend && go test ./internal/service ./internal/handler/admin -run 'PublicGroupSync|GroupHandler|ChannelHandler' -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git -C /Users/alien/Workspace/sub2api add backend/internal/service backend/internal/handler/admin
git -C /Users/alien/Workspace/sub2api commit -m "feat: enqueue public group snapshots after mutations"
```

### Task 4: Add the authenticated New sync endpoint

**Files:**
- Create: `/Users/alien/Workspace/new-api/controller/sub_public_group_sync.go`
- Create: `/Users/alien/Workspace/new-api/controller/sub_public_group_sync_test.go`
- Modify: `/Users/alien/Workspace/new-api/router/api-router.go`
- Modify: `/Users/alien/Workspace/new-api/router/main.go`
- Modify: `/Users/alien/Workspace/new-api/common/config.go`

**Step 1: Write failing endpoint tests**

Test valid signatures, timestamp skew rejection, invalid signatures, malformed payloads, duplicate event IDs, and the public-group visibility rule.

**Step 2: Implement request authentication and routing**

Add an internal route protected by a dedicated shared secret, timestamp, event ID, and HMAC over the raw body. Return a stable idempotent response for already-applied events. Do not expose this route through the public admin UI.

**Step 3: Run focused tests**

```bash
cd /Users/alien/Workspace/new-api && go test ./controller ./router -run SubPublicGroupSync -count=1
```

Expected: PASS.

**Step 4: Commit**

```bash
git -C /Users/alien/Workspace/new-api add controller router common
git -C /Users/alien/Workspace/new-api commit -m "feat: add authenticated Sub group sync endpoint"
```

### Task 5: Apply snapshots transactionally in New

**Files:**
- Create: `/Users/alien/Workspace/new-api/model/sub_public_group_sync.go`
- Create: `/Users/alien/Workspace/new-api/controller/sub_public_group_sync_apply.go`
- Modify: `/Users/alien/Workspace/new-api/model/channel.go`
- Modify: `/Users/alien/Workspace/new-api/model/ability.go`
- Modify: `/Users/alien/Workspace/new-api/model/option.go`
- Modify: `/Users/alien/Workspace/new-api/setting/ratio_setting/model_ratio.go`
- Modify: `/Users/alien/Workspace/new-api/setting/ratio_setting/group_ratio.go`
- Test: `/Users/alien/Workspace/new-api/model/sub_public_group_sync_test.go`

**Step 1: Write failing application tests**

Test create, update, full model replacement, mapping changes, price changes, ratio changes, disable/delete transitions, and cache refresh. Verify unrelated channels and option entries remain unchanged.

**Step 2: Implement the application transaction**

Resolve the mirror channel by a stable marker plus exact group name. Upsert channel metadata, replace only that channel's abilities, merge pricing option maps, update group and top-up ratios, update public group visibility, record the event ID, and commit as one transaction. A non-public snapshot disables the mirror channel and its abilities.

**Step 3: Refresh runtime state**

After commit, call the existing channel cache initialization/update path and invalidate pricing/model plaza caches. Keep cache refresh outside the database transaction so a cache failure does not roll back committed data.

**Step 4: Run focused tests**

```bash
cd /Users/alien/Workspace/new-api && go test ./model ./controller ./setting/ratio_setting -run SubPublicGroupSync -count=1
```

Expected: PASS.

**Step 5: Commit**

```bash
git -C /Users/alien/Workspace/new-api add model controller setting
git -C /Users/alien/Workspace/new-api commit -m "feat: apply Sub public group snapshots"
```

### Task 6: Connect configuration and local end-to-end verification

**Files:**
- Modify: `/Users/alien/Workspace/sub2api/backend/internal/config/config.go`
- Modify: `/Users/alien/Workspace/new-api/common/config.go`
- Modify: local dev compose/env files as needed
- Test: both repositories' existing integration test locations

**Step 1: Add local development settings**

Configure the New sync URL, shared secret, timeout, retry limits, and worker interval through environment variables. Keep production secrets out of git.

**Step 2: Run repository tests**

```bash
cd /Users/alien/Workspace/sub2api/backend && go test ./internal/service ./internal/repository ./internal/handler/admin ./internal/server -count=1
cd /Users/alien/Workspace/new-api && go test ./controller ./model ./router ./setting/ratio_setting -count=1
```

**Step 3: Run the local end-to-end flow**

Start both local dev services, update one public group's model list, rename a model, change a price, change the group ratio, then disable the group. Verify each event reaches New and that the model plaza response converges after every mutation.

**Step 4: Verify retry and recovery**

Stop New during a Sub mutation, confirm the event stays pending, restart New, and verify automatic delivery and idempotent application.

**Step 5: Commit local integration changes**

```bash
git -C /Users/alien/Workspace/sub2api add backend/internal/config
git -C /Users/alien/Workspace/sub2api commit -m "feat: configure public group sync delivery"
git -C /Users/alien/Workspace/new-api add common
git -C /Users/alien/Workspace/new-api commit -m "feat: configure Sub group sync receiver"
```


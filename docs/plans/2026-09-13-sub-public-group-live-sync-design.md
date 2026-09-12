# Sub Public Group Live Sync Design

## Goal

Keep New's model plaza aligned with Sub whenever an administrator changes a
public, enabled Sub group, its models, or its pricing. The sync must cover
group creation, updates, disable/delete transitions, model additions,
removals, renames, billing mode changes, model prices, and group ratios.

Only groups satisfying all of these conditions are mirrored:

- `status=active`
- `deleted_at IS NULL`
- `is_exclusive=false`

When a group stops satisfying the conditions, New disables its mirror channel
and abilities so stale models do not remain visible in the plaza.

## Recommended Architecture

The first local implementation uses a complete snapshot pull: New polls a
Sub internal endpoint every five seconds and submits the signed snapshot to
its local authenticated apply endpoint. This keeps source mutations
independent of New availability and converges changes without coupling every
Sub write path to an HTTP call. An outbox can be added later if a deployment
requires push-only delivery.

New accepts the snapshot idempotently, applies the complete desired state to
its channel and abilities records, updates model pricing and group ratios,
refreshes caches, and acknowledges the event. Failed deliveries remain in the
outbox with exponential retry and a bounded error record.

The payload is a complete snapshot rather than a field patch. It contains the
Sub group identity and visibility state, group ratio, billing mode, channel
metadata, public model names, upstream model mappings, and all explicit model
pricing fields. This makes model additions, removals, and renames converge
without relying on event ordering.

## Data Contract

The internal endpoint is conceptually:

`POST /api/internal/sync/sub/public-group`

Headers carry a timestamp, event ID, and HMAC signature over the canonical
request body. New rejects stale timestamps, invalid signatures, and duplicate
event IDs that have already been applied.

The snapshot uses stable Sub group ID plus group name for identity. New keeps
its own channel ID and uses the exact Sub group name as the public group key.
Display model names and upstream model names are separate fields so prefixes
such as `按量vet/` remain visible while routing and pricing use the raw model.

## Change Areas

### Sub

- Add a signed snapshot endpoint in Sub and a bounded polling worker in New.
- Build snapshots from active, non-exclusive groups and active channels.
- Reconcile missing snapshots by disabling stale mirror channels.
- Add bounded timeout and error logging around each polling cycle.

### New

- Add the authenticated internal sync route and request validation.
- Add a sync service that upserts or disables the mirror channel and replaces
  its abilities in one transaction.
- Merge model prices into the existing pricing option maps while preserving
  unrelated models.
- Update `GroupRatio`, `TopupGroupRatio`, and public group visibility.
- Invalidate channel and pricing caches after a successful transaction.
- Store applied event IDs for idempotency.

## Failure Handling

Sub retries network failures, 5xx responses, timeouts, and explicit transient
errors. New returns a stable response for an already applied event. Invalid
payloads and authentication failures are recorded as permanent failures and
are not retried indefinitely. A manual full resync command remains available
for recovery.

## Verification

- Unit-test public-group filtering and snapshot serialization in Sub.
- Unit-test New signature validation, idempotency, and snapshot application.
- Test model add/remove/rename, price-only changes, ratio changes, disable,
  delete, and private-group transitions.
- Run both repositories' Go tests for changed packages.
- Exercise a local end-to-end flow with a Sub mutation and verify New's
  channel, abilities, prices, ratios, and model-plaza response.

## Development Branches

Development is performed on local `dev` branches:

- `/Users/alien/Workspace/sub2api` already has `dev`.
- `/Users/alien/Workspace/new-api` now has a new local `dev` branch from
  `main`.

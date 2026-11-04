# Public group display pricing

The version-2 sync contract optionally includes `source_group_id`, `source_model`, and `source_group_ratio` on inherited category models. All three are required together. Ordinary legacy version-1 payloads remain accepted; inherited source fields require version 2. Every snapshot version must match its envelope. Sub emits version 2 globally so an old version-1 receiver rejects the payload safely after rollback. Category entry `group_ratio` is 1. Source prices are raw USD per token or per request, already resolved by Sub (including its selected context tier and aliases).

New multiplies each provided price by `source_group_ratio` exactly once and saves it in `PublicGroupDisplayPricing`. Missing prices stay absent; explicit zero stays zero. Input, output, image input/output, and cache prices remain independent, including zero input with paid output. This is a display quote, not a new New billing policy. The response adds `group_pricing[group_name]` to the existing single model row. Neither this option nor these fields are read by relay/billing code.

`PublicGroupSyncBillingModels` records ordinary-sync model ownership. Inherited models do not participate in global model price/ratio/billing-mode precedence, writes, or deletion. Normal snapshot behavior is preserved. Deploy the updated receiver/UI before enabling source category exports; old receivers reject version 2 instead of ignoring the source fields and corrupting global prices.

## Hide stale plaza channels while retaining routes

Set `PUBLIC_GROUP_SYNC_HIDDEN_CHANNEL_IDS=20` (or a comma-separated list of positive IDs) on the New application. Only GET `/api/pricing` applies this configuration. An association is omitted only when it is supplied by selected channels and no other visible channel supplies the same `(group, model)` association. Groups exclusively exposed by selected channels are removed from this response's selector, ratio, ordering and auto-group fields. Shared models/groups remain visible through other channels. Invalid IDs produce an error response rather than silently hiding a different scope.

No channel, ability, key, token, group permission, billing option, `/v1/models` response, or shared pricing cache is changed. Removing the environment setting reverses the exclusion. Do not remove the legacy group from `UserUsableGroups` or `GroupRatio`: token authentication still requires those entries.

For the audited production state, channel 20 has real credentials for Sub custom group 260; matching model routes target source 71 rather than source 128. Channel 47 is a placeholder mirror. Token 7 retains its legacy group; channel 20 accepts 12 prefixed model names. Do not retag, rename, delete, or migrate these resources as part of this display fix.

## Source provenance and release

The local branch starts at clean `/opt/new-api-build-58cdd568` commit `a962d54961d25503a94608ea96521f196cd0ea80`. A complete source-tree comparison with `/opt/new-api-build-group-order-20260917` found only its added 30-second pricing refresh and associated hook test; both are retained. The latter directory's HEAD was `1e567d5ce82c119f5e77025cafd2a9c771ed4937` with uncommitted changes. Do not use that HEAD alone as release source.

Audited running image ID: `sha256:14cd3a069a444c31dd62ea4ab1db7608db474466e5f97ff8d65e06a366024785`. Binary SHA-256: `08f9bdeb22e7a45c5e5ad52c0bda42c8cdc24abd8b1949382a98cfcab42e57d9`. Go build information reports Go 1.26.1, no VCS revision and an empty version ldflag. Exact binary-to-commit provenance therefore cannot be independently established. Preserve the complete audited source tree and refresh hook, build off production, and label the new artifact with its actual source commit.

Public pricing uses the existing channel/pricing cache invalidation after sync; display options are read into response-local copies. Existing 30-second browser polling is retained. Token/Redis caches need no invalidation for the display exclusion because token state is unchanged. Preserve previous application containers/static assets for rollback when deploying updated frontend chunks.

Missing `/static/` resources return HTTP 404 (like missing `/assets` resources), preventing SPA HTML with HTTP 200 from masking missing chunks. The deployment plan archives publicly served static files identified from the old HTML and runtime's asynchronous chunk manifest into a dedicated read-only directory. Nginx `/static/` first uses `try_files` for archived hashes and otherwise forwards to the new application. No old application worker is retained for this fallback. Deployment should verify missing static resources return 404, archived old chunks remain reachable, and new chunks still load. No production changes are included in this patch.

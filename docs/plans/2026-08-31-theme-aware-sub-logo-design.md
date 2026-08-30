# Theme-aware Sub Logo Sync Design

## Goal

Make New API use the same Passion API logo assets as the production Sub site, including the correct light and dark variants, while preserving an administrator-configured custom logo as the highest-priority override.

## Current State

- New API falls back to `/logo.png` when `/api/status` does not provide a logo.
- The production Sub site exposes separate light and dark 512x512 PNG assets.
- New API's existing default logo and favicon do not match those production assets.
- Most UI surfaces consume the logo through `useSystemConfig`, while a few startup or legacy paths use `/logo.png` directly.

## Options Considered

### 1. Centralized theme-aware resolution (selected)

Store both Sub assets in New API, resolve the active default logo in `useSystemConfig`, and keep custom backend logos unchanged. This gives all existing hook consumers consistent behavior without duplicating theme logic.

### 2. Replace only `/logo.png`

This is the smallest change but cannot display the dark variant and leaves the favicon inconsistent when the theme changes.

### 3. Resolve the theme separately in every logo component

This works visually but duplicates policy across navigation, authentication, setup, footer, and future consumers. It also increases the chance that surfaces drift apart.

## Design

### Assets

- Add `/logo-light.png` from Sub's production light logo.
- Add `/logo-dark.png` from Sub's production dark logo.
- Replace `/logo.png` with the light asset for direct and legacy references.

### Logo Resolution

The system configuration hook will resolve the displayed logo using this priority:

1. A non-default logo returned by the backend.
2. `/logo-dark.png` when the resolved UI theme is dark.
3. `/logo-light.png` for light theme and initial fallback.

The raw system configuration remains unchanged; only the logo exposed to UI consumers is theme-aware. This avoids treating the current default path as an administrator override.

### Loading and Favicon

The existing preload and favicon update flow will operate on the resolved logo. A theme change therefore preloads the matching asset and updates both rendered branding and the browser favicon. The HTML fallback favicon will use the light logo before React starts.

### Compatibility

- Existing backend custom logos continue to override both bundled variants.
- Direct `/logo.png` consumers receive the Sub light asset.
- Existing component dimensions and layout remain unchanged.
- Theme-aware behavior is centralized, so nav, login, setup, footer, and settings stay aligned.

## Testing

- Unit-test default light resolution.
- Unit-test default dark resolution.
- Unit-test custom-logo priority in both themes.
- Run existing frontend tests, type checks, and production build.
- Visually verify desktop and mobile pricing pages in light and dark themes.
- Verify the production deployment and favicon after rollout.

## Deployment

Commit and push the change to `main`, rebuild the existing Docker service on `142.252.101.47`, recreate only the New API application container, wait for health checks, and verify `https://new.passionapi.com`.

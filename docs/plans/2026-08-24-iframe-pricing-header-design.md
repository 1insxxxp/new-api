# iframe Model Square Header Design

**Goal:** Hide the public top navigation when the pricing/model-square page is rendered inside an iframe, while preserving the current standalone page layout.

## Design

- Extend `PublicLayout` with an optional `showHeader` prop that defaults to `true`, so existing public pages keep their current behavior.
- Add a small browser-safe `isEmbeddedWindow` utility that compares the current window with its top-level window without reading cross-origin properties.
- The `Pricing` page will use that utility to pass `showHeader={false}` only when it is embedded. Its loading state and loaded state will use reduced top padding in the same mode so the model square starts near the top of the iframe.
- The normal `/pricing` page, other public pages, and the existing pricing interactions remain unchanged.

## Behavior

- Standalone `/pricing`: public navigation remains visible and the existing top spacing is preserved.
- iframe `/pricing`: public navigation is omitted and the header compensation spacing is reduced.
- The model detail drawer remains in the pricing page and requires no separate navigation behavior.
- If the browser does not expose a window context, the utility returns `false`, preserving the standalone layout as the safe fallback.

## Testing

- Unit-test iframe detection for standalone, embedded, and unavailable window contexts.
- Unit-test `PublicLayout` to verify the default header and the opt-out behavior, including the reduced main-container spacing.
- Run the pricing-related tests, typecheck, lint, production build, and browser checks for standalone and iframe rendering.

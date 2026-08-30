# Mobile Model Name Layout Design

## Problem

Pricing model cards render the model name with Tailwind's `truncate` utility.
On narrow mobile screens, the details and copy actions share the same header
row and consume much of the available width, so renamed models are reduced to
an ambiguous prefix such as `gemini-3-...`.

## Considered Approaches

1. Keep the single header row and allow the title to wrap beside the actions.
   This exposes more text, but the remaining title column is still too narrow
   on small screens and creates awkward word wrapping.
2. Show the full title in a tooltip or details dialog. This preserves card
   density, but the complete name is still not visible while scanning cards.
3. Give the title the mobile card width and place the actions on a second row,
   then restore the current single-row layout at the `sm` breakpoint.

## Decision

Use approach 3. The mobile header will use a two-column grid for the icon and
content. The model name may wrap at any character boundary so long identifiers
remain inside the card. The details and copy actions move below the title and
price in the content column. At `sm` and above, the header becomes the existing
three-column icon, content, and actions layout, and the model name returns to a
single truncated line.

The card height may grow on mobile when a model has a long name. No model data,
pricing behavior, action callbacks, desktop card density, or model details are
changed.

## Verification

- A component test verifies that the title uses mobile wrapping and restores
  truncation at `sm`.
- The same test verifies that actions occupy the second mobile row and return
  to the right-hand desktop column at `sm`.
- Existing pricing tests, type checking, and the production build pass.
- Browser screenshots at a narrow mobile viewport confirm the full model name
  is visible, actions remain reachable, and card content does not overlap.
- A desktop screenshot confirms the existing compact header remains intact.

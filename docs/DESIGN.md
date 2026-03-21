# Design System: Luxury Sci-fi / Galactic Casino

## 1. Overview & Creative North Star
**The Creative North Star: "Interstellar Elegance"**

This design system rejects the clichéd, neon-heavy "cyberpunk" aesthetic in favor of something more sophisticated: an exclusive, high-stakes lounge floating at the edge of a nebula. We are blending the organic warmth of a traditional high-end casino (mahogany, velvet) with the precision of futuristic aerospace engineering (glassmorphism, PixiJS-driven glows, and mathematical symmetry).

To break the "template" look, we utilize **Intentional Asymmetry**. UI elements should not feel boxed in; they should feel like they are floating in a zero-gravity environment. We use overlapping layers—such as cards partially breaking the bounds of player avatars—and high-contrast typography scales to create a sense of cinematic depth.

---

## 2. Colors & Surface Philosophy
The palette is rooted in the "Deep Space" (`#10141a`) experience, punctuated by "Supernova Gold" (`#f2ca50`) and "Nebula Green" (`#afcdbd`).

### The "No-Line" Rule
Traditional 1px solid borders are strictly prohibited for layout sectioning. Separation must be achieved through:
- **Tonal Shifts:** Placing a `surface-container-high` element against a `surface-dim` background.
- **Luminous Glows:** Using a subtle outer glow (0-2px spread) in `primary` (Gold) to define an active player’s seat rather than a stroke.

### Surface Hierarchy & Nesting
Treat the UI as a physical stack of luxury materials. 
1. **Base Layer:** `surface-dim` (#10141a) — The infinite void of space.
2. **Table Surface:** `secondary-container` (#344f42) — The "Forest Green Velvet," utilized for the central field of play.
3. **Control HUD:** `surface-container-lowest` (#0a0e14) — The dark, recessed mahogany-inspired docks for action buttons.
4. **Active Elements:** `surface-bright` (#353940) — Floating modals or tooltips that catch the "light" of nearby stars.

### The "Glass & Gradient" Rule
Main CTAs (like "All-In") must use a linear gradient from `primary` (#f2ca50) to `primary-container` (#d4af37) at a 135-degree angle. Floating HUD elements should use **Glassmorphism**: `surface-container` at 60% opacity with a 12px backdrop-blur to allow the "velvet" of the table to bleed through the UI.

---

## 3. Typography
We use a dual-font system to balance technical precision with editorial luxury.

- **Display & Headlines (`Space Grotesk`):** This is our "Sci-fi" voice. It is used for high-impact moments: "BIG BLIND," "YOU WIN," or "ROYAL FLUSH." The wide apertures and geometric shapes feel engineered. Use `primary` (Gold) for headers to imply value.
- **Body & Labels (`Manrope`):** Our "Luxury" voice. Manrope provides the clean, high-readability required for chip counts and card values. 
- **Hierarchy:** Use `label-sm` for secondary metadata (e.g., "Pot: $4,500") and `headline-md` for primary actions. Always maintain a gold-over-white hierarchy to guide the eye toward the "wealth" (money/chips).

---

## 4. Elevation & Depth
Depth in this system is atmospheric, not structural.

- **The Layering Principle:** A player’s "Seat" should be a `surface-container-high` circle. When it is that player's turn, do not use a heavy stroke; instead, shift the background to `surface-bright` and add a `primary` (Gold) ambient glow.
- **Ambient Shadows:** For floating cards, use a shadow with `blur: 24px`, `opacity: 0.15`, and a color derived from `surface-container-lowest`. It should feel like the card is casting a shadow onto the green velvet table.
- **The "Ghost Border" Fallback:** If a boundary is required for accessibility (e.g., input fields), use `outline-variant` at 15% opacity. It should feel like a holographic projection, not a physical wire.

---

## 5. Components

### Player Seats (Avatars)
- **Shape:** Oval/Circular with a `surface-container-low` base. 
- **Detailing:** A "Ghost Border" of `primary` at 20% opacity. 
- **State:** When active, the avatar gains a subtle `primary` pulse animation (PixiJS filter).

### Action Bar (Fold, Check, Call, Raise)
- **Container:** A horizontal dock using `surface-container-lowest` with a `lg` (1rem) corner radius.
- **Buttons:** 
    - **Fold:** `surface-variant` with `on-surface-variant` text.
    - **Call/Check:** `secondary-container` with `on-secondary-container` text.
    - **All-In:** The "Hero" button. Gradient fill (`primary` to `primary-container`) with `on-primary` (Dark Mahogany) text for maximum contrast.
- **Spacing:** Use `spacing-3` (0.6rem) between action buttons.

### Betting Chips
- **Visuals:** Perfect circles. Use `primary` for the core, with a concentric "engraved" ring using `primary-fixed-dim`. 
- **Stacking:** Use `spacing-0.5` (0.1rem) vertical offsets to create 3D stacks. No borders—use the natural color transition of the gold tokens to define depth.

### Hand Strength Bar
- **Track:** `surface-container-highest`.
- **Progress:** A gradient from `secondary` (Mint/Green) to `primary` (Gold).
- **Animation:** Use a "shimmer" effect to indicate the bar is live and calculating in real-time.

### Game Cards
- **Front:** `on-primary-fixed` (White/Cream) background. Suits in traditional colors, but the card border is a `px` width of `primary` (Gold).
- **Back:** `secondary-container` (Dark Green) with a geometric diamond pattern in `secondary-fixed-dim`.

---

## 6. Do's and Don'ts

### Do
- **Do** use `space-20` and `space-24` to create massive breathing room between the table and the HUD.
- **Do** use `backdrop-blur` on all modal overlays to maintain the "Galactic" atmosphere.
- **Do** animate typography transitions (e.g., chip counts ticking up) using a smooth easing function.

### Don't
- **Don't** use pure `#000000` black. Always use the deep navy/charcoal of `surface-dim`.
- **Don't** use standard "Material" shadows. They are too heavy. Stick to low-opacity, wide-spread ambient glows.
- **Don't** use divider lines in the player list or game history. Use `spacing-4` or subtle background shifts in `surface-container` tiers to separate entries.
- **Don't** use sharp corners. Always stick to the `md` (0.75rem) or `lg` (1rem) roundedness scale to maintain the "luxury" feel.
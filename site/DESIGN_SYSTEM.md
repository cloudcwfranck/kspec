# kspec Design System

**Based on Linear.app's design philosophy**

This document outlines the design system implemented for the kspec marketing and documentation website. The design follows Linear.app's principles: minimal, fast, distraction-free UI with typography and spacing doing the work.

---

## Design Philosophy

**Core Principles:**
- Minimal, fast, distraction-free UI
- Reduce cognitive load at all times
- Typography and spacing do the work — never decoration
- Every element must justify its existence
- Keyboard-friendly, engineer-first experience

**Quality Bar:**
- The UI should feel fast, calm, and precise
- Resembles Linear's experience at first glance
- Clarity over expressiveness
- If an element feels decorative or loud, remove it

---

## Color System

### Dark Mode (Primary)

```typescript
// App backgrounds
'linear-bg': '#0E0F12'              // Main app background
'linear-surface': '#16181D'         // Cards, panels, elevated surfaces
'linear-surface-secondary': '#1C1F26' // Secondary surfaces

// Text colors
'linear-text': '#EDEEF0'            // Primary text (headings, body)
'linear-text-secondary': '#A8ADB7'  // Secondary text (descriptions)
'linear-text-muted': '#8A8F98'      // Muted text (meta information)

// Borders
'linear-border': '#262A33'          // All borders

// Accent (use sparingly)
'accent': '#5E6AD2'                 // Primary accent
'accent-hover': '#6B76DB'           // Accent hover state
```

### Semantic Colors

```typescript
'success': '#4CB782'  // Success states
'warning': '#F2C94C'  // Warning states
'error': '#EB5757'    // Error states
```

### Color Usage Rules

1. **Never oversaturate** - Colors are intentionally muted
2. **One accent color** - Only `#5E6AD2` for focus and CTAs
3. **Accent guides focus** - Never let it dominate the UI
4. **Text never pure black/white** - Always use the text scale
5. **Borders are subtle** - Low contrast, 1px solid only

---

## Typography

### Font Stack

```css
font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI',
             'Roboto', 'Helvetica Neue', 'Arial', sans-serif;
```

### Weights & Sizes

```typescript
// Weights
400: 'Body text'
500: 'Section headers'
600: 'Page titles'

// Sizes
text-5xl (48px): 'Page titles'
text-3xl (30px): 'Section headers'
text-xl (20px): 'Subsection headers'
text-base (16px): 'Body text'
text-sm (14px): 'Meta text, labels'
text-xs (12px): 'Tiny meta info'

// Line height
1.45: 'Standard for readability'
```

### Typography Rules

1. Short sentences, calm tone
2. No emojis in production UI
3. Labels over paragraphs
4. Prefer verbs over adjectives
5. Marketing language minimized

---

## Layout & Structure

### Container Widths

```typescript
'max-w-4xl': '56rem (896px)'  // Article content
'max-w-6xl': '72rem (1152px)' // Wide sections
'max-w-7xl': '80rem (1280px)' // App container
```

### Spacing System (8px base)

```typescript
spacing: {
  '1': '4px',   // 0.5 unit
  '2': '8px',   // 1 unit
  '3': '12px',  // 1.5 units
  '4': '16px',  // 2 units
  '6': '24px',  // 3 units
  '8': '32px',  // 4 units
  '12': '48px', // 6 units
}
```

### Border Radius

```typescript
'rounded-lg': '10px'  // Standard cards
'rounded-xl': '12px'  // Buttons, prominent elements
'rounded-full': '9999px' // Pills, dots
```

### Borders

- **Width:** Always 1px
- **Style:** Solid only
- **Color:** `border-linear-border` (#262A33)
- **No shadows** - Separation through space, not depth

---

## Components

### Buttons

#### Primary Button
```tsx
className="btn-primary"
// Solid accent color, 12px radius, 120ms transition
// px-6 py-3 bg-accent text-white rounded-xl
```

#### Secondary Button
```tsx
className="btn-secondary"
// Transparent with border, hover shows subtle fill
// px-6 py-3 bg-transparent border border-linear-border
```

#### Ghost Button
```tsx
className="text-accent hover:text-accent-hover"
// Text only, no background
```

### Cards

```tsx
<div className="bg-linear-surface border border-linear-border rounded-xl p-6">
  <h3 className="text-lg font-semibold text-linear-text mb-2">Title</h3>
  <p className="text-sm text-linear-text-secondary">Description</p>
</div>
```

### Code Blocks

```tsx
<pre className="bg-linear-bg border border-linear-border text-linear-text-secondary rounded-xl p-4">
  <code>// Your code here</code>
</pre>
```

---

## Motion & Transitions

### Timing

```typescript
'fast': '120ms'   // Hover states, simple changes
'normal': '180ms' // Panel transitions
'slow': '220ms'   // Modal transitions
```

### Easing

```typescript
'cubic-bezier(0.2, 0, 0, 1)' // All transitions (linear-ease)
```

### Motion Rules

1. Motion is **functional**, not expressive
2. Subtle fade + translate (2-4px max)
3. No bounce, no spring physics
4. No attention-seeking animations

---

## Iconography

### Style
- **Stroke-based only** (1.5-2px stroke)
- **No filled icons** unless semantic (status indicators)
- **Icons support text**, never replace it
- **Size:** 20-24px standard

### Usage
```tsx
<svg className="w-5 h-5 text-linear-text-secondary" fill="none" stroke="currentColor">
  {/* Icon paths */}
</svg>
```

---

## Page Structure

### Standard Page Layout

```tsx
<div className="min-h-screen bg-linear-bg">
  {/* Header */}
  <div className="border-b border-linear-border py-16">
    <div className="max-w-4xl mx-auto px-6">
      <h1 className="text-5xl font-bold text-linear-text">Page Title</h1>
      <p className="text-xl text-linear-text-secondary">Subtitle</p>
    </div>
  </div>

  {/* Content */}
  <div className="max-w-4xl mx-auto px-6 py-12">
    {/* Content sections */}
  </div>
</div>
```

---

## Implementation Notes

### Tailwind Configuration

The design system is implemented through Tailwind CSS custom tokens in `tailwind.config.js`:

```javascript
theme: {
  extend: {
    colors: { /* Linear color palette */ },
    spacing: { /* 8px system */ },
    borderRadius: { /* 10-12px */ },
    transitionDuration: { /* 120-220ms */ },
    transitionTimingFunction: { /* cubic-bezier */ }
  }
}
```

### CSS Classes

Global utility classes are defined in `globals.css`:
- `.btn-primary` / `.btn-secondary` - Button styles
- `.prose` - Typography overrides for MDX content
- Scrollbar styling

### Color Migration

All components use the Linear color tokens:
- `bg-linear-bg`, `bg-linear-surface`, `bg-linear-surface-secondary`
- `text-linear-text`, `text-linear-text-secondary`, `text-linear-text-muted`
- `border-linear-border`
- `bg-accent`, `text-accent`

---

## Do's and Don'ts

### ✅ Do

- Use the 8px spacing system
- Keep borders 1px and subtle
- Use accent color sparingly for CTAs and focus
- Write short, clear copy
- Test in dark mode
- Prioritize keyboard navigation
- Use semantic HTML

### ❌ Don't

- Use gradients or shadows for decoration
- Oversaturate colors
- Use more than one accent color
- Add bounce or spring animations
- Use emojis in production UI
- Create decorative elements
- Use pure black (#000) or pure white (#fff)
- Hardcode pixel values outside the spacing system

---

## File Organization

```
/site
├── app/                    # Next.js pages
│   ├── globals.css        # Global styles + utilities
│   └── layout.tsx         # Root layout with Linear theme
├── components/            # Reusable components
├── tailwind.config.js     # Design system tokens
└── DESIGN_SYSTEM.md       # This file
```

---

## Deliverables Checklist

- [x] Complete Tailwind configuration with Linear tokens
- [x] Core layout components (Navigation, Footer)
- [x] Button system (primary, secondary, ghost)
- [x] Typography scale and hierarchy
- [x] Color system implementation
- [x] Spacing system (8px base)
- [x] Motion/transition standards
- [x] Fully-designed pages demonstrating system
- [x] Design system documentation

---

## References

- **Design Inspiration:** [Linear.app](https://linear.app)
- **Framework:** Next.js 14 App Router
- **Styling:** Tailwind CSS
- **Font:** Inter (Google Fonts)
- **Icons:** Heroicons (stroke-based)

---

**Last Updated:** 2025-12-31
**Version:** 1.0.0
**Status:** Production-ready

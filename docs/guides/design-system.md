# Design System

This document defines the visual language, component patterns, and interaction conventions used across Paca's web interface. Every page and component should follow these rules to maintain consistency.

**Reference implementation:** `apps/web/src/components/projects/interactions/task-detail/`

---

## Table of Contents

- [Design Concept](#design-concept)
- [Color Philosophy](#color-philosophy)
- [Typography](#typography)
- [Spacing & Layout](#spacing--layout)
- [Surfaces & Cards](#surfaces--cards)
- [Borders & Dividers](#borders--dividers)
- [Shadows & Depth](#shadows--depth)
- [Opacity & Text Hierarchy](#opacity--text-hierarchy)
- [Buttons & Interactive Elements](#buttons--interactive-elements)
- [Badges & Status Chips](#badges--status-chips)
- [Avatars](#avatars)
- [Forms & Inputs](#forms--inputs)
- [Checkboxes](#checkboxes)
- [Progress Bars](#progress-bars)
- [Tags / Pills](#tags--pills)
- [Popovers & Dropdowns](#popovers--dropdowns)
- [Section Headings](#section-headings)
- [Field Rows](#field-rows)
- [Empty States](#empty-states)
- [Modals & Dialogs](#modals--dialogs)
- [Side Panels](#side-panels)
- [Activity Feed](#activity-feed)
- [Scrollbars](#scrollbars)
- [Transitions & Animations](#transitions--animations)
- [File Patterns](#file-patterns)

---

## Design Concept

### High-Contrast Minimalism

Paca's visual language is **High-Contrast Minimalism** — an aesthetic built on two principles that reinforce each other:

**Minimalism** strips away noise. No gradient meshes, no dot grids, no layered translucency effects. Surfaces are flat, pure, and intentional. Every visual element earns its presence.

**High Contrast** makes hierarchy unmistakable. Deep black text on pure white, pure white text on near-black, and a single electric lime accent (`#9ed957`) that signals action, focus, and energy without ambiguity.

### Palette

| Token | Light | Dark | Role |
|---|---|---|---|
| `--background` | `#ffffff` | `#0a0a0a` | Page and modal root — absolutely pure |
| `--foreground` | `#111111` | `#f0f0f0` | Primary text — near-black / near-white |
| `--primary` | `#9ed957` | `#9ed957` | Lime accent — CTAs, active states, focus rings |
| `--primary-foreground` | `#0d0d0d` | `#0a0a0a` | Text on lime buttons — always dark |
| `--card` | `#ffffff` | `#111111` | Component surface |
| `--muted` | `#f5f5f5` | `#1a1a1a` | Subtle background fills |
| `--muted-foreground` | `#737373` | `#888888` | Labels, placeholders, secondary text |
| `--border` | `#d4d4d4` | `#2a2a2a` | Structural dividers |
| `--sidebar` | `#fafafa` | `#0d0d0d` | Navigation surface |

### Design Rules

1. **Pure backgrounds**: Light mode is `#ffffff` — no tints, no gradients. Dark mode is `#0a0a0a`. Never tint the root canvas.
2. **One accent, consistently applied**: Lime (`#9ed957`) is the only chromatic accent. Use it for primary actions, active nav indicators, focus rings, and kicker labels.
3. **No decorative backgrounds**: No mesh gradients, no dot grids. Contrast comes from content relationships, not ornamental texture.
4. **Sharp but subtle radii**: Border radius is `0.5rem` (8px) — just enough softness to feel crafted, not rounded to the point of looking playful.
5. **Flat shadows**: Shadows are `0 1px 3px rgba(0,0,0,0.07)` at most — present for elevation cues, never decorative.
6. **Emerald for completion**: The emerald green (`bg-emerald-500`) is reserved exclusively for success/completed states (checkboxes, progress bars at 100%).

---

## Color Philosophy

We use **opacity-modulated semantic tokens** — never raw hex values in component code. Colors are expressed as `token/opacity` where the opacity communicates hierarchy:

| Layer | Token | Usage |
|---|---|---|
| Background | `bg-background` | Page/modal root |
| Surface | `bg-card/50` to `bg-card/80` | Cards, panels, hover states |
| Muted surface | `bg-muted/20` to `bg-muted/60` | Toolbars, chip backgrounds, inactive states |
| Primary | `bg-primary`, `text-primary` | CTAs, active indicators, links |
| Destructive | `text-destructive` | Delete buttons, validation errors |
| Success | `bg-emerald-500` (via Tailwind) | Completed states, checkmarks, confirmations |

**Key rule:** Every "secondary" or "subtle" background uses a fraction of `muted` or `card` with opacity between 10–50%. Because `--muted` is a neutral gray (`#f5f5f5` light / `#1a1a1a` dark), opacity fractions produce clean, achromatic layers — no blue tinting, no color bleed. The single chromatic accent (`--primary: #9ed957`) therefore lands with full punch wherever it appears.

---

## Typography

### Font Families

| Role | Family | Tailwind class |
|---|---|---|
| Display / Headings | Syne | `font-[Syne]` |
| Body / UI text | DM Sans | default (`font-sans`) |
| Monospace / IDs | JetBrains Mono | `font-[JetBrains_Mono,monospace]` |

### Type Scale

| Role | Size | Weight | Tracking | Class |
|---|---|---|---|---|
| **Page title** | 26px | bold | tight | `text-[26px] font-bold tracking-tight` |
| **Section heading** | 11px | semibold | 0.08em uppercase | `text-[11px] font-semibold uppercase tracking-[0.08em]` |
| **Body text** | 14px | normal | default | `text-[14px]` |
| **Field values** | 13px | medium | default | `text-[13px] font-medium` |
| **Field labels** | 13px | medium | default | `text-[13px] font-medium text-muted-foreground` |
| **Small labels** | 12px | medium | default | `text-[12px] font-medium` |
| **Mini text / IDs** | 11px | semibold/bold | wider | `text-[11px] font-semibold tracking-wider` |
| **Micro text** | 10px | bold | default | `text-[10px] font-bold` |

### Inline Edit Pattern

When text is click-to-edit, both display and edit modes share the same typographic class so the layout doesn't shift:

```tsx
const TITLE_CLASSES = "font-[Syne] text-[26px] font-bold leading-snug text-foreground tracking-tight w-full";

// Display
<h1 className={cn(TITLE_CLASSES, canEdit && "cursor-text hover:bg-muted/15 rounded-md px-2 -ml-2 py-1")}>{title}</h1>

// Edit
<textarea className={cn(TITLE_CLASSES, "resize-none bg-transparent outline-none py-0")} />
```

---

## Spacing & Layout

### Content Container

```tsx
<div className="px-8 py-7 space-y-8 max-w-3xl mx-auto">
```

- **Horizontal padding:** `px-8` (32px)
- **Vertical padding:** `py-7` (28px)
- **Section gap:** `space-y-8` (32px between major sections)
- **Max content width:** `max-w-3xl` (768px), centered with `mx-auto`

### Component-Level Spacing

| Context | Gap | Pattern |
|---|---|---|
| Between sections | 32px | `space-y-8` |
| Within a section | 12px | `space-y-3` |
| Between field rows | 10px | `py-2.5` |
| Inline item gap | 8–12px | `gap-2` to `gap-3` |
| Icon-to-text gap | 6px | `gap-1.5` |

---

## Surfaces & Cards

All container surfaces follow a layered approach with very low opacity:

### Primary Card

```tsx
<div className="rounded-xl border border-border/30 bg-card/50 divide-y divide-border/20">
```

### Hover-Elevated Card

```tsx
<div className="rounded-xl border border-border/25 bg-muted/15 hover:bg-muted/25 hover:border-border/35 transition-all duration-150">
```

### Toolbar / Header Surface

```tsx
<div className="bg-muted/20 border-b border-border/30">
```

### Sidebar / Secondary Surface

```tsx
<div className="bg-muted/10 border-l border-border/25">
```

### Drop Zone (Empty State)

```tsx
<div className="rounded-xl border-2 border-dashed border-border/25 bg-muted/5 hover:border-border/40 hover:bg-muted/10 transition-all duration-200">
```

**Key rule:** Border opacity ranges from `/15` (ghostly) to `/50` (hover/emphasis). Default resting state is `/25`–`/30`.

---

## Borders & Dividers

### Thickness

- Standard border: `border` (1px)
- Dividers within cards: `divide-y divide-border/20`
- Separator accents: `h-px bg-gradient-to-r from-border/40 to-transparent`

### Divider Pattern for Section Headings

Every section heading has a horizontal gradient line that fades to transparent:

```tsx
<h3 className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70 flex items-center gap-2">
  <span>Section Name</span>
  <div className="flex-1 h-px bg-linear-to-r from-border/40 to-transparent" />
</h3>
```

---

## Shadows & Depth

Shadows are minimal — High-Contrast Minimalism uses content relationships and borders to convey depth, not elaborate shadow stacks:

| Context | Shadow |
|---|---|
| **Modal** | `shadow-[0_8px_32px_-4px_rgba(0,0,0,0.18),0_0_0_1px_rgba(0,0,0,0.06)]` |
| **Popover / Dropdown** | `shadow-md` |
| **Send / CTA button** | `shadow-sm` |
| **Completed checkbox** | `shadow-sm shadow-emerald-500/20` |
| **Progress bar (100%)** | `shadow-sm shadow-emerald-500/30` |
| **Focused input** | `shadow-sm shadow-primary/10` |
| **island-shell** | `0 1px 3px rgba(0,0,0,0.07), 0 1px 2px rgba(0,0,0,0.04)` |

**Rule:** Never use blue- or color-tinted shadows. All shadows are neutral black at low opacity.

---

## Opacity & Text Hierarchy

Text uses `/opacity` modifiers on semantic color tokens to establish hierarchy:

| Level | Token | Example |
|---|---|---|
| **Primary text** | `text-foreground` | Titles, values, body text, assignee names, description content |
| **Secondary text** | `text-foreground/80` | Dialog labels, tag text, breadcrumb current item |
| **Muted label** | `text-muted-foreground` | Field labels, date pills, type/sprint triggers, activity header |
| **Secondary muted** | `text-muted-foreground/80` | Subtask status pills, count badges, formatting toolbar items |
| **Tertiary muted** | `text-muted-foreground/70` | Section headings, "share" button, track time, relationships, drop zone text |
| **Placeholder text** | `text-muted-foreground/60` | Hash icon, "created" date, close button, add-tag trigger |
| **Subtle text** | `text-muted-foreground/50` | Empty field values, "unassigned"/"no sprint", textarea placeholder |
| **Ghost text** | `text-muted-foreground/45` to `/40` | Empty states, time-ago labels, micro hints |
| **Disabled** | `disabled:opacity-40` | Inactive elements |

**Key rule:** Readability comes first — especially in dark mode where `--muted-foreground` (`#7a8db3`) is already a muted blue against a near-black background (`#070c18`). Never go below `/40` for any text that a user might need to read. Reserve `/45`–`/40` for decorative or timestamp-level text only. Primary content (`text-foreground`) and labels (`text-muted-foreground`) use full opacity or no opacity modifier at all.

---

## Buttons & Interactive Elements

### Ghost Button (Primary action in panels)

```tsx
<button className="flex items-center gap-1.5 rounded-lg bg-primary/8 text-primary/80
  hover:bg-primary/15 hover:text-primary px-2.5 py-1.5 text-[11px] font-semibold
  transition-all duration-150">
  <Plus className="size-3" />
  Add Task
</button>
```

### Secondary Button

```tsx
<button className="flex items-center gap-1.5 rounded-lg bg-muted/40 text-muted-foreground/80
  hover:bg-muted/60 hover:text-foreground px-2.5 py-1.5 text-[11px] font-semibold
  transition-all duration-150">
```

### Icon Button

```tsx
<button className="flex size-7 items-center justify-center rounded-md
  text-muted-foreground/60 hover:text-foreground hover:bg-muted/60
  transition-all duration-150">
  <X className="size-3.5" />
</button>
```

### CTA / Submit Button

```tsx
<button className="rounded-lg bg-primary px-4 py-2 text-[13px] font-semibold
  text-primary-foreground hover:bg-primary/90 shadow-sm transition-all duration-150">
  Create field
</button>
```

### Inline Text Button

```tsx
<button className="text-[12px] text-muted-foreground/70 hover:text-foreground
  transition-colors duration-150 font-medium">
```

---

## Badges & Status Chips

### Type Badge (with dynamic color)

```tsx
<span
  className="inline-flex items-center gap-1.5 rounded-md px-2.5 py-1
    text-[11px] font-bold leading-tight tracking-wide border"
  style={{
    borderColor: color ? `${color}44` : "var(--border)",
    backgroundColor: color ? `${color}15` : "var(--muted)",
    color: color ?? "inherit",
  }}
>
  <TypeIcon className="size-3.5 opacity-70" />
  {name}
</span>
```

### Status Chip

```tsx
<span className="inline-flex items-center gap-2 rounded-full border border-border/40
  bg-muted/40 px-3 py-1 text-[11px] font-semibold text-muted-foreground
  tracking-wide backdrop-blur-sm">
  <span className="size-1.75 rounded-full shrink-0 ring-2 ring-offset-1 ring-offset-background"
    style={{ background: color, boxShadow: `0 0 6px ${color}40` }} />
  {name}
</span>
```

### ID Chip

```tsx
<div className="flex items-center gap-1.5 rounded-md bg-muted/60 px-2 py-1
  border border-border/30">
  <Hash className="size-3 text-muted-foreground/60" />
  <span className="font-mono text-[11px] font-semibold text-muted-foreground tracking-wider">
    {shortId}
  </span>
</div>
```

### Count Badge

```tsx
<span className="rounded-full bg-muted/60 px-2 py-0.5 text-[10px] font-bold
  text-muted-foreground/70 tabular-nums">
  {count}
</span>
```

---

## Avatars

### User Avatar

```tsx
<div className="flex size-6 items-center justify-center rounded-full
  bg-linear-to-br from-primary/20 to-primary/10 text-primary text-[10px] font-bold
  ring-1 ring-primary/20">
  {initial}
</div>
```

### Inactive Avatar

```tsx
<div className="flex size-6 items-center justify-center rounded-full
  bg-linear-to-br from-muted/80 to-muted/40 text-muted-foreground text-[10px] font-bold
  ring-1 ring-border/25">
  {initial}
</div>
```

---

## Forms & Inputs

### Text Input

```tsx
<input className="w-full rounded-lg border border-border/30 bg-muted/15
  px-3.5 py-2.5 text-[13px] outline-none
  focus:border-primary/40 focus:ring-2 focus:ring-primary/15
  placeholder:text-muted-foreground/50 transition-all duration-150" />
```

### Date Pill Input

```tsx
<label className="inline-flex items-center gap-1.5 rounded-lg border border-border/25
  bg-muted/25 px-2.5 py-1.5 text-[11px] text-muted-foreground/70
  hover:border-border/50 hover:bg-muted/40 transition-all duration-150
  cursor-pointer font-medium">
  <CalendarDays className="size-3 shrink-0 opacity-70" />
  <span>{displayDate(date) ?? "Start date"}</span>
  <input type="date" className="sr-only" />
</label>
```

### Number Input

```tsx
<input type="number" className="w-16 rounded-lg border border-border/30 bg-muted/25
  px-2.5 py-1 text-[13px] text-center tabular-nums font-medium
  focus:ring-2 focus:ring-primary/20 focus:border-primary/40 transition-all duration-150" />
```

---

## Checkboxes

### Standard Checkbox

```tsx
<button className={cn(
  "flex size-4.5 shrink-0 items-center justify-center rounded-[5px]
    border-2 transition-all duration-200",
  checked
    ? "border-emerald-500 bg-emerald-500 text-white shadow-sm shadow-emerald-500/20"
    : "border-border/40 text-transparent hover:border-border/70 hover:bg-muted/40"
)}>
  <Check className="size-2.5" strokeWidth={3} />
</button>
```

### Dashed Placeholder Checkbox (for "add item" rows)

```tsx
<div className="size-4.5 shrink-0 rounded-[5px] border-2 border-dashed border-border/25" />
```

---

## Progress Bars

```tsx
<div className="h-1.5 rounded-full bg-border/25 overflow-hidden">
  <div className={cn(
    "h-full rounded-full transition-all duration-500 ease-out",
    pct === 100
      ? "bg-emerald-500 shadow-sm shadow-emerald-500/30"
      : "bg-primary/60"
  )} style={{ width: `${pct}%` }} />
</div>
```

---

## Tags / Pills

### Tag

```tsx
<span className="inline-flex items-center gap-1 rounded-md bg-muted/50 px-2 py-0.5
  text-[11px] font-medium text-foreground/80 border border-border/20
  hover:border-border/40 transition-colors duration-150">
  {tag}
  <button className="text-muted-foreground/60 hover:text-destructive transition-colors duration-150">
    <X className="size-2.5" />
  </button>
</span>
```

### Add Tag Trigger

```tsx
<button className="inline-flex items-center gap-1 rounded-md border border-dashed
  border-border/30 px-2 py-0.5 text-[11px] text-muted-foreground/60
  hover:border-border/60 hover:text-muted-foreground transition-all duration-150">
  <Plus className="size-2.5" />
  Add tag
</button>
```

---

## Popovers & Dropdowns

### Popover Container

```tsx
<PopoverContent className="w-52 p-1 rounded-xl border border-border/40 shadow-lg" align="start">
```

### Popover Item

```tsx
<button className="flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-[13px]
  hover:bg-muted/60 transition-colors duration-100">
  <Icon className="size-3.5 text-muted-foreground/80 shrink-0" />
  <span className="flex-1 text-left">{label}</span>
  {selected && <Check className="size-3.5 text-primary" />}
</button>
```

---

## Section Headings

Every content section uses the same heading pattern — uppercase micro text with a trailing gradient line:

```tsx
<h3 className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70
  mb-3 flex items-center gap-2">
  <span>Section Name</span>
  <div className="flex-1 h-px bg-linear-to-r from-border/40 to-transparent" />
</h3>
```

The heading is rendered by the `SectionHeading` primitive in `primitives.tsx`.

---

## Field Rows

Property rows use a CSS grid with a fixed label column and a flexible value column:

```tsx
<div className="grid grid-cols-[9.5rem_1fr] items-center gap-4 py-2.5 px-1
  group/field rounded-lg hover:bg-muted/30 transition-colors duration-150">
  <span className="text-[13px] font-medium text-muted-foreground leading-snug select-none">
    {label}
  </span>
  <div className="min-w-0">{children}</div>
</div>
```

Empty values: `<span className="text-[13px] text-muted-foreground/50 italic">Empty</span>`

---

## Empty States

Empty states use an icon inside a rounded container with centered text:

```tsx
<div className="flex flex-col items-center py-8 text-muted-foreground/40">
  <Icon className="size-6 mb-2" />
  <p className="text-[12px] font-medium">No items yet</p>
</div>
```

For inline empty states:

```tsx
<div className="flex items-center gap-3 px-1 py-3 text-muted-foreground/45">
  <ListChecks className="size-4 opacity-70" />
  <p className="text-[13px] italic">No subtasks yet</p>
</div>
```

### Clickable Empty State (add placeholder)

```tsx
<button className="w-full rounded-xl border-2 border-dashed border-border/25 bg-muted/10
  px-5 py-6 text-left hover:border-primary/20 hover:bg-muted/20
  transition-all duration-200 group/add">
  <div className="flex items-center gap-3">
    <div className="flex size-8 items-center justify-center rounded-lg bg-muted/40
      text-muted-foreground/45 group-hover/add:text-muted-foreground/70 transition-colors">
      <FileText className="size-4" />
    </div>
    <span className="text-[13px] text-muted-foreground/60 group-hover/add:text-muted-foreground
      font-medium transition-colors">
      Add a description…
    </span>
  </div>
</button>
```

---

## Modals & Dialogs

### Backdrop

```tsx
<div className="fixed inset-0 z-50 bg-black/30 backdrop-blur-[3px]
  transition-opacity duration-200" />
```

### Modal Panel

```tsx
<div role="dialog" aria-modal="true"
  className="fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2
    flex h-[90vh] w-[92vw] max-w-6xl flex-col overflow-hidden
    rounded-xl border border-border/50 bg-background
    shadow-[0_8px_32px_-4px_rgba(0,0,0,0.18),0_0_0_1px_rgba(0,0,0,0.06)]
    transition-all duration-200 origin-center">
```

### Entry Animation

```tsx
// Open
"opacity-100 scale-100"
// Closed
"opacity-0 scale-[0.97] pointer-events-none"
```

### Dialog (nested, like "Add Field")

```tsx
<div className="relative z-10 w-full max-w-sm rounded-xl border border-border/40
  bg-background p-6
  shadow-[0_8px_32px_-4px_rgba(0,0,0,0.14),0_0_0_1px_rgba(0,0,0,0.05)]">
```

---

## Side Panels

### Structure

```tsx
<div className="flex w-80 shrink-0 flex-col overflow-hidden border-l border-border/25 bg-muted/10">
  {/* Header */}
  <div className="flex shrink-0 items-center gap-2.5 border-b border-border/25 px-5 py-3 bg-muted/20">
    ...
  </div>

  {/* Scrollable content */}
  <ScrollArea className="flex-1 px-4 py-4">...</ScrollArea>

  {/* Fixed footer / input */}
  <div className="shrink-0 border-t border-border/25 p-3 bg-background/50">...</div>
</div>
```

---

## Activity Feed

### Activity Item (non-comment)

```tsx
<div className="flex gap-3">
  <div className="flex size-6 shrink-0 items-center justify-center rounded-full
    bg-muted/40 text-muted-foreground/80 ring-1 ring-border/20">
    {initial}
  </div>
  <div className="flex flex-wrap items-baseline gap-1.5 py-0.5">
    <span className="text-[12px] font-medium text-foreground/80">{author}</span>
    <span className="text-[12px] text-muted-foreground/70">{content}</span>
    <span className="text-[10px] text-muted-foreground/45">{timeAgo}</span>
  </div>
</div>
```

### Activity Item (comment)

```tsx
<div className="flex gap-3">
  <div className="flex size-6 shrink-0 items-center justify-center rounded-full
    bg-linear-to-br from-primary/20 to-primary/10 text-primary ring-1 ring-primary/15">
    {initial}
  </div>
  <div className="rounded-xl rounded-tl-lg border border-border/25 bg-card/70 px-3.5 py-2.5">
    <div className="mb-1 flex items-center gap-2">
      <span className="text-[12px] font-semibold text-foreground">{author}</span>
      <span className="text-[10px] text-muted-foreground/50">{timeAgo}</span>
    </div>
    <p className="text-[13px] text-foreground leading-relaxed">{content}</p>
  </div>
</div>
```

### Comment Input

```tsx
<div className="flex items-end gap-2 rounded-xl border border-border/30 bg-card/80
  px-3 py-2.5 transition-all duration-200
  focus-within:border-primary/25 focus-within:shadow-sm focus-within:shadow-primary/5">
  <textarea className="flex-1 resize-none bg-transparent text-[13px] outline-none
    placeholder:text-muted-foreground/55 leading-relaxed" />
  <button className="flex size-7 shrink-0 items-center justify-center rounded-lg
    bg-primary text-primary-foreground shadow-sm hover:bg-primary/90
    disabled:opacity-40 transition-all duration-150">
    <Send className="size-3" />
  </button>
</div>
```

---

## Scrollbars

Custom visible scrollbar for content areas:

```tsx
<div className="flex-1 overflow-y-auto [scrollbar-gutter:stable]
  [&::-webkit-scrollbar]:w-2
  [&::-webkit-scrollbar-track]:bg-transparent
  [&::-webkit-scrollbar-thumb]:rounded-full
  [&::-webkit-scrollbar-thumb]:bg-border/60
  [&::-webkit-scrollbar-thumb]:hover:bg-border">
```

For side panels where auto-hide is acceptable, use `<ScrollArea>` from `@/components/ui/scroll-area`.

---

## Transitions & Animations

### Default Transition

```tsx
transition-all duration-150
```

Use for hover states, background changes, and border color shifts.

### Medium Transition

```tsx
transition-all duration-200
```

Use for modal open/close, input focus rings, and component visibility changes.

### Long Transition (progress bars, layout shifts)

```tsx
transition-all duration-500 ease-out
```

### Hover Reveal Pattern

Elements that should appear on hover use opacity transition:

```tsx
// The element
<span className="opacity-0 group-hover/parent:opacity-100 transition-opacity duration-200">

// The parent needs a group name
<div className="group/parent">
```

### Entry Animation (page-level)

```tsx
<div className="rise-in">  // uses @keyframes rise-in from index.css
```

---

## File Patterns

### Component Structure

Each major UI section is its own file with a clear prop interface:

```
section-name.tsx       → Main section component
section-name-row.tsx   → Repeated row item (if applicable)
primitives.tsx         → Shared layout primitives (FieldRow, FieldValue, SectionHeading)
helpers.ts             → Pure formatting/display functions
types.ts               → TypeScript interfaces
```

### Shared Primitives

Always use the primitives from `primitives.tsx` for consistent layout:

- `FieldRow` — Label + value grid row
- `FieldValue` — Formatted value with empty state
- `SectionHeading` — Uppercase heading with gradient divider

### Hover Patterns

| Pattern | Class |
|---|---|
| **Row hover** | `hover:bg-muted/30` |
| **Card hover** | `hover:bg-muted/25 hover:border-border/35` |
| **Button hover** | `hover:bg-muted/60 hover:text-foreground` |
| **Text hover** | `hover:text-foreground` |
| **Interactive reveal** | `opacity-0 group-hover:opacity-100` |

### Icon Sizing

| Context | Size |
|---|---|
| Inline with text | `size-3` (12px) |
| Standard buttons | `size-3.5` (14px) |
| Feature/empty state | `size-4` to `size-5` (16–20px) |
| Large illustration | `size-7` (28px) |
| Status dot | `size-[7px]` |
| Checkbox checkmark | `size-2.5` (10px) |

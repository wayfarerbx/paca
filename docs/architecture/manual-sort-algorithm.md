# Manual Sort Algorithm

## Overview

When a view's `sort_by` configuration is set to `"manual"` (or is unset), tasks are ordered by a floating-point `position` value stored in the `view_task_positions` table. This document describes how that position is computed, why `DOUBLE PRECISION` (float64) is used instead of `INTEGER`, and how to handle edge cases.

---

## Why Float64 Instead of Integer?

The naive approach of storing positions as consecutive integers (0, 1, 2, 3…) requires shifting every record after the insertion point whenever an item is moved. This is an O(n) write that becomes expensive at scale and creates race-condition windows.

A better approach is **fractional indexing**: keep the positions of all untouched items constant and only write the single record that moved. To insert a task between two neighbours, assign it the arithmetic midpoint of their positions.

With **integers**, midpoint insertion is fundamentally broken:

| After N halvings | Gap between positions |
|---|---|
| 0 | 1 000 |
| 1 | 500 |
| 2 | 250 |
| … | … |
| 10 | < 1 → **collision** |

Ten re-inserts into the same slot produce duplicate position values because `floor((n + n+1) / 2) == n`. Duplicate positions make the ordering undefined.

With **`DOUBLE PRECISION` (float64)**, the mantissa carries 52 bits of precision. Starting with an initial spacing of 2¹⁶ = 65 536, you can halve approximately **52 times** before adjacent positions collapse:

$$2^{16} \div 2^{52} \approx 10^{-11}$$

In practice that means **billions of drag-and-drop operations inside the same gap** before renormalisation is needed. For normal usage, renormalisation never occurs.

---

## Position Assignment Rules

All logic lives in `handleReorderTask` in [`interaction-layout.tsx`](../../apps/web/src/components/projects/interactions/interaction-layout.tsx).

All positions are confined to the open interval **(0, `Number.MAX_SAFE_INTEGER`)** by always computing midpoints toward the boundaries. This makes two classes of bug structurally impossible:

- **Negative positions** — the prepend formula is `next / 2`, which is always `> 0`.
- **Overflow** — the append formula is `(prev + POSITION_MAX) / 2`, which is always `< POSITION_MAX`.

Given:
- `prev` — the `view_position` of the task immediately above the drop slot (or `null`)
- `next` — the `view_position` of the task immediately below the drop slot (or `null`)
- `POSITION_MAX = Number.MAX_SAFE_INTEGER` (2⁵³ − 1 ≈ 9 × 10¹⁵)

| Scenario | Formula | Bound guarantee |
|---|---|---|
| Inserted between two tasks | `(prev + next) / 2` | stays inside `(prev, next)` |
| Appended after last task | `(prev + POSITION_MAX) / 2` | always `< POSITION_MAX` |
| Prepended before first task | `next / 2` | always `> 0` |
| No positioned neighbours yet | `POSITION_MAX / 2` | midpoint of full range |

> **No `Math.floor`** — positions are floating-point numbers. Truncating to an integer defeats the purpose of fractional indexing.

### Initial spacing and the unpositioned zone

New tasks that have never been manually positioned have `view_position = null`. They sort to the **bottom** of the group (after all explicitly positioned tasks) in `created_at` ascending order. This is the *unpositioned zone*.

#### Dragging within the unpositioned zone

When a drag lands next to a null-positioned task, the raw `view_position` of that neighbour is `null` and cannot be used in the midpoint formula. The algorithm assigns each null-positioned task in the group a **virtual position** in the range `(lastExplicit, POSITION_MAX)` based on its slot in the desired post-drag order:

$$\text{virtualPos}(i) = \text{lastExplicit} + \frac{(\text{POSITION\_MAX} - \text{lastExplicit}) \cdot (i+1)}{N+1}$$

where `i` is the 0-based index of the null task among its peers (ordered by `created_at`, which is their pre-drag order) and `N` is the total number of null non-moved tasks.

These virtual positions are then used in the midpoint formula exactly like real positions.

#### Materialisation

Using only the virtual position for the moved task is not enough — the other null-positioned tasks would revert to `created_at` order on the next render, destroying the order the user established.

Therefore, **when at least one immediate neighbour is null-positioned**, the algorithm also sends position updates for every other null-positioned task in the group, using their virtual positions. This *materialises* the unpositioned zone: after the first drag within it, every task in the group has an explicit `DOUBLE PRECISION` position and the standard fractional-indexing rules apply from that point on.

```
Before drag (null tasks in created_at order):
  [A(pos=1000), B(null,ca=1), C(null,ca=2), D(null,ca=3)]

User drags D before B:
  reordered = [A(1000), D, B, C]

  lastExplicit = 1000, null non-moved = [B,C]
  virtual B = 1000 + (MAX-1000)*1/3 ≈ MAX/3
  virtual C = 1000 + (MAX-1000)*2/3 ≈ MAX*2/3

  prev = pos(A) = 1000, next = virtual(B) ≈ MAX/3
  D.position = (1000 + MAX/3) / 2 ≈ MAX/6

Updates sent: D→MAX/6, B→MAX/3, C→MAX*2/3
Result: A(1000) < D(MAX/6) < B(MAX/3) < C(MAX*2/3) ✓
```

---

## Database Schema

### `view_task_positions`

```sql
CREATE TABLE view_task_positions (
    id        UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    view_id   UUID           NOT NULL REFERENCES sprint_views(id) ON DELETE CASCADE,
    task_id   UUID           NOT NULL,
    position  DOUBLE PRECISION NOT NULL DEFAULT 0,
    group_key TEXT,
    CONSTRAINT uq_view_task_positions_view_task UNIQUE (view_id, task_id)
);
```

Key design decisions:
- **One row per (view, task) pair** — the unique constraint prevents duplicates; upsert semantics update the position in-place.
- **`group_key`** — records which column/group panel the task belongs to (e.g. a status ID on a board view). A task that changes group gets a new position *and* a new `group_key` in the same upsert.
- **`DOUBLE PRECISION`** — 8-byte IEEE 754 float, sufficient for ~52 halvings per gap.

### `sprint_views`

The `position` column on `sprint_views` is also `DOUBLE PRECISION`. However, views use the *full-reorder* strategy (the client sends the complete ordered list via `PUT /views/positions`), so fractional indexing is not strictly necessary here. Float64 is kept for consistency.

---

## API Contract

### Bulk-move tasks (preferred)

```
PUT /projects/:projectId/views/:viewId/task-positions

Body:
{
  "items": [
    { "task_id": "uuid", "position": 98304.0, "group_key": "status-uuid-or-null" },
    { "task_id": "uuid", "position": 163840.0, "group_key": null }
  ]
}
```

Upserts all items in a **single database round-trip** (INSERT … ON CONFLICT DO UPDATE).  
Use this endpoint whenever a drag operation must write positions for more than one task (e.g. materialising virtual positions for previously-unpositioned neighbours).

### Move a single task

```
PUT /projects/:projectId/views/:viewId/task-positions/:taskId

Body:
{
  "position": 98304.0,    // float64, required
  "group_key": "status-uuid-or-null"  // optional
}
```

Single-task upsert. Still available but the bulk endpoint is preferred on the client because a drag that lands next to unpositioned tasks may need to materialise multiple rows.

### Read positions

Manual ordering is read from the shared view positions endpoint:

```text
GET /projects/:projectId/views/:viewId/task-positions
```

Each item returns the task id, position, and optional group key for that view. Tasks with no recorded position in the view sort after all positioned tasks on the client, using creation time as the fallback.

---

## Renormalisation

Renormalisation is the process of evenly redistributing all positions in a view to restore large gaps. It is **not yet automated** because the float64 space makes it unnecessary for typical usage.

### When is it needed?

Each insertion between two neighbours with positions `prev` and `next` halves the gap `next - prev`. Starting from a gap of `POSITION_MAX / 2 ≈ 4.5 × 10¹⁵`, a single slot can be halved approximately **52 times** before the gap drops below 1. That requires ~4.5 quadrillion drag-and-drops into the exact same slot — unreachable in practice.

### How to renormalise (if ever needed)

1. Fetch all `view_task_positions` for the view ordered by `position ASC`.
2. Assign new evenly-spaced positions: `POSITION_MAX / (count + 1) * index` for `index` in `1..count`.
3. Bulk-update all rows in a single transaction.

This can be triggered from a maintenance script or a future admin API endpoint.

---

## Sorting on the Client

```typescript
// In interaction-layout.tsx
const sortedTasks = useMemo(() => {
  if (isManualSort) {
    return [...tasks].sort((a, b) => {
      const pa = a.view_position;
      const pb = b.view_position;
      if (pa != null && pb != null) return pa - pb;
      if (pa != null) return -1;   // positioned tasks come first
      if (pb != null) return 1;
      return a.created_at.localeCompare(b.created_at); // fallback
    });
  }
  return sortTasksByConfig(tasks, activeViewConfig, viewCtx);
}, [isManualSort, tasks, activeViewConfig, viewCtx]);
```

Tasks that have never been positioned (`view_position == null`) are placed after all explicitly positioned tasks, ordered by creation date. This ensures newly created tasks always appear at the bottom until explicitly moved.

---

## Common Bugs (now fixed)

| Bug | Root cause | Fix |
|---|---|---|
| Tasks swap back after a few moves | `Math.floor((prev + next) / 2)` collapsed integer gaps in ~10 halvings | Removed `Math.floor`; use pure float midpoint |
| Position goes negative on repeated prepend | `max(0, next − 65536)` clamps to 0 when `next < 65536`; subsequent prepends all get 0 (collision) | Use `next / 2` — always positive, converges toward 0 but never reaches it |
| Position grows unboundedly on repeated append | `prev + 65536` accumulates without bound; could theoretically exhaust float64 range | Use `(prev + POSITION_MAX) / 2` — always strictly less than `POSITION_MAX` |
| First drag gives position 0 (`newIndex × 65536` when `newIndex = 0`) | Prepending before a 0-position task with old formula gave 0 again (collision) | Use `POSITION_MAX / 2` for no-neighbour case; prepend uses `next / 2` which avoids 0 |
| Position type mismatch between API and DB | `INTEGER` in DB vs `number` in TypeScript | Migrated to `DOUBLE PRECISION`; Go uses `float64` |
| Task keeps old group after column change | `group_key` not updated alongside `position` | Client always sends `group_key` in the upsert payload |

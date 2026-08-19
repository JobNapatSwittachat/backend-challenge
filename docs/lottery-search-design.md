# Lottery Ticket Search System — Design Proposal

> Design exercise only, per the challenge instructions. No code is implemented.

## 1. Problem Restatement

- **Dataset**: 10M+ lottery tickets. Each ticket carries a 6-digit number (`000000`–`999999`).
- **Query**: a 6-character pattern of digits and wildcards, e.g. `****23`, `1****5`, `123***`. Each position is either a fixed digit or `*`.
- **Constraint**: when many users search the same pattern at the same time, the same ticket must never be handed to two users simultaneously.
- **Goal**: fast search at 10M+ scale plus a correct, atomic allocation mechanism.

## 2. Key Insight: the Number Space Is Tiny

Although there are 10M+ *tickets*, there are only **10⁶ = 1,000,000 distinct numbers**. Tickets are instances of numbers (several tickets can share a number — different draws, series, or books).

This lets us split the problem into two independent tiers:

| Tier | Question | Size |
| --- | --- | --- |
| 1. Pattern matching | Which *numbers* match the pattern? | 1M-element space, fits in memory |
| 2. Allocation | Which *ticket* with one of those numbers does this user get? | 10M+ rows, needs atomicity |

Because every position in the pattern is either an exact digit or a full wildcard, a pattern is just a set of **equality constraints on digit positions**. A pattern with `k` wildcards matches exactly `10^k` numbers. There is no regex engine and no collection scan anywhere in the hot path.

## 3. Architecture Overview

```
                       ┌─────────────────────┐
User ── pattern ─────► │  Search/Allocation   │
                       │  Service (stateless, │
                       │  N instances)        │
                       └────┬──────────┬─────┘
             Tier 1         │          │        Tier 2
   ┌────────────────────────▼───┐  ┌───▼──────────────────────────┐
   │ In-memory positional       │  │ PostgreSQL (source of truth) │
   │ bitmap index (per instance)│  │ tickets + atomic reservation │
   │ pattern → matching numbers │  │ via FOR UPDATE SKIP LOCKED   │
   └────────────────────────────┘  └──────────────────────────────┘
```

## 4. Tier 1 — Wildcard Matching: Positional Bitmap Index

Maintain 60 bitmaps (**6 positions × 10 digits**), each 1M bits, where bitmap `B[p][d]` has bit `n` set iff number `n` has digit `d` at position `p` **and at least one available ticket exists** for `n`.

- Memory: 60 × 1M bits ≈ **7.5 MB raw** (far less with Roaring compression). Trivial to hold in every service instance.
- Query: AND together the bitmaps of the fixed positions. `1****5` → `B[0][1] AND B[5][5]`. Bitwise AND over 1M bits is microseconds on modern CPUs.
- Result: the set of matching numbers that still have availability, ordered/sampled however product wants (e.g. random sample for fairness).
- Maintenance: when the last available ticket of a number is allocated, clear bit `n`; when a reservation expires, set it back. Instances refresh via a change feed (Postgres LISTEN/NOTIFY or CDC) — the bitmap is a *hint*, so brief staleness is safe: correctness is enforced in Tier 2.

**Simpler fallback (no bitmap infra):** enumerate the `10^k` matching numbers directly from the pattern (k ≤ 6, worst case 1M strings, still cheap) and query availability per number. For patterns with ≤ 4 wildcards this is at most 10,000 numbers — negligible. The bitmap index is an optimization for wildcard-heavy patterns and for filtering out sold-out numbers up front.

### Why not other indexing approaches

- **`LIKE`/regex over 10M rows**: a pattern like `**3*4*` cannot use a B-tree; it degenerates to a scan. Rejected for the general case.
- **B-tree on `number` + B-tree on `reverse(number)`**: covers prefix (`123***`) and suffix (`***123`) patterns nicely, but not mixed patterns (`1**4*5`). Usable as a complement, but the positional approach covers *all* pattern shapes uniformly, so we standardize on it.
- **Search engines (Elasticsearch)**: workable (indexed n-grams / per-position keyword fields) but heavy operational cost for a problem that fits in 8 MB of RAM. Rejected as unnecessary.

## 5. Tier 2 — Storage & Atomic Allocation

### Recommended store: PostgreSQL

| Criterion | Why PostgreSQL |
| --- | --- |
| Correctness | Real ACID transactions and row locks make "no duplicate simultaneous assignment" provable, not probabilistic. |
| Concurrency | `FOR UPDATE SKIP LOCKED` was designed for exactly this: concurrent workers pulling from a shared pool without blocking each other. |
| Scale | 10M rows is small for Postgres. A ticket row (~50 bytes + indexes) puts the whole table comfortably in shared buffers on a modest instance. |
| Operations | Boring, well-understood HA/backup/monitoring story; every team can run it. |

Schema sketch:

```sql
CREATE TABLE tickets (
  id             BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  number         CHAR(6)  NOT NULL,           -- '042317'
  status         SMALLINT NOT NULL DEFAULT 0, -- 0 available, 1 reserved, 2 sold
  reserved_by    UUID,
  reserved_until TIMESTAMPTZ,
  draw_id        INT NOT NULL
);

-- Partial index: the hot path only ever looks at available tickets.
CREATE INDEX idx_tickets_available ON tickets (number) WHERE status = 0;
```

### Allocation algorithm (single atomic statement)

```sql
WITH picked AS (
  SELECT id FROM tickets
  WHERE number = ANY($matching_numbers)   -- from Tier 1
    AND status = 0
  LIMIT $n
  FOR UPDATE SKIP LOCKED                  -- concurrent users skip, never share
)
UPDATE tickets t
SET status = 1, reserved_by = $user_id, reserved_until = now() + interval '5 minutes'
FROM picked WHERE t.id = picked.id
RETURNING t.id, t.number;
```

Why this is correct under concurrency:

- `FOR UPDATE` row-locks each picked ticket inside the transaction; **two transactions can never lock the same row**, so a ticket cannot be returned to two users — this is the duplicate-prevention guarantee, enforced by the database rather than by application code.
- `SKIP LOCKED` makes concurrent searchers for the *same pattern* skip already-claimed rows and take the next match instead of blocking — high throughput, no lock queues, no deadlocks from this statement.
- The partial index keeps the candidate scan tight even when most of a number's tickets are sold.

### Reservation lifecycle

1. **Reserve** (above): ticket shown to the user, held for a TTL (e.g. 5 minutes).
2. **Confirm** (purchase): `UPDATE ... SET status = 2 WHERE id = $id AND reserved_by = $user AND status = 1`.
3. **Expire**: a small background job (or lazy check at read time) returns stale reservations to the pool: `UPDATE ... SET status = 0, reserved_by = NULL WHERE status = 1 AND reserved_until < now()`, then re-sets the Tier 1 bits.

Idempotency: confirm/release are keyed by `(ticket_id, reserved_by)`, so client retries are safe.

### High-throughput variant: Redis allocation layer

If flash-sale traffic (e.g. tens of thousands of allocations/sec on popular patterns) exceeds what a single Postgres primary should absorb, put an allocation cache in front:

- One Redis **set per number**: `avail:{number} = {ticket_ids...}`.
- Allocation = `SPOP avail:{number}` — atomic by Redis's single-threaded execution, so no duplicate handout is possible; a Lua script can pop across several matching numbers atomically.
- Redis holds only the *available* pool (worst case 10M small IDs ≈ a few hundred MB); Postgres remains the durable source of truth, updated asynchronously via an outbox/stream, and rebuilds Redis on restart.

Trade-off: adds an infrastructure component and an eventual-consistency edge (a crash between SPOP and persistence can temporarily strand a ticket until reconciliation). Recommendation: **start with Postgres-only** — it comfortably handles thousands of allocations/sec — and add the Redis layer only if measured load demands it.

## 6. Performance Analysis

| Step | Cost | Notes |
| --- | --- | --- |
| Pattern → numbers (bitmap AND) | µs, in-process | 6 × 1M-bit ANDs worst case; no I/O |
| Availability pre-filter | free | encoded in the same bitmaps |
| Allocation query | ~1 ms | index lookup on `(number) WHERE status=0` + row locks on `n` rows |
| Throughput | thousands of reservations/sec on one Postgres primary | `SKIP LOCKED` avoids lock convoys; hot patterns scale because matching numbers spread across many rows |
| Memory | ~8 MB/instance (bitmaps) | rebuildable from the DB in seconds |

Scaling levers, in order: read replicas for non-allocating searches → table partitioning by `draw_id` (old draws age out) → the Redis allocation layer → sharding by leading digit(s) of `number` if ever needed (allocation never crosses numbers, so sharding is clean).

## 7. Failure Modes & Mitigations

- **Service instance dies mid-flow**: reservation TTL returns tickets to the pool automatically.
- **Stale bitmap says a number is available when it isn't**: allocation query simply returns fewer rows; service retries with the next numbers. Correctness never depends on the bitmap.
- **Clock issues on TTL expiry**: use DB time (`now()`) exclusively, never application clocks.
- **Thundering herd on one pattern**: `SKIP LOCKED` degrades gracefully — contention cost is skipping locked rows, not queuing on them; randomizing candidate-number order across requests spreads load further.

## 8. Summary

Exploit the small number space: answer *"which numbers match"* with an in-memory positional bitmap index (microseconds, all pattern shapes), and answer *"which ticket does this user get"* with a single atomic `FOR UPDATE SKIP LOCKED` reservation in PostgreSQL (database-enforced exclusivity, TTL-based reservations). Simple to operate, provably correct on the duplicate-assignment constraint, and with a clear scaling path (replicas → partitioning → Redis `SPOP` layer) if traffic outgrows the baseline.

# Architecture

## Why integers?

Money is an integer in the currency's minor unit, never a float. Floating-point arithmetic introduces rounding errors that are unacceptable in financial systems. Using an integer amount explicitly prevents these kinds of cumulative precision issues.

## Why are facts never edited?

To follow standard ledger practices, facts represent immutable economic events. Once recorded, a fact is never altered. Instead, mistakes are corrected by emitting a `Reversed` fact. This creates a clean, append-only history that is fully auditable.

## The Idempotency Key

Every fact has a deterministic `Key()` composed of its `Kind` and the `Type` and `ID` of its `Source`. This guarantees that if a process retries or duplicates a message, the receiving `Sink` can recognize and discard it if it was already processed.

## Plugging in Accounting

This module is not an accounting engine itself; rather, it formalizes the "money events" across the entire system. A later accounting module simply implements the `Sink` port to receive all these facts. It will translate them into standard double-entry bookkeeping rules without any operational module needing to understand what a debit or credit is.

## Reconcile: The Safety Net

The `Reconcile` function provides a safety net against lost events. Because `Accept` on a sink is fully idempotent, an application can periodically ask a `Replayer` to re-emit all facts in a given time window. Any fact that the `Sink` missed will be processed, while facts it already knows about will safely have no effect.

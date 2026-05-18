# Architectural Decisions: Fix Shadow Registry API

## Decision 1: Handle Inactive Source Objects in Mutating Actions
**Context:** When a source object is marked as `is_active = false`, the API returned a 404 error because the internal lookup required `is_active = TRUE`. This caused confusion in the UI.

**Decision:** We activated the object manually to unblock the user. In the future, the UI should probably disable buttons for inactive objects or the API should return a more descriptive 403/400 error instead of a generic 404.

**Status:** Implemented (Manual DB Update).

## Decision 2: API Unification (Phase 4)
**Context:** The system is migrating from legacy `/api/` to `/api/v1/`.
**Decision:** We verified that `create-default-columns` is correctly registered under `/api/v1/` and that the 404 was NOT a routing issue. We maintained the current Phase 4 routing strategy.

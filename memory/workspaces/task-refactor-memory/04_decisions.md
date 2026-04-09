# 04_decisions.md - Decision Log

## ADR-001: Split Global vs Feature
- **Context**: Old `active_plans.md` mixed everything. Hard to switch tasks.
- **Decision**: Split into `global/` (static) and `workspaces/<feature>/` (dynamic).
- **Consequence**: Better context isolation. Requires stricter file management.

## ADR-002: Consolidate DB/Docs
- **Context**: `use-db` and `work-desc` folders were disconnected using distinct paths.
- **Decision**: Move key context from these into `db_context.md` and `docs_context.md` inside the feature workspace.
- **Consequence**: "One-stop shop" for feature context.

## ADR-003: Explicit Triggers (Fix)
- **Context**: Removed Brain/Muscle references in V2.0, causing confusion.
- **Decision**: Re-added explicit "Who does what" in V2.1 `context-manager`.
- **Consequence**: Restored ROLE compliance.

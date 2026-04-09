# Active Plan: Refactor Agent Memory

## Goal
Restructure `agent/memory` to support Global vs Feature-Specific contexts and update workflows.

## Status
- Core Structure Created: [x]
- Global Files Migrated: [x]
- Workflows Updated (Initial): [x]
- **Fixing Workflow Context**: [In Progress]
- **Dogfooding (Self-Workspace)**: [In Progress]

## Tasks
- [x] Analyze & Design V2 Structure
- [x] Create folders `global` and `workspaces/feature-goopay-backend`
- [x] Migrate `project_context`, `tech_stack` to `global`
- [x] Migrate `active_plans` to `feature-goopay-backend`
- [x] Update `brain-delegate.md`
- [ ] Fix `context-manager.md` to include Brain/Muscle roles
- [ ] Verify using this workspace (`task-refactor-memory`)

## Next Steps
- Hoàn thiện fix context-manager.
- Verify lại toàn bộ flow.

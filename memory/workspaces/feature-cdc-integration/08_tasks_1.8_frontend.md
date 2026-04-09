# Task: Phase 1.8 - CMS Frontend Execution

Chi tiết các bước thực hiện hoàn thiện giao diện quản trị CDC.

- `[x]` 1. Infrastructure Setup:
    - `[x]` Create `/Users/trainguyen/Documents/work/cdc-cms-web/.env`
    - `[x]` Fix `CMS_API` port in `src/services/api.ts` (8090 -> 8080)
- `[ ]` 2. Feature: Schema Approval
    - `[ ]` Add Loading states to Approve/Reject buttons in `SchemaChanges.tsx`
    - `[ ]` Verify payload structure for `/api/schema/approve`
- `[ ]` 3. Feature: Table Registry
    - `[ ]` Verify "Standardize" and "Discover" button handlers in `TableRegistry.tsx`
    - `[ ]` Test "Bulk Import" with sample JSON
- `[ ]` 4. Build & Verify:
    - `[ ]` Run `npm run build` to ensure no TS errors
    - `[ ]` Manual verification of Login + Dashboard flow
- `[ ]` 5. Governance:
    - `[ ]` Update `05_progress.md` after each step
    - `[ ]` Record any UX decisions in `04_decisions.md`

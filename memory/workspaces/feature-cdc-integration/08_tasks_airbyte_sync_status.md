# Task: Airbyte Active/Inactive Sync Execution

Đồng bộ trạng thái `is_active` của Table Registry sang Airbyte Connection.

- `[ ]` 1. Backend Implementation:
    - `[ ]` Modify `RegistryHandler.Update` in `internal/api/registry_handler.go`
    - `[ ]` Implement `syncIsActiveWithAirbyte` helper method
    - `[ ]` Fetch Connection -> Update Stream Selection -> Update Connection
- `[ ]` 2. Audit UI features:
    - `[ ]` Check `Priority` sync
    - `[ ]` Check `SyncEngine` sync
    - `[ ]` Check `Mapping Rules` sync
- `[ ]` 3. Verification:
    - `[ ]` Test Toggle Active -> Check Airbyte
    - `[ ]` Test Toggle Inactive -> Check Airbyte
- `[ ]` 4. Governance:
    - `[ ]` Update `05_progress.md`
    - `[ ]` Update `04_decisions.md`

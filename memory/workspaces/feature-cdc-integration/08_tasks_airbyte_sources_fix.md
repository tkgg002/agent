# Task: Fix Airbyte Sources 500 Error

- [ ] `[ ]` **Phase 1: Configuration Updates**
    - [ ] `[ ]` Add `WorkspaceID` to `AirbyteConfig` in `config/config.go`
    - [ ] `[ ]` Update `NewConfig` to handle `AIRBYTE_WORKSPACE_ID` env override
- [ ] `[ ]` **Phase 2: Airbyte Client Enhancement**
    - [ ] `[ ]` Update `airbyte.Client` struct with `workspaceID`
    - [ ] `[ ]` Update `NewClient` and `getDefaultWorkspaceID` logic
- [ ] `[ ]` **Phase 3: Wiring & Config**
    - [ ] `[ ]` Pass `WorkspaceID` in `internal/server/server.go`
    - [ ] `[ ]` Update `config-local.yml`
- [ ] `[ ]` **Phase 4: Verification**
    - [ ] `[ ]` Verify with `curl`
    - [ ] `[ ]` Check logs

# Status

- Workspace: `feature-cdc-schema-design-v2`
- Phase: Design / Architecture
- Current Status: Completed design proposal
- Output:
  - Detailed V2 schema proposal
  - Mapping from legacy tables to V2
  - Project-level implementation plan for `centralized-data-service`
- Next Recommended Step:
  - Tách task implementation thành Phase 1:
    - migrations V2
    - model/repo scaffolding
    - connection manager

## 2026-04-24 Update

- Current Status: Cutover-ready for namespace/bootstrap
- Control plane:
  - `cdc_system` là nguồn sự thật chính cho system tables
- Data plane:
  - shadow theo `shadow_<source_db>`
  - master theo binding/schema đích
- Remaining caution:
  - `public` schema của PostgreSQL vẫn tồn tại ở mức engine, nhưng app system tables không nên còn nằm ở đó sau migration đầy đủ.

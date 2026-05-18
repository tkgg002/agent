# Solution Notes

- Replace Flow1-heavy connector setup with a single operator-facing `/sources` page:
  - `New Connect` modal with per-database fields for MongoDB, MySQL, PostgreSQL.
  - connection list joined with runtime connector state and source fingerprint state.
  - `Edit Config` pushes updates back to active Kafka Connect connector and refreshes stored fingerprint.
- Add CMS API support for connector config mutation:
  - Kafka Connect client `PUT /connectors/:name/config`
  - command handler `system-connector.update-config`
  - HTTP endpoint `PATCH /api/v1/system/connectors/:name/config`
- Rename UI concept from registry to shadow:
  - `/shadow` becomes primary route
  - old `/registry` routes redirect to `/shadow`
  - menu label and TableRegistry copy updated accordingly
- Hide/remove Flow1 pages from FE to stop users from entering broken wizard paths.
- Root-cause fixes discovered during browser validation:
  - stale CMS runtime binary missing PATCH route
  - invalid fingerprint status value `updated` violating `cdc_system.sources` constraint

# Tasks Checklist - Fix Ambiguous Transform Scope (V2 Hybrid)

- [ ] Modify `MetadataRegistryService.ReloadAll` to cache both flat and qualified names.
- [ ] Update `ResolveTargetSchema` and `ResolveTargetTableConfig` to parse fully qualified target table names.
- [ ] Update `BatchTransformHandler.runTransformJob` to support qualified target names, extract the schema and pure table segments, and resolve mapping rules uniquely by source object ID.
- [ ] Enrich `TableRegistryRepo.GetAllActive` to join `shadow_binding` and populate `ShadowSchema` dynamically.
- [ ] Update scheduler to publish `QualifiedTarget()` table names.
- [ ] Update `cdc-cms-service` manual transform trigger to publish fully qualified target table names in the NATS payload.
- [ ] Verify using unit tests in `centralized-data-service` and `cdc-cms-service`.

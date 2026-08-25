# Requirements - Fix Ambiguous Transform Scope (V2 Hybrid)

## Problem Description
Currently, multiple active shadow bindings can share the same `target_table` name under different schemas (e.g. `reconcile_final` in `shadow_test33`, `shadow_testsftp30`, etc.).
1. In `MetadataRegistryService`, `targetCache` and `targetRouteMap` key on the flat `TargetTable` string name. This causes collision, where the map key gets overwritten by the last loaded config. Only one schema wins the cache key.
2. In `BatchTransformHandler` (worker), when a transform message is processed, it resolves schema based on the flat table name, which routes to the wrong (colliding) schema. It also queries mapping rules by source table name flat string, fetching rules for all 6 source objects.
3. In `cdc-cms-service` and `centralized-data-service` scheduler, transform commands are published with only the flat target table name, which makes it impossible for the worker to resolve the correct target schema uniquely.

## Goals
1. Support fully qualified target table names (e.g., `shadow_schema.target_table`) in the cache maps `targetCache` and `targetRouteMap` within `MetadataRegistryService`.
2. Update helper function `ResolveTargetSchema` and `ResolveTargetTableConfig` to parse fully qualified target table names.
3. Update `BatchTransformHandler.runTransformJob` to support fully qualified target table names, parse them into schema and table segments, resolve mapping rules uniquely using `ListActiveBySourceObject` if the route is found (instead of flat table query), and execute SQL commands on the correct schema.
4. Update the scheduler to query and publish `QualifiedTarget()` table names.
5. Update `cdc-cms-service` manual transform trigger to publish `QualifiedTarget()` in the payload.

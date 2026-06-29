# Plan: Bug Data Integrity Missing Tables

## Phase 1: Research & Root Cause Analysis
1. Locate where the tables (master and shadow) are loaded or defined in `cdc-cms-service` (backend).
2. Locate how `cdc-cms-web` requests and renders these tables at `/data-integrity`.
3. Investigate why "Shadow" tables are not showing (backend filter, database empty, or frontend rendering issue).
4. Investigate why `master_centrallized_export_service.export_jobs` and `master_centrallized_export_service_2.export_jobs` are missing (are they filtered out, is there a schema configuration/mapping rule missing, or connection issue?).
5. Propose a solution based on findings.

## Phase 2: Implementation (Pending Research)
*TBD after research phase*

## Phase 3: Verification (Pending Research)
*TBD after research phase*

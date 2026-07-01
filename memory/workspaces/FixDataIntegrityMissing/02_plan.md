# Plan - FixDataIntegrityMissing

1. **Step 1: Code Search in cdc-cms-service**
   - Find all routes or handler methods in `cdc-cms-service` containing `/data-integrity` or returning statistics about reconciliation/sync/data integrity.
   - Examine how it retrieves the list of shadow tables and master tables.
2. **Step 2: Investigate cdc-cms-web Frontend**
   - Look up the routing, store, or view component for `/data-integrity` in `cdc-cms-web`.
   - See how the API response is mapped and if shadow tables are filtered out or if they are missing from the API response itself.
3. **Step 3: Analyze the database/repository queries**
   - Look at the tables/metadata in `cdc_system` or mappings configuration.
   - Find why `master_centrallized_export_service.export_jobs` and `master_centrallized_export_service_2.export_jobs` are missing. Are they skipped due to a pattern match, schema name mismatch, or active state status?
4. **Step 4: Formulate and implement the fixes**
   - Correct the logic in backend/frontend.
   - Verify the changes.

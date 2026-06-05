# Context: Fix CMS Pipeline Bugs

Current state:
The CDC CMS pipeline has 4 main bugs:
1. Mapping inheritance issue: Mapping rules were being inherited globally by `source_object_id` instead of being scoped to `shadow_binding_id`. (Backend rules fetching scope is being fixed).
2. Missing Metadata `source_data_type` & wrong `Status` usage: We added `source_data_type` to DB, now updating worker to infer this type when scanning raw data.
3. Remove "Preview" and "Backfill" buttons from UI.
4. Snapshot V2 error: `shadow_binding_id=4 not in active registry routes`. The user clarified that the binding IS active in the DB (`is_active=true`), so the issue is NOT about inactive bindings. The error is a bug in the code that loads or queries the active routes. We need to find the root cause instead of using a "cheat" synthetic route.

Goal:
Resolve these 4 bugs completely, maintaining system integrity, and adhering strictly to the user's workflow rules.

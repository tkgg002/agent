# 09 Technical Solutions — Force Transform & Bug Fix

## 1. Solution for Bug FE Transform In-Progress Collision
- **Problem:** `activeTransformJobs` map stored entries by `record.id`. For child binding tables, `r.source_object_id` was erroneously used as the index, which matched the parent's `record.id`.
- **Solution:** Switched the child table render prop to index via `activeTransformJobs[r.id]`, where `r.id` is the unique binding ID.

## 2. Solution for BuildCastExpr Fallback for `timestamptz`
- **Problem:** When casting timestamp strings from `_raw_data`, `BuildCastExpr` used `::TIMESTAMP` for all timestamp variants in the ELSE branch, stripping tz offsets.
- **Solution:** Separated `"timestamptz"` and `"timestamp with time zone"` into a dedicated case that evaluates fallback string conversions using `::TIMESTAMPTZ`.

## 3. Solution for Force Transform Mode
- **Problem:** Regular batch transform requires fields to be `NULL` (`col IS NULL`), which skips records already populated after an ALTER TYPE.
- **Solution:**
  - Extended `BatchTransformPayload` with `Force: bool` and `ForceFields: []string`.
  - When `Force: true`:
    1. Worker strictly filters rules matching `ForceFields`.
    2. WHERE condition becomes `TRUE` instead of checking `col IS NULL`.
    3. The chunked CTE loop iteratively executes without lock issues across all records.

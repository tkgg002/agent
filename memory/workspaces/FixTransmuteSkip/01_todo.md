# Todo - FixTransmuteSkip

- [x] Investigate counts in shadow table vs master table to check duplication or status.
- [x] Determine the exact cause of 1979 skips:
  - [x] Is it due to duplicate `_id` values? (No, verified that all `_id`s in shadow table are unique)
  - [x] Is it due to validation or formatting skips? (Yes, encoding/format issue)
- [x] Verify why `occ_skipped` was `1979` in the successful run (ID 3379) but `0` in the degraded errors.
- [x] Provide a clear and detailed explanation to the user.
- [x] Support updating `_deleted = true` in master table on delete events in transmute strategies.


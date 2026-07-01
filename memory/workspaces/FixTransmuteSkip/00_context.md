# Context - FixTransmuteSkip

## Background
- During the sync run for binding code `mb_master_payment_bill_service_payment_bills_1782715583` (`payment_bills`), the logs show `scanned: 39979, skipped: 1979, updated: 38000, inserted: 0`.
- The user is asking why exactly 1979 rows were skipped (`sao skip tận 1979`).
- We need to investigate whether this skip is expected (e.g. due to duplicate records, OCC conflict checks, or format/validation issues) and verify it mathematically.

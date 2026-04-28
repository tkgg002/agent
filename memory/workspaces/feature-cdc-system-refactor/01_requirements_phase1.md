# Requirements Phase 1

- Unify reconciliation hashing to `ID|UnixMilli` with `xxhash64`.
- Heal orphaned destination rows using Tier 2 `missingFromSrc`.
- Force Mongo healing reads to primary.
- Replace advisory lock key generation with stable 32-bit hash.
- Make DLQ retry loop non-blocking and transactional.
- Add DLQ state machine worker for replay.
- Route UPSERT SQL generation through `SchemaAdapter.BuildUpsertSQL`.
- Add schema drift alert suppression cache.
- Harden coercion and CDC insert preparation logic.


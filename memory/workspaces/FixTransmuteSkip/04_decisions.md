# Decisions - FixTransmuteSkip

## Decision: Coerce Scalar Values to String for Text/Varchar Destination Columns

- **Context**: In `payment-bill-service.payment-bills` mappings, the `completedAt` column in the master table is typed as `TEXT`. However, some documents in MongoDB store it as a bare epoch integer (e.g. `1782717323818`). When transmuted, this value remains `int64` because the mapping rule type resolves to `TEXT` (not `TIMESTAMP`). Passing `int64` to PGX parameter binding for a Postgres `TEXT` column raises encoding errors.
- **Decision**: Update `coerceForColumn` to automatically convert any non-string, non-nil scalar type (including `int64`, `float64`, `bool`, and `time.Time`) into its string representation when the destination column's data type is `TEXT`, `VARCHAR`, or `CHAR`.
- **Alternatives considered**:
  - Changing the mapping rule in `mapping_rule_master` to `TIMESTAMP`: Rejected, as the target table column in the master table is actually `TEXT` and we cannot alter the database schema or force specific data types on the destination dynamically here.
  - Adding type conversion transforms in GORM: Rejected, too complex and doesn't solve the core issue of robust coercion in the transmuter.
- **Status**: Implemented and verified via unit tests.

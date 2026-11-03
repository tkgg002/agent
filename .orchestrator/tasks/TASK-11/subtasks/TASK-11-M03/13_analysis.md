# 13 Analysis: TASK-11-M03 (`cdc-cms-web`)
| Hạng mục | Phân loại | Bằng chứng `file:line` trên đĩa |
| :--- | :---: | :--- |
| Điểm chạm mã nguồn `cdc-cms-web/src/constants/postgresDataTypes.ts:1 (export const POSTGRES_DATA_TYPES = [)` | `VERIFIED` | `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/constants/postgresDataTypes.ts` (missing on disk and not in diff.patch: cdc-cms-web:src/constants/postgresDataTypes.ts) |
| Hợp đồng Đầu vào (`Wave 1`) | `VERIFIED` | `UpstreamReady = false` (DependsOn: `TASK-11-M01, TASK-11-M02`) |
| Nhánh cô lập (`Isolated Branch`) | `VERIFIED` | `task/TASK-11/task-11-m03` (`.orchestrator/worktrees/TASK-11/cdc-cms-web`) |

## Trích xuất mã nguồn thực tế trên đĩa (`/Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/constants/postgresDataTypes.ts`)
```typescript
// VERIFIED ON-DISK SOURCE: /Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/constants/postgresDataTypes.ts (lines 1-25)
   1 | export const POSTGRES_DATA_TYPES = [
   2 |   'VARCHAR',
   3 |   'VARCHAR(50)',
   4 |   'VARCHAR(100)',
   5 |   'VARCHAR(255)',
   6 |   'CHAR',
   7 |   'TEXT',
   8 |   'INTEGER',
   9 |   'BIGINT',
  10 |   'SMALLINT',
  11 |   'NUMERIC',
  12 |   'DECIMAL(18,6)',
  13 |   'DOUBLE PRECISION',
  14 |   'REAL',
  15 |   'BOOLEAN',
  16 |   'TIMESTAMP',
  17 |   'TIMESTAMPTZ',
  18 |   'DATE',
  19 |   'JSONB',
  20 |   'JSON',
  21 |   'BYTEA',
  22 | ] as const;
  23 | 
  24 | export type PostgresDataType = typeof POSTGRES_DATA_TYPES[number];
  25 | 
```

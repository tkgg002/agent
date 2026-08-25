# 09_tasks_solution_gp2_3667.md - Hồ sơ Giải pháp Kỹ thuật Cụ thể (Technical Solutions)

## Giải pháp Tối ưu CDC MongoDB -> PostgreSQL cho Transaction History

### 1. DDL Schema Master Table `transaction_history` (PostgreSQL)
```sql
CREATE TABLE IF NOT EXISTS public.transaction_history (
    id VARCHAR(64) PRIMARY KEY,
    trans_code VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    merchant_id VARCHAR(64),
    amount BIGINT NOT NULL DEFAULT 0,
    fee BIGINT NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL,
    trans_type VARCHAR(32) NOT NULL,
    extra_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Tối ưu Index cho truy vấn lịch sử giao dịch hiệu năng cao
CREATE UNIQUE INDEX IF NOT EXISTS idx_trans_his_trans_code ON public.transaction_history(trans_code);
CREATE INDEX IF NOT EXISTS idx_trans_his_user_created ON public.transaction_history(user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trans_his_merchant_created ON public.transaction_history(merchant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_trans_his_status_created ON public.transaction_history(status, created_at DESC);
```

### 2. Định nghĩa Mapping Rules trong CDC Engine
Chuyển đổi dữ liệu BSON ExtJSON từ MongoDB sang PostgreSQL types:
- `_id` (`{$oid: "..."}`) -> `id` (`VARCHAR`)
- `trans_code` -> `trans_code` (`VARCHAR`)
- `user_id` -> `user_id` (`VARCHAR`)
- `merchant_id` -> `merchant_id` (`VARCHAR`)
- `amount` (`{$numberLong: "..."}`) -> `amount` (`BIGINT`)
- `fee` -> `fee` (`BIGINT`)
- `status` -> `status` (`VARCHAR`)
- `trans_type` -> `trans_type` (`VARCHAR`)
- `created_at` (`{$date: "..."}`) -> `created_at` (`TIMESTAMPTZ`)
- `updated_at` (`{$date: "..."}`) -> `updated_at` (`TIMESTAMPTZ`)
- `extra_data` -> `extra_data` (`JSONB`)

### 3. Cấu hình Connection Keys & Metadata Integrity
Tuân thủ nghiêm ngặt bộ ba định danh Metadata:
- `connection_key`: `mongodb_local` / `postgres_master`
- `schema`: `public` (PostgreSQL Master Schema) / `gpaylocal` (MongoDB Database Name)
- `table`: `transaction_history`

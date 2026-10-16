# Báo Cáo Phân Tích Gốc Rễ: Bug JSON String Parsing tại `ReconPipelineGrid.tsx` Khiến Mọi Update Biến Thành Insert

**Thời điểm phát hiện:** 2026-10-07 16:15:00 +07:00  
**Tác giả:** Brain (Architect)  
**Trạng thái:** Đã xác minh 100% qua Database và Codebase thực tế (Không suy diễn)

---

## 1. Hiện Tượng (Symptom)
- Người dùng thực hiện **UPDATE 1 bản ghi** thực tế trên luồng CDC ngày hôm nay (`2026-10-07`).
- Bảng thống kê luồng đối soát hàng ngày (`ReconPipelineGrid.tsx` -> Daily Flow Modal) hiển thị:
  ```text
  2026-10-07 | 3 3 0 0 | 3 3 0 0 | 3 0 0 0 | 0 0 0 0
  ```
  - **Kafka Ingest:** Total = 3, c = 3, u = 0, d = 0 (Báo 3 Insert, 0 Update)
  - **Shadow Ingest:** Total = 3, c = 3, u = 0, d = 0 (Báo 3 Insert, 0 Update)
  - **Master Transmute:** Total = 3, Inserted = 0, Updated = 0, Skipped = 0 (Báo 0 Insert, 0 Update)
  - **DLQ / Drift:** 0 0 0 0

---

## 2. Truy Vết Cơ Sở Dữ Liệu Thực Tế (Physical Ground Truth)

### A. Kiểm tra `cdc_dw.cdc_system.cdc_activity_log`
Query thực tế trên container `gpay-postgres-cdc`:
```sql
SELECT id, operation, target_table, rows_affected, details::text, started_at
FROM cdc_system.cdc_activity_log
WHERE started_at::text LIKE '2026-10-07%' AND target_table LIKE '%export_jobs%'
ORDER BY id ASC;
```

**Kết quả thu được:**
1. **ID 107079 (04:06:41 UTC):**
   - `operation`: `kafka-consumer`
   - `rows_affected`: 1
   - `details`: `{"written": 1, "trace_id": "6d644b3c...", "batch_size": 3}` (Code cũ chưa đếm c,u,d)
2. **ID 107080 (04:06:42 UTC):**
   - `operation`: `transmute`
   - `rows_affected`: 1
   - `details`: `{"scanned": 1, "skipped": 0, "updated": 0, "inserted": 1, ...}` -> **Inserted = 1**
3. **ID 107084 (04:06:49 UTC):**
   - `operation`: `kafka-consumer`
   - `rows_affected`: 1
   - `details`: `{"written": 1, "trace_id": "ecf61fd4...", "batch_size": 3}` (Code cũ chưa đếm c,u,d)
4. **ID 107085 (04:06:50 UTC):**
   - `operation`: `transmute`
   - `rows_affected`: 1
   - `details`: `{"scanned": 1, "skipped": 0, "updated": 0, "inserted": 1, ...}` -> **Inserted = 1**
5. **ID 107595 (08:16:54 UTC — 15:16 Chiều nay, thời điểm User test UPDATE):**
   - `operation`: `kafka-consumer`
   - `rows_affected`: 1
   - `details`: `{"c": 0, "d": 0, "u": 1, "written": 1, "batch_size": 1, "buffer_type": "cdc-stream", "is_snapshot": false}` -> **Backend CDS ĐÃ GHI ĐÚNG u: 1, c: 0, d: 0!**
6. **ID 107596 (08:16:55 UTC):**
   - `operation`: `transmute`
   - `rows_affected`: 1
   - `details`: `{"scanned": 1, "skipped": 0, "updated": 1, "inserted": 0, ...}` -> **Transmute ĐÃ GHI ĐÚNG updated: 1, inserted: 0!**

### B. Kiểm tra bảng Shadow vật lý `shadow_traitestces.export_jobs`
- Record `_id = '6ac5c54caaad9d67d4844dbd'`:
  - `_created_at`: `2026-10-07 04:06:41`
  - `_updated_at`: `2026-10-07 08:16:54`
- **Kết luận:** Backend CDS và Database ĐÃ HOẠT ĐỘNG HOÀN TOÀN CHÍNH XÁC:
  - Bản ghi được Insert vào buổi sáng (04:06).
  - Bản ghi được Update vào buổi chiều (08:16 UTC = 15:16 GMT+7).
  - `cdc_activity_log` ĐÃ GHI NHẬN `u: 1` và `updated: 1`!

---

## 3. Nguyên Nhân Cốt Lõi (Root Cause)

Điểm đứt gãy nằm hoàn toàn ở tầng Frontend: **`cdc-cms-web/src/components/ReconPipelineGrid.tsx`**.

### Bước 1: Wire Contract API trả về JSON String
Trong backend Go `cdc-cms-service`:
`internal/app/queries/system/activity_log_read_models.go`:
```go
type ActivityLogRow struct {
    ...
    Details *string `json:"details"`
    ...
}
```
Và trong `cdc-cms-web/src/hooks/useReconStatus.ts`:
```ts
export interface ActivityLogEntry {
    ...
    details?: string | null;  // JSON string
    ...
}
```
**API `/api/activity-log` trả về trường `details` là một chuỗi văn bản (JSON String)** dạng `"{\"c\": 0, \"u\": 1, ...}"`.

### Bước 2: Type Checking ngớ ngẩn tại `ReconPipelineGrid.tsx`
Tại dòng 414 và 474:
```tsx
// Dòng 414 (Kafka Consumer loop):
const details = (typeof k.details === 'object' && k.details !== null ? k.details : {}) as Record<string, any>;

// Dòng 474 (Transmute loop):
const details = (typeof t.details === 'object' && t.details !== null ? t.details : {}) as Record<string, any>;
```
Trong JavaScript/TypeScript:
`typeof "{\"c\": 0}" === "string"`.
Do đó, điều kiện `typeof k.details === 'object'` **LUÔN LUÔN TRẢ VỀ FALSE**!

### Bước 3: Hậu quả dây chuyền (Domino Effect)
1. Biến `details` của $100\%$ các dòng log luôn bị fallback thành `{}` (Empty Object).
2. Khi `details = {}`:
   - `details.c` là `undefined`
   - `details.u` là `undefined`
   - `details.d` là `undefined`
   - `details.inserted` là `undefined`
   - `details.updated` là `undefined`
3. Điều kiện `hasExplicitOps`:
   ```tsx
   const hasExplicitOps = details.c != null || details.u != null || details.d != null;
   ```
   Trở thành: `undefined != null` -> **`hasExplicitOps = FALSE`**!
4. Frontend rơi vào nhánh Fallback:
   ```tsx
   if (hasExplicitOps) {
       ...
   } else {
       row.kafka.c += written;  // written = 1 => kafka.c += 1!
       row.shadow.c += written; // written = 1 => shadow.c += 1!
   }
   ```
   **Toàn bộ số lượng bản ghi (kể cả UPDATE hay DELETE) đều bị ép cộng mù vào Create/Insert (`c`)!**
   - Bản ghi 1 (04:06): `written = 1` -> `c += 1`
   - Bản ghi 2 (04:06): `written = 1` -> `c += 1`
   - Bản ghi 3 (08:16 - UPDATE): `written = 1` -> `c += 1`
   -> **Tổng cộng: `kafka.c = 3`, `shadow.c = 3`, `u = 0`, `d = 0`!**
5. Tại Transmute:
   ```tsx
   const ins = Number(details.inserted) || 0; // = 0
   const upd = Number(details.updated) || 0;  // = 0
   const skip = Number(details.skipped) || 0; // = 0
   const totalWritten = ins + upd;            // = 0
   row.master.total += totalWritten > 0 ? totalWritten : (Number(t.rows_affected) || 0); // += 1
   row.master.inserted += ins; // += 0
   row.master.updated += upd;  // += 0
   row.master.skipped += skip; // += 0
   ```
   -> **`master.total = 3`, nhưng `inserted = 0, updated = 0, skipped = 0`!**

---

## 4. Giải Pháp Khắc Phục Triệt Để

### Giải pháp 1: Hàm Helper `parseLogDetails` đa năng
Bổ sung hàm parse an toàn tiếp nhận cả dạng `string` (JSON String từ API wire) lẫn dạng `object` (nếu có parser middleware):
```ts
const parseLogDetails = (raw: any): Record<string, any> => {
  if (!raw) return {};
  if (typeof raw === 'object' && raw !== null) return raw;
  if (typeof raw === 'string') {
    try {
      const parsed = JSON.parse(raw);
      return typeof parsed === 'object' && parsed !== null ? parsed : {};
    } catch {
      return {};
    }
  }
  return {};
};
```

### Giải pháp 2: Chuẩn hóa tính toán số liệu tại `ReconPipelineGrid.tsx`
1. Thay thế dòng 414 và dòng 474 bằng `parseLogDetails(k.details)` và `parseLogDetails(t.details)`.
2. Khi `hasExplicitOps = true`:
   - `row.kafka.total += (c + u + del)`
   - `row.kafka.c += c`
   - `row.kafka.u += u`
   - `row.kafka.d += del`
   - `row.shadow.total += (c + u + del)` (hoặc `written`)
   - `row.shadow.c += c`
   - `row.shadow.u += u`
   - `row.shadow.d += del`
3. Khi `hasExplicitOps = false` (bản ghi cũ trong quá khứ):
   - `row.kafka.total += written`
   - `row.kafka.c += written`
   - `row.shadow.total += written`
   - `row.shadow.c += written`
4. Master Transmute:
   - `const hasExplicitMaster = details.inserted != null || details.updated != null || details.skipped != null;`
   - Nếu có explicit:
     - `row.master.total += (ins + upd)`
     - `row.master.inserted += ins`
     - `row.master.updated += upd`
     - `row.master.skipped += skip`
   - Nếu không có explicit:
     - `const rows = Number(t.rows_affected) || 0;`
     - `row.master.total += rows;`
     - `row.master.inserted += rows;`

---

## 5. Kết Quả Dự Kiến Sau Khi Fix
Với ngày `2026-10-07` của pipeline `export_jobs`:
- **Kafka CDC:** Total = 3 | c = 2 | u = 1 | d = 0
- **Shadow Ingest:** Total = 3 | c = 2 | u = 1 | d = 0
- **Master Transmute:** Total = 3 | Inserted = 2 | Updated = 1 | Skipped = 0
- **Recon / DLQ:** 0 | 0 | 0 | 0
- **Trạng thái:** `BALANCED` (Xanh đồng bộ 100%)

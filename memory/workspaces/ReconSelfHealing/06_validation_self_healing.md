# Kế hoạch Kiểm thử & Xác minh - ReconSelfHealing

## 1. Unit Test Case (`TestTransmuter_OrphanMasterSoftDelete`)
- **Mục tiêu**: Đảm bảo orphan records bị soft-delete, active records được giữ nguyên, và timestamp `_source_ts` được cập nhật chống race condition.
- **Dữ liệu Test**:
  - `master_test`: ID1 (active), ID2 (active), ID3 (active)
  - `shadow_test`: ID1 (active), ID2 (logical deleted), ID3 (missing)
- **Kết quả mong đợi**:
  - Master ID1: Không đổi (`_deleted = false`)
  - Master ID2: Bị soft-delete (`_deleted = true`), `_source_ts` tăng
  - Master ID3: Bị soft-delete (`_deleted = true`), `_source_ts` tăng

## 2. Kết quả Chạy thực tế (Test Execution Output)
```
=== RUN   TestTransmuter_OrphanMasterSoftDelete
UPDATE "master_test" SET _deleted = true, _source_ts = 1782759421215, _updated_at = datetime('now') WHERE _source_id IN ("id3","id2")
transmuter: soft-deleted orphan master rows successfully {"master": "master_test", "count": 2, "physical_orphans": 1, "marked_deleted": 1}
--- PASS: TestTransmuter_OrphanMasterSoftDelete (0.03s)
PASS
ok  	centralized-data-service/internal/service/master	0.668s
```
Test pass thành công tuyệt đối!

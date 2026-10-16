# BÁO CÁO PHÂN TÍCH PHÁP Y KỸ THUẬT: HIỆN TƯỢNG TRANSMUTE "SUCCESS 4,683" NHƯNG "INSERTED: 0, UPDATED: 4,683"

- **Thời gian phân tích:** 2026-09-16T17:25:00+07:00
- **Đối tượng khảo sát:** Master Binding `#55`
  - **Bảng Shadow:** `shadow_traitestmongodevct.trans_his`
  - **Bảng Master đích:** `public.cdc_transaction_history`
  - **Kết nối đích (Target Connection):** `conn: postgres-develop`
  - **Log hệ thống ghi nhận:**
    ```json
    {
      "master": "public.cdc_transaction_history",
      "source": "trans_his",
      "scanned": 4683,
      "skipped": 0,
      "updated": 4683,
      "inserted": 0,
      "active_gate": "",
      "duration_ms": 952,
      "rule_misses": 0,
      "type_errors": 111
    }
    ```

---

## 1. NGUYÊN NHÂN CỐT LÕI 1: BẢN CHẤT CỦA CƠ CHẾ POSTGRESQL UPSERT (TẠI SAO INSERTED = 0 VÀ UPDATED = 4,683)

### 1.1. Mã nguồn thực thi tại `centralized-data-service/internal/service/master/transmuter.go` (dòng 1144–1168)
```sql
INSERT INTO %s (%s) VALUES %s
    ON CONFLICT (_gpay_id) DO UPDATE SET %s
    WHERE COALESCE(EXCLUDED._source_ts, 0) >= COALESCE(%s._source_ts, 0) - %d
    RETURNING (xmax = 0) AS inserted
```
```go
var inserted, updated int64
for _, r := range resRows {
    if r.Inserted {
        inserted++
    } else {
        updated++
    }
}
```

### 1.2. Cơ chế vật lý của PostgreSQL Engine đối với trường ẩn `xmax`
Trong PostgreSQL, mỗi hàng dữ liệu vật lý (tuple) đều có các trường hệ thống ẩn: `xmin`, `xmax`, `ctid`.
- Khi một hàng **mới hoàn toàn** được chèn vào bảng qua lệnh `INSERT`: Giá trị `xmax` của tuple đó được khởi tạo bằng `0`.
  -> Biểu thức `RETURNING (xmax = 0) AS inserted` sẽ trả về `true`.
  -> Code Go đọc `r.Inserted == true` và tăng bộ đếm: `inserted++`.
- Khi bản ghi bị **xung đột khóa chính** (trùng `_gpay_id`): PostgreSQL bỏ qua việc chèn mới và kích hoạt mệnh đề `ON CONFLICT DO UPDATE`. Lúc này, PostgreSQL thực hiện cập nhật tuple hiện có và gán giá trị `xmax` bằng Transaction ID (XID) của giao dịch đang thực thi (`xmax > 0`).
  -> Biểu thức `RETURNING (xmax = 0) AS inserted` sẽ trả về `false`.
  -> Code Go đọc `r.Inserted == false` và tăng bộ đếm: `updated++`.

### 1.3. Kết luận về con số `scanned: 4683, updated: 4683, inserted: 0`
- Toàn bộ **4,683 bản ghi này đã tồn tại sẵn trong bảng Master `public.cdc_transaction_history`** (từ lần sync trước hoặc đã được ghi trước đó).
- Khi bấm "Sync ngay" (Immediate Sync), Transmuter quét lại toàn bộ 4,683 bản ghi từ bảng Shadow và gửi câu lệnh UPSERT hàng loạt.
- Do 100% bản ghi đều bị trùng khóa `_gpay_id`, PostgreSQL kích hoạt nhánh `DO UPDATE` cho toàn bộ 4,683 dòng.
- **Do đó, không có dòng nào được chèn mới (`inserted = 0`), mà 100% rơi vào cập nhật (`updated = 4,683`). Hệ thống không bị treo hay drop dữ liệu.**

---

## 2. NGUYÊN NHÂN CỐT LÕI 2: CẢNH BÁO NGUY CƠ "SILENT FALLBACK" VỀ DEFAULT MASTER DB

Trong file `centralized-data-service/internal/service/source/connection_manager.go` (dòng 123–129):
```go
// 3. Fallback to destination pool if still not resolved
if dsn == "" {
    if m.logger != nil {
        m.logger.Warn("master connection key not found in overrides or connection_registry, fallback to destination pool",
            zap.String("key", key))
    }
    return m.reg.GetDB(database.RoleDestination)
}
```

### Rủi ro tiềm ẩn:
- Nếu connection code `postgres-develop` chưa được cấu hình đúng DSN trong `cdc_system.connection_registry` (hoặc `status != 'active'`), CDS Engine **không ném lỗi mà âm thầm fallback trỏ về Default Destination DB (`RoleDestination` - port 5434)**!
- Nếu xảy ra fallback này:
  * Transmuter vẫn chạy thành công và báo `updated: 4,683`.
  * Nhưng dữ liệu thực tế lại nằm trên Database mặc định (port 5434) chứ không nằm trên Database của `postgres-develop`.
  * Người dùng mở Database của `postgres-develop` ra kiểm tra thì thấy bảng trống trơn.
- **Khuyến nghị kiến trúc:** Khi `MasterConnectionKey` được chỉ định tường minh (`postgres-develop`), nếu không tìm thấy DSN hoặc lỗi kết nối, hệ thống **BẮT BUỘC PHẢI FAIL-FAST (BÁO LỖI NGAY)**, tuyệt đối CẤM fallback âm thầm về `RoleDestination`.

---

## 3. PHÂN TÍCH CHỈ SỐ `type_errors: 111` TRONG METADATA

- Trong log của phiên chạy có chỉ số: `"type_errors": 111`.
- Chỉ số này sinh ra từ tầng Transform Strategy (`BuildEmits` / `toTypedValue`):
  * Có **111 bản ghi** gặp trường dữ liệu không thể ép kiểu chính xác theo `mapping_rule_master` (ví dụ: chuỗi không parse được sang timestamp, số nguyên bị tràn kiểu, hoặc định dạng JSON sai).
  * Các bản ghi này vẫn được ghi xuống Master (nhưng trường bị lỗi có thể mang giá trị null hoặc fallback).
  * Cần rà soát lại định nghĩa kiểu dữ liệu trong `mapping_rule_master` của Binding `#55` đối chiếu với dữ liệu thực tế từ collection `trans_his` để sửa dứt điểm 111 lỗi ép kiểu này.

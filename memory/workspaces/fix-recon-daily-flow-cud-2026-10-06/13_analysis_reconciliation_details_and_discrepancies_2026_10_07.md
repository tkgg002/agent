# Phân Tích Kỹ Thuật: Details Log, Lịch Sử 2026-10-06 & Độ Lệch payment_bills 186 vs 168

**Thời điểm:** 2026-10-07 16:50:00 +07:00  
**Tác giả:** Brain (Architect)

---

## 1. Yêu cầu 1: Bổ sung "Details Log" vào 3 Tab Đối Soát

Hiện tại ở Modal Drilldown của Pipeline, 3 Tabs chỉ hiển thị một số trường cơ bản:
- Tab **Log Kafka**: Chỉ có Thời gian, Trạng thái, Thời lượng, Rows, Trace ID, Kích hoạt, Lỗi. Chưa có cột hiển thị chi tiết các phép toán `c, u, d`, `batch_size`, `buffer_type` hay xem raw JSON `details`.
- Tab **Log Transmute**: Chỉ có Thời gian, Trạng thái, Thời lượng, Rows, Trace ID. Chưa có cột hiển thị `inserted`, `updated`, `skipped`, `scanned` hay xem raw JSON `details`.
- Tab **Nhật ký đối soát**: Chỉ có chuỗi tóm tắt `src -> dest`, chưa xem được chi tiết `diff`, `missing_ids`, `stale_ids`, `field_diffs`, v.v.

**Giải pháp:** Thêm cột "Chi tiết log (Details)" trực quan kèm nút xem raw JSON/Popover cho cả 3 Tab.

---

## 2. Yêu cầu 2: Tại sao ngày 2026-10-06 anh có Update nhưng Kafka/Shadow vẫn báo `5 | 5 | 0 | 0`?

Dữ liệu thực tế truy vấn từ `cdc_system.cdc_activity_log` cho `export_jobs` ngày 2026-10-06:
- Có 5 lần chạy `kafka-consumer` và 5 lần `transmute`:
  1. ID 105891: `kafka-consumer` | `written: 1` -> Transmute ID 105894: `inserted: 1, updated: 0` (Insert)
  2. ID 105895: `kafka-consumer` | `written: 1` -> Transmute ID 105897: `inserted: 0, updated: 1` (**Update**)
  3. ID 105903: `kafka-consumer` | `written: 1` -> Transmute ID 105904: `inserted: 1, updated: 0` (Insert)
  4. ID 106338: `kafka-consumer` | `written: 1` -> Transmute ID 106340: `inserted: 1, updated: 0` (Insert)
  5. ID 106607: `kafka-consumer` | `written: 1` -> Transmute ID 106609: `inserted: 0, updated: 1` (**Update**)

**Nguyên nhân gốc rễ:**
- Ở Master Transmute: Ghi nhận chính xác `Inserted: 3, Updated: 2` (tổng 5).
- Nhưng ở Kafka & Shadow: Vào ngày hôm qua (2026-10-06), Backend Go `centralized-data-service` đang chạy phiên bản cũ chưa có tính năng đếm `c, u, d` trong `batch_buffer.go` (tính năng này mới được triển khai vào 11:15 sáng nay 2026-10-07).
- Do đó, trong DB ngày hôm qua, 5 bản ghi `kafka-consumer` chỉ có `details = {"written": 1, "trace_id": "...", "batch_size": ...}` và KHÔNG HỀ CÓ TRƯỜNG `c, u, d`.
- Vì thiếu trường `c, u, d`, Frontend fallback gán toàn bộ `written` vào `c` $\rightarrow$ dẫn tới hiển thị `c: 5, u: 0, d: 0` trên UI.

---

## 3. Yêu cầu 3: Độ lệch của `payment_bills` (`186 163 23 0 | 168 163 23 0 | 168 139 29 0`)

Dữ liệu thực tế ngày 2026-10-07 của `payment_bills`:
- **Kafka CDC:** `Total: 186 | c: 163 | u: 23 | d: 0` (163 + 23 = 186)
- **Shadow Ingest:** `Total: 168 | c: 163 | u: 23 | d: 0` (163 + 23 = 186, nhưng Total = 168)
- **Master Transmute:** `Total: 168 | Ins: 139 | Upd: 29 | Skip: 0` (139 + 29 = 168)

### Bản chất kỹ thuật:
1. **Tại sao Kafka = 186 nhưng Shadow = 168 và Master = 168?**
   - Kafka là Message Streaming: nhận 186 message events. Trong đó có 18 events là các update lặp lại trên cùng Primary Key.
   - Khi ghi vào DB Shadow (PostgreSQL) qua cơ chế `UPSERT ON CONFLICT (pk) DO UPDATE`, các updates trùng PK được gộp vào dòng hiện có $\rightarrow$ Shadow chỉ ghi thành công **168 dòng vật lý** (`written = 168`).
   - Master Transmute quét các dòng thay đổi từ Shadow và ghi vào Master đúng **168 dòng vật lý** (`139 Insert + 29 Update = 168`).
   - **Kết luận:** Số dòng thực tế được ghi vào Shadow (168) và Master (168) là **KHỚP NHAU 100%**!

2. **Tại sao Shadow lại bị lệch nội bộ: Total 168 nhưng c: 163, u: 23 (tổng 186)?**
   - Do code Frontend: Total của Shadow thì lấy `written` (168 dòng vật lý), nhưng detail `c` và `u` lại lấy thẳng từ Kafka events (`row.shadow.c += c` = 163, `row.shadow.u += u` = 23).
   - Dẫn đến Total là 168 nhưng các số con bên trong cộng lại thành 186.
   - Cần chuẩn hóa phân bổ `shadow.c` và `shadow.u` theo đúng số dòng vật lý `written`.

3. **Tại sao Master có 139 Insert và 29 Update, khác với Shadow (163 c, 23 u)?**
   - Trong 76 batches buổi sáng (146 rows), do chưa có metadata `c, u, d` nên Frontend fallback coi cả 146 rows là Create. Nhưng thực tế trong 146 rows đó có 24 rows là Update đã tồn tại ở Master.
   - Master Transmute đối chiếu trực tiếp giữa Shadow và Master: nếu bản ghi đã có ở Master $\rightarrow$ thực hiện UPDATE (29 rows), chưa có $\rightarrow$ INSERT (139 rows).
   - Số liệu ở Master phản ánh chính xác hành vi DML trên DB Master.

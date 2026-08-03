# 09 — Hồ Sơ Giải Pháp Kỹ Thuật Chuẩn: Chunk-Based Stream-to-Bucket

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Cập nhật:** 2026-07-21  
> **Tác giả:** System Architect & Chief Engineer  
> **Trạng thái:** FINAL APPROVED ARCHITECTURE SPECIFICATION  

---

## 1. Bức Tranh Tổng Thể Quy Trình 3 Tầng (Final Enterprise Flow)

```
 [ TẦNG 1: OUTER LOOP — QUẢN LÝ JOB & CHECKPOINT ]
  - Scope: Chia 30 ngày thành 30 Jobs độc lập (Mỗi Job = 1 Ngày = 96 sub-windows 15m).
  - State Persistence: Lưu Checkpoint (checkpoint_ts) vào cdc_system.recon_jobs sau mỗi ngày.
  - Resiliency: Nếu crash/restart, hệ thống tự động resume từ ngày kế tiếp.

 [ TẦNG 2: INNER LOOP — STREAMING & GO BUCKETS RAM ]
  - Query: SELECT id, last_updated_at FROM payment_bills WHERE last_updated_at >= start_day AND last_updated_at < end_day ORDER BY last_updated_at ASC
  - Strict Bounds: Sử dụng nửa khoảng [start, end) triệt tiêu 100% dùng BETWEEN.
  - Normalization: timestamp = t.UnixMilli() triệt tiêu sai số Millis (Mongo) vs Micros (Postgres).
  - Memory Footprint: O(1) Constant RAM (Mảng tĩnh Buckets[96] uint64 ~ 768 bytes).

 [ TẦNG 3: IN-MEMORY COMPARISON & TARGETED DRILL-DOWN ]
  - RAM Match: So sánh 96 Buckets Mongo vs Postgres trên RAM Go (< 0.001ms).
  - Targeted Query: Khay nào lệch -> Bắn 1 query lấy list ID của ĐÚNG 15 phút đó để báo cáo.
```

---

## 2. Quy Tắc Kỹ Thuật Bắt Buộc (Mandatory Engineering Standards)

### A. Tiêu Chuẩn Trị Dứt Điểm Boundary Skew (Phân Giải Thời Gian)
1. **Ép Kiểu Unix Milliseconds Trên Go:**
   Mọi phép tính Hash và phân bổ chỉ số Bucket (`windowIndex`) phải ép kiểu cắt bỏ microsecond:
   ```go
   tsMilli := t.UnixMilli()
   windowIndex := (tsMilli - startDayMilli) / (15 * 60 * 1000)
   ```
2. **Cấm Sử Dụng Toán Tử `BETWEEN`:**
   Mọi truy vấn SQL và Mongo Aggregation **BẮT BUỘC** dùng nửa khoảng mở $[start, end)$:
   ```sql
   WHERE last_updated_at >= $1 AND last_updated_at < $2
   ```

### B. Nguyên Lý Kháng Thể Flash Sale (KISS Principle & O(1) Space)
1. **Kháng thể RAM $O(1)$:** Mảng `Buckets[96]` giữ nguyên kích thước $768\text{ bytes}$ bất kể dải dữ liệu ngày Flash Sale là 1 triệu hay 50 triệu bản ghi.
2. **Khái niệm Chunk Cố Định 1-Ngày:** Giữ nguyên Chunk 1-ngày cố định để đơn giản hóa hệ thống (KISS). Chỉ kích hoạt Dynamic Sub-chunking (6h) nếu volume ngày vượt ngưỡng $100\text{ triệu}$ records làm rớt kết nối TCP.

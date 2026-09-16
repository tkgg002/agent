# 01_requirements_shadow_trigger_master_index.md
# TÀI LIỆU YÊU CẦU KỸ THUẬT: PHỤC HỒI SONYFLAKE TRIGGER SHADOW & MASTER INDEX RECOMMENDATION SUITE

---

## I. MỤC TIÊU
1. **Phần 1 (Shadow Sonyflake Trigger Resilience):**
   - Đảm bảo 100% các bảng Shadow khi được tạo từ bất kỳ nguồn nào (nút `Create` trên UI, lệnh NATS `create-default-columns`, hoặc auto-create khi CDC event tới) BẮT BUỘC có Trigger `trg_<table_name>_sonyflake_fallback`.
   - Ngăn chặn triệt để tình trạng `_gpay_id` bị NULL khi chạy Snapshot hoặc Debezium realtime.

2. **Phần 2 (Master Index & Recommendation Multi-Connection):**
   - Khôi phục tính năng Đề xuất Index (Index Recommendations) cho Master Table trên CMS Web UI (`TableIndexManager.tsx`).
   - Xóa bỏ hoàn toàn hardcode kết nối `"default"` trong `IndexHandler` (CDS Worker Engine) để Master Table nằm trên kết nối riêng (như `postgres-develop`) hiển thị và tạo index đúng DB đích.
   - Bảo đảm khi Master Table được Approved, các index mặc định và index đề xuất được tự động khởi tạo trên đúng Database đích.

---

## II. PHẠM VI & TIÊU CHÍ HOÀN THÀNH (DEFINITION OF DONE)
- [ ] G1: Bảng Shadow được tạo qua nút `Create` trên UI tự động có sequence, 2 functions, và 1 trigger sonyflake.
- [ ] G2: Lệnh `CreateEmptyTable` / `EnsureCDCColumnsInSchema` không bao giờ để lại bảng thiếu trigger.
- [ ] G3: `TableIndexManager.tsx` hiển thị đầy đủ đề xuất index cho cả `shadow` và `master`.
- [ ] G4: `IndexHandler` trong Worker kết nối đúng target database của Master Table theo `connection_key`.
- [ ] G5: Tất cả code mới tuân thủ Simplicity First, Minimal Impact, không can thiệp trực tiếp vào DB mà không thông qua code chuẩn.

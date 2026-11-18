---
tier: T2
mode: standard
suffix: mapping_and_trigger_cms
workspace: master-mapping-bulk-and-trigger-cms-2026-11-16
created_at: 2026-11-16T13:47:15+07:00
---
# 01 Requirements — Giải Pháp Cấu Hình Nhanh Master Mapping (Task 1) & Đánh Giá / Thiết Kế Quản Trị Trigger `Master CDC -> Master Business` Trên CMS (Task 2)

## 1. Yêu cầu của User (2 Tasks)

### Task 1: Giải pháp cấu hình hàng loạt trên trang Master Mapping (`/masters/:name/mappings?binding_id=95`)
- **Bối cảnh thực tế:** Trên màn hình `/masters/cdc_payment/mappings?binding_id=95` (hiện có ~101 Mapping Rules cho bảng `cdc_payment_bill`), User cần ánh xạ sang đặc tả bảng `payment_bill` (48 cột: 34 cột scalar đổi sang `snake_case` + kiểu chuẩn `NUMERIC`/`TIMESTAMPTZ`/`TEXT`, 10 cột bóc từ nested JSON như `merchantInfo.id -> merchant_info_id`, `transactionParties.payer.merchantCode -> payer_merchant_code`, `embedData.collectResend.nextAt -> collect_resend_next_at`, cộng thêm các cột JSONB payload dùng chung cho trigger `payment_bill_payload` như `merchant_info`, `payments`, `payment_ids`, `payment_method`, `item`, `merchant_customer`, `embedData`/`embed_data`, `extra_info`, `routing_info`, `transaction_parties`, `device_info`, và loại bỏ các cột không dùng như `__v`, `bizDomain`, `bizType`, `currencies`, `isActive`, `sourceType`, `tokenCreate`...).
- **Nỗi đau (Pain Point):** Hiện tại User đang phải đi sửa tên cột (`camelCase -> snake_case`), chọn kiểu dữ liệu (`INTEGER -> NUMERIC`, `TEXT -> TIMESTAMPTZ`), thêm từng cột nested, bật/tắt/approve/reject **từng dòng một** trên UI.
- **Câu hỏi:** Hệ thống hiện tại đã có công cụ nào chưa, đang thiếu gì (`file:line`), và **giải pháp tối ưu nhất** để User không phải ngồi làm thủ công từng cái một là gì?

### Task 2: Đánh giá ("Có nên không?") & Giải pháp quản trị Trigger/Function đồng bộ `Master CDC -> Master Business` trên CMS
- **Bối cảnh thực tế:** Sau khi CDC đồng bộ dữ liệu từ Shadow sang bảng Master dạng `cdc_<table_name>` (`public.cdc_payment_bill`), hiện tại User phải tự vào PostgreSQL DB đích chạy tay các script SQL tạo PL/pgSQL Function & Trigger:
  1. `fn_sync_bill_derived_fields()` / `trg_sync_cdc_to_payment_bill()` (đồng bộ từ `public.cdc_payment_bill` sang `public.payment_bill`, tính 4 cột dẫn xuất `payment_channel_ids`, `is_atomic`, `is_atomic_collection`, `has_unapplied_lock_unavailable`, xử lý soft-delete/DELETE và OCC `last_updated_at`).
  2. `trg_sync_cdc_to_payment_bill_payload()` (đồng bộ từ `public.cdc_payment_bill` sang `public.payment_bill_payload`).
- **Câu hỏi:**
  1. **Có nên đưa việc quản trị / chạy Trigger `cdc_<table_name> -> business_table` lên CMS không?** Phân tích ưu/nhược điểm, rủi ro vận hành/bảo mật/hiệu năng, và cả các lỗi tiềm ẩn ngay trong đoạn SQL Trigger User đang viết tay (đối chiếu với Mapping Rules ở Task 1!).
  2. **Giải pháp kiến trúc tối ưu nhất trên `data-hub` CMS** nếu hỗ trợ (hoặc phạm vi hỗ trợ an toàn nhất) là gì?

## 2. Ràng buộc Thực thi (Technical Constraints từ `GEMINI.md`, `INDEX.md` & `system_overview_and_design_pattern.md`)
- **L003 (Verify Sync-Layer JSONB Array Representation Over Downstream Business Array Types):** Bắt buộc tách bạch hoàn toàn lược đồ bảng Master CDC (`cdc_payment_bill`, dùng `JSONB` cho trường mảng) với lược đồ bảng nghiệp vụ hạ lưu (`payment_bill` / `payment_bill_payload`, dùng `TEXT[]` cho `payment_channel_ids`, `refunded_operation_ids` ở sau bước Trigger). Không nhầm lẫn hai tầng này.
- **L004 (Rule-Flag Driven Metadata Column Detection Over Hardcoded Column Names):** Tuyệt đối cấm gán cứng tên bảng/tên cột nghiệp vụ (`payment_bill`, `payment_bill_payload`, `embedData`, `ATOMIC`, `VIET_QR`...) vào lõi động cơ `centralized-data-service` hay `cdc-cms-service`. Mọi tính năng mới phải 100% tổng quát (agnostic).
- **L080 (Pure CDC Unified Multi-Shadow Binding & SRP Split):** Bảo toàn nguyên tắc Single-Responsibility (SRP) — tách file/component riêng đúng layer (`cdc-cms-web/src/components/master/`, `cdc-cms-service/internal/{api,app,domain,infra}/master/`, `centralized-data-service/internal/{handler,service}/master/`), tuyệt đối không nhồi hàng trăm dòng vào các God-files (`MasterMappingFieldsPage.tsx`, `master_ddl_handler.go`, `transmuter.go`).
- **L086 (Spec-Driven PK & Timestamp Field Resolution):** Mọi cấu hình PK, cột thời gian và mapping phải tường minh qua spec/schema, không đoán mò ngầm gây lệch dữ liệu.
- **S2.5 Pre-Plan Critique (DG0):** Bắt buộc đối chiếu chéo chính đoạn SQL Trigger mà User gửi trong prompt với bảng Mapping Rules của `binding_id=95` và đặc tả `payment_bill` / `payment_bill_payload` — nếu có lệch tên cột, thiếu cột, hoặc bug logic trong SQL của User thì phải chỉ thẳng bằng chứng vật lý ("la làng lên") ngay đầu báo cáo!
- **Open-Closed & Propose-Only (Rule #1, #13 & `system_overview_and_design_pattern.md` Mục 0, 2, 5):** Giữ 100% Read-Only đối với source code, thiết kế **ĐÚNG 1 PHƯƠNG ÁN TỐI ƯU NHẤT** cho mỗi phát hiện kèm Code Demo thực tế, chạy `DG0` + `DG1/DG2` (`PASS`), trình User và dừng chờ `APPROVE`.

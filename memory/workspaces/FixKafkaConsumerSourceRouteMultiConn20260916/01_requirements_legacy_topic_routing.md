# 01_requirements_legacy_topic_routing.md

## Yêu cầu Nghiệp vụ & Kỹ thuật

### 1. Vấn đề
- Hệ thống có 2 kết nối MongoDB chứa cùng cơ sở dữ liệu `core-trans-proxy-history-service` và collection `trans-his`:
  1. `traitestctphs`: Topic cũ `cdc.goopaylocal.core-trans-proxy-history-service.trans-his`
  2. `traitestmongodevct`: Topic mới `cdc.goopaylocal.traitestmongodevct.core-trans-proxy-history-service.trans-his`
- Do topic cũ mang định dạng 4 segments, tầng router không nhận diện được tên kết nối dẫn đến fan-out sang cả 2 shadow tables và transmute nhầm sang bảng master `cdc_transaction_history`.

### 2. Yêu cầu Bắt buộc
1. **Truy nguyên Connection Nguồn:**
   - Mỗi Kafka topic đi vào worker BẮT BUỘC phải được ánh xạ chính xác về Connection Code sở hữu nó.
   - Nguồn chân lý: Bảng `cdc_system.sources` chứa `(connector_name, topic_prefix)`.
2. **Không can thiệp cấu hình cứng:**
   - TUYỆT ĐỐI KHÔNG hardcode danh sách mapping vào file `config-local.yml`.
   - TUYỆT ĐỐI KHÔNG can thiệp chỉnh sửa DB thủ công (no cheat DB).
3. **Phân lập Dữ liệu Tuyệt đối:**
   - Record từ `cdc.goopaylocal...` CHỈ được ghi vào `shadow_traitestctphs.trans_his` -> Master `master_core_trans_proxy_history_service.trans_his`.
   - Record từ `cdc.goopaylocal.traitestmongodevct...` CHỈ được ghi vào `shadow_traitestmongodevct.trans_his` -> Master `public.cdc_transaction_history`.
4. **Tương thích ngược 100%:**
   - Mọi topic cổ điển khác của hệ thống khi không có sự trùng lặp tên bảng hoặc chưa có trong bảng `sources` vẫn tiếp tục vận hành bình thường qua cơ chế fallback.

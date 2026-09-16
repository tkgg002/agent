# Requirements

1. Transmute trigger không được silent drop — mỗi drop phải có log + metric
2. HandleTransmuteShadow: lookup fail phải log error + emit metric
3. NATS publish error không được ignore (`_ =`)
4. Debouncer backpressure phải có observability (metric + log)
5. Scheduler stuck schedule phải tự cleanup (không chỉ khi restart)
6. Smoke recon CountRows không được full table scan trên bảng lớn
7. Index thiếu cho timestamp_field trên shadow + master table

## Non-scope
- Migrate NATS Core → JetStream (scope quá lớn, để phase sau)
- Thay đổi kiến trúc debouncer
- Sửa CMS frontend Log Transmute (vấn đề riêng)

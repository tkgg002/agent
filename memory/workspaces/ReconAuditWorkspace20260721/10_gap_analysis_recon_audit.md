# 10 — Phân Tích Lỗ Hổng Kiến Trúc & Rủi Ro (Gap Analysis)

> **Workspace:** `ReconAuditWorkspace20260721`  

---

## I. ĐÁNH GIÁ CÁC ĐIỂM NGUY CƠ ĐÃ ĐƯỢC KÍCH HOẠT VÀ XỬ LÝ

1. **Rủi ro Cửa sổ Ngày Lẻ (Non-Midnight Lookback Bounds):**
   - *Phát hiện:* Lookback window ngắn (như 2 tiếng) khi gọi vào ngày lẻ nếu chia 96 buckets không cắt biên sẽ gây lãng phí query DB qua mốc `dayEnd`.
   - *Khắc phục:* Đã khóa cứng biên `subEnd = min(subStart+15m, dayEnd)` và `break` khi `subStart >= dayEnd`.

2. **Rủi ro Mất Trace Context Qua NATS:**
   - *Phát hiện:* NATS publish message nếu không inject OTel carrier sẽ làm đứt gãy trace giữa `CheckHandler` và `ReconJobWorker`.
   - *Khắc phục:* Đã xác nhận `otel.GetTextMapPropagator().Inject/Extract` được gọi đầy đủ trong header NATS message.

3. **Rủi ro Trôi Mốc Thời Gian (Time Drift):**
   - *Phát hiện:* Thời gian client không đồng bộ gây lệch dữ liệu đối soát.
   - *Khắc phục:* Đã chốt cứng `Fixed Watermark Freeze` bằng cách làm tròn phút `:00` trừ đi 120s buffer lag trong `resolveTimeRange`.

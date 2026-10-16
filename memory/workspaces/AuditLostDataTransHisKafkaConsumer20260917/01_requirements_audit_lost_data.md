# 01 Requirements: Audit Mất Dữ Liệu Source (trans-his) -> Shadow & Tê Liệt Kafka Consumer

## 1. Bối cảnh & Hiện tượng
- **Hiện tượng 1**: Recon phát hiện drift trong khoảng `[10:07:00 -> 10:22:00]`:
  - Source MongoDB: `core-trans-proxy-history-service.trans-his` có dữ liệu phát sinh.
  - PostgreSQL Shadow: `shadow_traitestctphs.trans_his` bị thiếu dữ liệu (drift count = 212 records).
  - Không có cảnh báo/thông báo nào được gửi đi và hệ thống không tự động chạy lại (Auto-Heal) bù dữ liệu.
- **Hiện tượng 2**: Từ hôm qua tới nay không có bất cứ một `kafka-consume` nào chạy (không có logs / OTel spans `kafka.consume`).

## 2. Mục tiêu (Definition of Done)
1. **Root Cause 1**: Làm rõ 100% bằng chứng code & metrics vì sao mất dữ liệu từ Source MongoDB sang PostgreSQL Shadow `shadow_traitestctphs.trans_his`.
2. **Root Cause 2**: Làm rõ vì sao từ hôm qua tới nay không có bất kỳ `kafka-consume` nào chạy.
3. **Root Cause 3**: Làm rõ vì sao không có Auto-Heal và không có thông báo (Alerting).
4. **Giải pháp Kỹ thuật**:
   - Phương án sửa lỗi code Over-filtering tại `IsGenericConnectionCode` / `kafka_consumer.go` / `event_handler.go`.
   - Phương án khôi phục (Heal) 212 bản ghi bị mất mà không cần can thiệp DB trực tiếp (tuân thủ Rule #12).
   - Thiết kế nâng cấp Auto-Heal & Notification.

# 01_requirements — Audit miss record Source → Shadow khi tăng Kafka consumer (production)

## Yêu cầu của User (2026-10-03)
"audit hệ thống xem vì sao chạy trên production, số lượng kafka consumer nhiều, thì bắt đầu xuất hiện miss record từ source -> shadow"

## Definition of Done
- R1. Xác định luồng Kafka → Shadow thực tế bằng code (file:dòng).
- R2. Chỉ ra cơ chế làm mất record và vì sao nó tăng theo số consumer, phân biệt rõ **đã chứng minh** vs **giả thuyết**.
- R3. Loại trừ giả thuyết sai bằng chứng cứ.
- R4. Đề xuất giải pháp đúng 1 hướng (Brain: Propose Only, chờ APPROVE trước khi Muscle sửa code).
- R5. Cung cấp checklist xác minh trên production (read-only).

## Ngoài phạm vi
- Không sửa source code, không đụng DB/Kafka production.
- Hop Shadow → Master (Transmuter) không thuộc audit này.

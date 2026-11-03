---
tier: T2
suffix: daily_flow_broken_oct27_20261030
workspace: audit-daily-flow-broken-oct27-2026-10-30
mode: standard
created_at: 2026-10-30T13:12:35+07:00
---

# 01. REQUIREMENTS — Audit & Khắc phục Lỗi Bảng Thống Kê Đối Soát 3 Trạm & DLQ Theo Ngày Gãy Từ 27-28/10 (`daily_flow_broken_oct27_20261030`)

## 1. Bối cảnh & Yêu cầu của User
User cung cấp bảng số liệu **"Thống kê đối soát 3 trạm & DLQ theo ngày — Pipeline"**:
- `2026-10-30`: Kafka `0` | Shadow `0 Ops (0 dòng DB)` | Master `0 Ops (0 dòng DB)` | DLQ `0` | `CÂN BẰNG`
- `2026-10-29`: Kafka `0` | Shadow `0 Ops (0 dòng DB)` | Master `0 Ops (0 dòng DB)` | DLQ `0` | `CÂN BẰNG`
- `2026-10-28`: Kafka `0` | Shadow `0 Ops (0 dòng DB)` | Master `0 Ops (0 dòng DB)` | DLQ `0` | `CÂN BẰNG`
- `2026-10-27`: Kafka `163` (`75 c`, `88 u`, `0 d`) | Shadow `163 Ops (83 dòng DB)` (`75 c`, `88 u`, `0 d`) | Master `163 Ops (163 dòng DB)` (`75 Inserted`, `88 Updated`, `0 Delete`, `0 Skipped`) | DLQ `0`, Lệch thiếu `2`, Lệch stale `0`, Đã heal `0` | `CÓ LỆCH`
- `2026-10-26`: Kafka `528` (`242 c`, `286 u`, `0 d`) | Shadow `528 Ops (303 dòng DB)` (`242 c`, `286 u`, `0 d`) | Master `491 Ops (491 dòng DB)` (`245 Inserted`, `246 Updated`, `0 Delete`, `0 Skipped`) | DLQ `350`, Lệch thiếu `5`, Lệch stale `0`, Đã heal `5` | `CÓ LỆCH`

Câu hỏi của User:
> "sao cái Thống kê đối soát 3 trạm & DLQ theo ngày — Pipeline của tao lại gãy rồi từ ngày 27,28 check log diff cho tao. đợt đó có fix vụ time 24h đêm, mày lại phá gì rồi"

## 2. Ràng buộc Thực thi (Technical Constraints từ GEMINI.md & Lessons)
- **L031 & L032 (Zero DB CLI / No Cheat DB):** Tuyệt đối cấm gọi `docker exec ... psql` hoặc kết nối CLI vào DB; chỉ kiểm tra qua `git log`, `git diff`, `read_file` và log ứng dụng nếu có.
- **L005 & G2 (Propose-Only Discipline):** Ở pha điều tra và trình bày giải pháp (S2 -> S3 -> DG1), tuyệt đối không sửa source code trước khi User `APPROVE`.
- **L022 (Direct-Answer Summary First):** Mở đầu câu trả lời bằng khối Direct Summary 2–4 câu trả lời trực diện nguyên nhân gãy từ ngày 27, 28 dựa trên `git log` / `git diff` và `file:line`.
- **L041 / L042 / L045 / L046:** Kiểm tra toàn trình pipeline (`Kafka Consumer -> Batch Buffer Shadow -> NATS JetStream / Debouncer -> Transmute Master -> Activity Log / GetDailyFlow`) và đối chiếu chính xác các bản sửa từ đợt fix "time 24h đêm" (ngày 27/10) tới ngày 28–30/10.

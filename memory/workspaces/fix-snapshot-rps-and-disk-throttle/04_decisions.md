# 04_decisions.md — Nhật ký quyết định kiến trúc (ADRs)

## ADR-01: Lựa chọn cơ chế Continuous Throttling (Rate Limiting) thay vì Manual Pause/Resume
- **Bối cảnh**: Disk I/O của PostgreSQL bị chạm trần 95-100% khi chạy snapshot tốc độ cao dẫn đến restart máy chủ DB.
- **Phương án cân nhắc**:
  1. *Option A (Manual Chunking)*: Cứ nạp 1M bản ghi thì tự pause, bắt Operator vào web bấm Resume.
  2. *Option B (Continuous Throttling via RPS)*: Dùng `SnapshotMaxRPS` để worker tự động `time.Sleep` điều tiết tốc độ đều đặn sau mỗi batch, chạy một mạch từ đầu đến cuối mà không cần can thiệp thủ công.
- **Quyết định**: Chọn **Option B**. Việc ép Operator click Resume thủ công là anti-pattern về trải nghiệm vận hành. Continuous Throttling đảm bảo tính tự động 100%, bảo vệ đĩa ổn định 25-35%.

## ADR-02: Quy ước giá trị 0 = Clear về NULL cho SnapshotMaxRPS
- **Bối cảnh**: Người dùng muốn gỡ bỏ giới hạn tốc độ (trở về Unthrottled).
- **Quyết định**: Khi người dùng xóa trắng ô input trên UI, Frontend gửi `snapshot_max_rps: 0`. Backend `UpdateSourceObjectV2Handler` nhận diện giá trị `0` và cập nhật `snapshot_max_rps = NULL` trong DB, đồng bộ với convention của `snapshot_batch_size`.

# 10_gap_analysis.md — Phân tích lỗ hổng kiến trúc

## 1. Lỗ hổng phát hiện (Architectural Gaps)

1. **Thiếu tính đồng bộ toàn diện giữa Hạ tầng và Giao diện Quản trị**:
   - Khi Migration 064 thêm cột `snapshot_max_rps` vào DB và CDS cài đặt logic `time.Sleep`, CMS Service và CMS Web đã bị bỏ quên, không được cập nhật tương ứng.
   - Hậu quả: Operator chỉ có thể can thiệp bằng cách chạy SQL thủ công vào database, không cấu hình được trực tiếp trên giao diện CMS.

2. **Cơ chế Stale Progress Cleanup thiếu phân biệt trạng thái**:
   - CMS `ListSnapshotProgress` quét `updated_at < NOW() - INTERVAL '5 minutes'` và đánh dấu `status = 'error'` cứng.
   - Nếu một batch snapshot chạy lâu (> 5 phút) do DB lag hoặc rate limiting thấp, tiến trình có thể bị đánh dấu nhầm thành lỗi (False Positive Timeout).
   - *Khuyến nghị tương lai*: Thêm heartbeat background độc lập cho `SnapshotRunner` định kỳ ping `updated_at = NOW()`.

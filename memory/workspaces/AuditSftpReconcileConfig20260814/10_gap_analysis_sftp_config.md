# 10_gap_analysis: Phân Tích Lỗ Hổng Kiến Trúc (Dev vs Production Gap Analysis)

## 1. Bảng So Sánh Lỗ Hổng (Gap Comparison Table)

| Hạng mục | Cấu Hình Cũ (Dev Local Sandbox) | Cấu Hình Mới (Production-Ready Spec) | Mức Độ Rủi Ro Cũ |
|---|---|---|---|
| Mật khẩu | Hardcode `sftp_password` trên URI | ${file:/secrets/sftp-credentials.properties:SFTP_PASS} | 🔴 Nguy cơ lộ mật khẩu |
| Tên Miền | `host.docker.internal` | `sftp-prod-internal.goopay.vn` | 🔴 Sập connector trên K8s |
| Tần suất Poll | 3000ms (3 giây) đệ quy `recursive=true` | 60000ms (1 phút) phẳng `recursive=false` | 🔴 DDoS SFTP Server |
| Nhận diện File | `^reconcile_.*\\.csv$` trực tiếp | Atomic Upload: `.tmp` -> `.csv` khi 100% | 🔴 Mất dữ liệu dở dang (Partial Read) |
| Cấu hình Header | Shotgun config (3 thuộc tính đè nhau) | Standard `file_reader.delimited.header=true` | 🟡 Header rơi vào row data |
| Kafka Record Key | Key = Null (Round-Robin partitions) | SMT `ValueToKey` theo `transaction_id` | 🔴 Mất thứ tự dòng CSV |
| Xử Lý Lỗi | Thiếu DLQ (Gặp rác crash Connector) | `errors.tolerance=all` + DLQ Topic | 🔴 Connector FAILED 24/7 |
| Quản lý File Cũ | Tồn đọng vĩnh viễn trong `/reconcile` | Cronjob Archive file +3 ngày, xóa +90 ngày | 🟡 Nghẽn Directory Listing |

## 2. Giải Pháp Lấp Lỗ Hổng (Remediation Strategy)
1. Tháo gỡ 100% rủi ro bảo mật bằng FileConfigProvider.
2. Áp dụng Atomic Upload Standard trên hạ tầng đẩy file của hệ thống nguồn.
3. Deploy Golden Connector JSON Spec lên Kafka Connect Cluster.
4. Cài đặt Cronjob Cleanup Archive trên máy chủ SFTP Server.

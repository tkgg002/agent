# Implementation Plan: Full Clearance - Flow 1 & G-11

Boss đã phê duyệt "chơi hết đi". Kế hoạch này tổng hợp tất cả các hạng mục đang treo để unblock Flow 1 và xử lý dứt điểm các vấn đề kỹ thuật tồn đọng.

## User Review Required

> [!IMPORTANT]
> - Việc **Swap Binary CMS** sẽ làm gián đoạn dịch vụ CMS trong vài giây.
> - Việc **Drop Path A schemas** là hành động không thể đảo ngược (đã xác nhận data an toàn ở iter#7).

## Proposed Changes

### Phase 1: Swap CMS Binary (P0)
**Mục tiêu**: Chuyển sang binary mới hỗ trợ A3 Hybrid (Shadow DB tách biệt) để unblock Flow 1.

- [ ] Dừng service cũ: `kill -TERM 64511`
- [ ] Sao lưu & Ghi đè: `mv /tmp/cdc-cms-service-flow1.new /tmp/cdc-cms-service-flow1`
- [ ] Chạy service mới: `nohup /tmp/cdc-cms-service-flow1 > /tmp/cdc-cms-service-flow1.log 2>&1 &`
- [ ] Kiểm tra Health: `curl http://localhost:8083/health`

### Phase 2: Commit A3 Code (P1)
**Mục tiêu**: Lưu giữ các thay đổi cấu hình đa DB vào hệ thống version control.

- [ ] `git add` các file liên quan trong `cdc-cms-service`.
- [ ] `git commit -m "feat(cms): support hybrid shadow db configuration (A3)"`.

### Phase 3: G-11 Full Fix (P1)
**Mục tiêu**: Xử lý lỗi hyphen trong tên bảng MongoDB để `src 44` có thể tiến tới `active`.

- [ ] Cập nhật plan `02_plan_g11` để bao gồm cả `shadow_binding`.
- [ ] Sửa code `centralized-data-service/internal/service/provisioning_orchestrator.go` để normalize tên bảng (thay `-` bằng `_`).
- [ ] Chạy SQL backfill cập nhật `master_binding` và `shadow_binding` cho các record đã lỗi.

### Phase 4: Path A Cleanup (P2)
**Mục tiêu**: Dọn dẹp 6 schema dư thừa tại Path A (`5433`).

- [ ] Thực thi `DROP SCHEMA ... CASCADE` cho 6 schema đã xác định tại iter#7.

## Verification Plan

### Automated Tests
- `go test ./...` trong cả `cdc-cms-service` và `centralized-data-service`.
- Kiểm tra log worker để xác nhận `src 44` chuyển trạng thái sang `active`.

### Manual Verification
- Truy cập CMS UI, kiểm tra trạng thái của các Source Object.
- Verify data tại Path B (Postgres `5436`) sau khi `src 44` hoạt động.

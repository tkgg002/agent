# 01 Requirements — fix-batch-transform-transmute-trigger

**Ngày tạo:** 2026-08-24  
**Người yêu cầu:** User  
**Loại:** Hotfix / Feature Gap

---

## Vấn đề (Problem Statement)

Khi chạy job `transform` (`cdc.cmd.batch-transform`) trên shadow table (VD: `shadow_payment_service.payments`), worker chỉ thực hiện UPDATE các cột từ `_raw_data` trong shadow mà **không publish `cdc.cmd.transmute-shadow`** sau khi hoàn thành.

Kết quả: Master table **không được auto-update** sau transform. 100M+ records phải chạy lại transmute thủ công, gây lãng phí và sai luồng nghiệp vụ.

---

## Yêu cầu chức năng

1. Sau khi `batch-transform` hoàn thành thành công (status = `COMPLETED`), **tự động publish** NATS message `cdc.cmd.transmute-shadow` để `TransmuteHandler.HandleTransmuteShadow` xử lý → fan-out → materialise master.
2. Trigger này là **fire-and-forget**: không block kết quả transform, không làm thay đổi trạng thái job.
3. **Không trigger** khi job kết thúc với trạng thái `FAILED`, `CANCELLED`, `skipped`, hoặc `error`.
4. Payload phải đúng format đang dùng trong hệ thống (`shadow_table`, `shadow_schema`, `triggered_by: "batch-transform"`), **không có `_source_ids`** (để `HandleTransmuteShadow` → `HandleTransmute` chạy full sync).
5. Không thay đổi constructor `NewBatchTransformHandler` hay signature — dùng `h.NatsConn` từ `BaseHandler` đã được inject sẵn.

---

## Yêu cầu phi chức năng

- Thay đổi tối thiểu: chỉ sửa `batch_transform_handler.go`
- Không phá vỡ test hiện có
- Không thay đổi `server_setup.go` hay bất kỳ file wiring nào
- Phải log khi publish thành công/thất bại để traceable

---

## Definition of Done (DoD)

- [x] `batch_transform_handler.go` publish `cdc.cmd.transmute-shadow` khi COMPLETED
- [x] Build pass (`go build ./...`)
- [x] Existing tests không bị break
- [x] Log xuất hiện: `"post-transform transmute trigger published"` khi thành công

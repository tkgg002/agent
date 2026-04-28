# Tasks — Phase 9 Namespace Finalization

- [x] Audit runtime SQL còn sót ngoài `cdc_system`
- [x] Chuyển sinkworker sang function namespace `cdc_system`
- [x] Chuyển fencing trigger sang `cdc_system`
- [x] Chuyển master RLS helper sang `cdc_system`
- [x] Cập nhật integration tests sang `cdc_system`
- [x] Thêm migration finalization để drop `cdc_internal`
- [x] Cập nhật grant trên `cdc_system`
- [x] Verify bằng search + gofmt + go test

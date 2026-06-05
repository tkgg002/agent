# 09_tasks_solution_phase_p1 — Hồ sơ giải pháp P1

## M-6: DynamicMapper integration
- **Root cause**: Mapper gọi `maskRawData(targetTable, rawData)` — wrapper cứng không có strategy context.
- **Solution**: Inject `eventID + sourceCode` vào MaskTableData để audit log có đủ trace.
- **Lý do xóa helper cũ**: Dead code sau refactor; tránh confusion.

## M-7: BatchBuffer + ReconHeal + KafkaConsumer align
- **Root cause**: 3 path gọi MaskJSONPayload(`"***"`) — sau refactor MaskingService, chỉ cần re-route.
- **Solution**: Tất cả đi qua `MaskTableData` cùng instance MaskingService.
- **Lý do tập trung 1 service**: Single source of truth (theo lesson L-2026-05-23-cross-cutting-concern-single-source).
- **Edge case**: Invalid JSON ở DLQ path → return `null` thay vì `"***"`. Lý do: data đã invalid, không thể recover; `null` consistent với DROP strategy.

## M-8: AuditWriter
- **Root cause**: Audit log cần evidence cho thanh tra, nhưng 100% rate quá tốn I/O.
- **Solution**: Sample rate configurable (ADR-006, default 1%).
- **Lý do batch + ticker**: Giảm DB round-trip (500 record/batch, flush 5s hoặc khi đầy).
- **Non-blocking emit**: channel buffered → drop nếu full, log warning. Trade-off: chấp nhận miss < 0.01% trong burst để bảo vệ pipeline hot path.

## M-9: E2E testcontainers
- **Root cause**: Refactor lớn cần regression test thực tế thay vì mock.
- **Solution**: postgres:16-alpine + migration + 3 strategy seed + assert đầu ra cụ thể.
- **Lý do build tag `integration`**: Tránh tốn time CI unit test.
- **Acceptance định lượng**: 3 strategy đầu ra khớp + audit log ≥ 3 record.

## Tổng impact P1
- Worker pipeline E2E dùng Strategy engine, không còn `"***"` literal.
- Audit log generate evidence compliance.
- Test bảo vệ regression dài hạn.

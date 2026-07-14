# Phân tích Chi tiết & Báo cáo Audit: Recon Module

> **Scope**: `internal/handler/recon/` + `internal/service/recon/`  
> **Tổng LOC phân tích**: ~8,535 LOC

---

## 1. Dead Code (13 items)

### Handler Layer (5 items)
- `TypeReconPrune` (`recon_base_handler.go:31`): Hằng số `"prune"` khai báo nhưng không được sử dụng.
- `healFetchResult` struct (`recon_heal_fetch.go:116-121`): Chỉ được dùng bởi `marshalHealPayload` (cũng dead).
- `marshalHealPayload()` (`recon_heal_fetch.go:123-126`): Không được gọi ở bất kỳ đâu.
- `var _ = servicerecon.ReconCore{}` (`recon_heal_fetch.go:128`): Blank identifier vô nghĩa.
- `SanitizeRetryRawJSONForTest()` (`recon_sysops_handler.go:375-377`): Hàm test helper không có test nào gọi.

### Service Layer (8 items)
- `fnvHash32` + `var _ = fnvHash32` (`recon_engine.go:334, 344`): Khai báo nhưng không dùng.
- `buildLegacyChunkHash` (`recon_legacy.go:45`): Không được gọi.
- `hashIDPlusTs` (phiên bản time.Time) (`recon_hash.go:158`): Chỉ wrap bởi test; runtime dùng phiên bản `int64`.
- `pickScanRange` (`recon_tier_a.go:328`): "backward compat wrapper" không ai gọi.
- `InvalidateMaskCache` (`recon_heal.go:77`): Hàm no-op, comment nói "now a no-op".
- `ListAllIDs` (`recon_stream.go:73`): Bị deprecated bởi `StreamAllIDs`.
- `GetIDs` (`recon_dest_stream.go:67`): Legacy API, không caller.
- `GetAllIDs` (`recon_dest_stream.go:97`): Trả về empty slice + cảnh báo.
- Biến `useDomainTS` (`recon_dest_query.go:196, 236, 271`): Gán giá trị rồi bypass `_ = useDomainTS`.

---

## 2. Legacy Code cần xem xét deprecate
- `internal/service/recon/recon_dest_legacy.go` (30 dòng): Chứa `GetChunkHashes`.
- `internal/service/recon/recon_legacy.go` (56 dòng): Chứa `GetChunkHashes` (source-side), `redactURL` và `buildLegacyChunkHash`.
- Struct `ChunkHash` (`recon_models.go:32`): Chỉ dùng cho logic chunk hash cũ.

---

## 3. Code trùng lặp (Code Duplication)
- **Trùng lặp Tier B (~170 dòng)**: Logic giữa `RunHashWindowCheckB` và `RunDeepCheckB` trong `recon_tier_b.go:196-545` gần như giống hệt nhau. Nên gom chung vào một hàm private `runSegmentBCommon`.
- **Trùng lặp cấu trúc NATS Handler**: Phân tích parse + collect heal IDs lặp 4 lần cho Segment A và B tại `recon_check_heal_handler.go` và `recon_execute_heal_handler.go`.
- **Lookback context**: Trùng lặp logic tạo context `manual_lookback` và `cold_lookback` bằng string key lặp 3 lần.
- **Tách Schema/Table**: Logic string split `schema.table` lặp hơn 6 lần inline thay vì gọi hàm helper `splitSchemaTable` từ `recon_dest_query.go:91`.

---

## 4. Code Smell & Vấn đề Logic nguy hiểm (Critical)
- **🔴 Rủi ro SQL Injection** (`recon_execute_heal_handler.go:292`):
  Dùng `fmt.Sprintf` với định dạng `%q` cho tên bảng động trong `h.shadowDB.Raw(...)`. `%q` dùng escape kiểu Go, không an toàn để trích dẫn SQL identifiers.
- **🔴 Context Key dùng kiểu string** (5+ chỗ):
  Sử dụng string literals trực tiếp như `"manual_lookback"` hay `"cold_lookback"` trong `context.WithValue` gây rủi ro collision.
- **🟡 Khớp prefix sai trong Test**:
  `recon_heal_v4_test.go:232` mong đợi prefix `"sd_"` nhưng code chạy thực tế dùng `"shadow_"`.
- **🟡 Prune Logic không hoạt động thực tế**:
  Các logic prune chỉ đếm số lượng và log "soft-delete pending" mà không xóa dữ liệu thực, làm sai lệch số liệu healed count.
- **🟡 ShadowPrefix Hardcode**:
  Handler đang dùng hằng số `"shadow_"` trực tiếp, bỏ qua cấu hình cấu trúc schema động từ biến môi trường `CDC_SHADOW_SCHEMA_PREFIX`.

---

## 5. Danh sách God Functions (>100 LOC)
1. `CheckAllUnified` (`recon_smoke.go:428-713`, 285 dòng)
2. `RunSmokeCheck` (`recon_tier_a.go:626-862`, 236 dòng)
3. `RunHashWindowCheck` (`recon_tier_a.go:865-1050`, 185 dòng)
4. `RunDeepCheckB` (`recon_tier_b.go:368-545`, 177 dòng)
5. `RunHashWindowCheckB` (`recon_tier_b.go:196-366`, 170 dòng)
6. `DLQWorker.tryApply` (`dlq_worker.go:268-371`, 103 dòng)
7. `HandleDebeziumSignal` (`recon_sysops_handler.go:139-232`, 93 dòng)

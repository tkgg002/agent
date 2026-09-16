# Phân Tích Gốc Rễ: Sự Cố Recon Smoke Báo Lỗi Kép SQLSTATE 25P02

**Ngày lập:** 2026-09-15  
**Tác giả:** Brain (Architect)  
**Trạng thái:** Confirmed Root Cause  

---

## 1. Hiện Tượng Gặp Phải (User Report)
Khi User chạy Smoke Recon kiểm tra đối soát, hệ thống báo 2 lỗi:
1. `prefetch.master ERROR: current transaction is aborted, commands ignored until end of transaction block (SQLSTATE 25P02)`
2. `shadow_err=<nil> master_err=ERROR: current transaction is aborted, commands ignored until end of transaction block (SQLSTATE 25P02)`

---

## 2. Truy Vết Mã Nguồn & Vị Trí Phát Sinh

### Vị trí 1: Span `prefetch.master` và `scanExact`
- **File:** `centralized-data-service/internal/service/recon/recon_smoke.go` (L104–L115):
  ```go
  traceCtx, span := observability.ChildSpan(ctx, "prefetch."+kind,
      attribute.String("db.system", "postgresql"),
      attribute.String("db.sql.table", relation),
      attribute.String("table.role", kind),
      attribute.String("query.total.method", "count_star_exact"),
  )
  defer func() { observability.EndSpan(span, &err) }()

  total, err = agent.CountRows(traceCtx, relation, "_gpay_id")
  ```
- **File:** `centralized-data-service/internal/service/recon/recon_dest_query.go` (L29–L45):
  ```go
  result, err := da.breaker.Execute(func() (interface{}, error) {
      tx := da.readOnlyDB(ctx)
      defer tx.Rollback()
      var count int64
      if pkColumn != "" && validateIdent(pkColumn) == nil {
          sql := fmt.Sprintf(`SELECT COUNT(%s) FROM %s`, quoteIdent(pkColumn), quoteRelation(tableName))
          if err := tx.Raw(sql).Scan(&count).Error; err == nil {
              return count, nil
          }
          // Fallback về COUNT(*) nếu pkColumn không tồn tại trên bảng
      }
      sql := fmt.Sprintf(`SELECT COUNT(*) FROM %s`, quoteRelation(tableName))
      if err := tx.Raw(sql).Scan(&count).Error; err != nil {
          return nil, err
      }
      return count, nil
  })
  ```

### Vị trí 2: Log `shadow_err=<nil> master_err=...` trong Segment B
- **File:** `centralized-data-service/internal/service/recon/recon_smoke.go` (L527–L531):
  ```go
  if shadowErr != nil || masterErr != nil {
      status = "failed"
      errMsg := fmt.Sprintf("shadow_err=%v master_err=%v", shadowErr, masterErr)
      runErr = fmt.Errorf("%s", errMsg)
      rc.finishRun(ctx, handle, "failed", errMsg)
  ...
  ```

---

## 3. Phân Tích Nguyên Nhân Gốc Rễ Kép (Dual Root Cause)

### Root Cause 1: Lỗi Logic Fallback trong PostgreSQL Transaction (`CountRows`)
- Trong PostgreSQL, khi một câu lệnh SQL bên trong một Transaction Block (`BEGIN ... COMMIT/ROLLBACK`) bị lỗi:
  - Transaction ngay lập tức bị chuyển sang trạng thái **ABORTED (`INERROR`)**.
  - Bất kỳ câu lệnh tiếp theo nào được gửi trên transaction này (ngoại trừ `ROLLBACK`) đều sẽ bị PostgreSQL từ chối với mã lỗi `SQLSTATE 25P02`:
    `ERROR: current transaction is aborted, commands ignored until end of transaction block (SQLSTATE 25P02)`
- Trong `CountRows`:
  1. Mở transaction: `tx := da.readOnlyDB(ctx)` (hàm này thực thi `BEGIN` + `SET TRANSACTION READ ONLY`).
  2. Chạy câu lệnh 1: `SELECT COUNT(_gpay_id) FROM tableName`.
  3. Khi câu lệnh 1 bị lỗi (ví dụ: relation không tồn tại, hoặc cột `_gpay_id` không có), `tx` chuyển sang trạng thái Aborted.
  4. Khối fallback ngay bên dưới cố chấp chạy câu lệnh 2: `SELECT COUNT(*) FROM tableName` **trên chính transaction `tx` đã bị aborted**!
  5. PostgreSQL trả về lỗi `SQLSTATE 25P02`.
  6. Lỗi `25P02` này đè bẹp và nuốt mất hoàn toàn lỗi thực sự của câu lệnh 1 (như `relation does not exist` hoặc `column "_gpay_id" does not exist`).

### Root Cause 2: Khiếm Khuyết Kiến Trúc: Recon Bị Mù Multi-Connection Master
- Tại sao câu lệnh 1 lại bị lỗi khi query bảng Master `master_centrallized_export_service.export_jobs`?
- **Khảo sát hệ thống:**
  - Bảng Master `master_centrallized_export_service.export_jobs` của User vừa được tạo và approve trên target connection `master_2` (container PostgreSQL port 5437).
  - Tuy nhiên, trong `internal/server/server_setup.go`:
    ```go
    if masterDB, mErr := registry.GetDB(database.RoleDestination); mErr == nil {
        reconCore.SetMasterAgent(recon.NewReconDestAgentWithConfig(masterDB, masterDB, recon.ReconDestAgentConfig{}, logger))
        reconCore.SetPlaneDBs(shadowDB, masterDB)
    }
    ```
    `reconCore` chỉ được gán một instance duy nhất `rc.masterAgent` trỏ vào `database.RoleDestination` (`default_master`, port 5434).
  - Trong `recon_engine_segment_b.go`:
    `MasterBindingRef` không có trường `MasterConnectionKey`. Hàm `ListActiveMasterBindings` không SELECT connection code của Master.
  - Trong `recon_smoke.go`:
    Khi duyệt danh sách `validRefs`, code gọi:
    `addTarget(r.MasterRel(), rc.masterAgent, "master", r.ShadowTable)`
    Toàn bộ các master target bất kể thuộc connection nào đều bị ép chạy qua `rc.masterAgent` (port 5434)!
  - Trên database `default_master` (port 5434), bảng `master_centrallized_export_service.export_jobs` **KHÔNG TỒN TẠI** (nó chỉ tồn tại trên container `master_2` port 5437).
  - Kết quả: PostgreSQL port 5434 trả về `relation does not exist` $\rightarrow$ trigger lỗi Fallback Transaction Aborted ở Root Cause 1 $\rightarrow$ sinh ra lỗi `SQLSTATE 25P02`.

---

## 4. Rủi Ro Tiềm Ẩn Nếu Không Xử Lý Triệt Để
1. **Va chạm Cache (Collision Risk):** Nếu có 2 bảng master cùng tên trên 2 database đích khác nhau (vd: `export_jobs` trên `default_master` và `export_jobs` trên `master_2`), hàm `smokeCountCache` và `scanIdx` của Recon dedup theo tên `Relation`, dẫn đến việc COUNT bảng này nhưng ghi đè kết quả của bảng kia.
2. **Sai Lệch Đối Soát Segment B (Segment B Drift):** Sau khi `CountRows` chạy được, `RunTotalOnlyB` vẫn gọi `rc.masterAgent.MaxWindowTs`, `CountInWindow`, `CountRecentDeletedRows`, và `BucketCounts` vào port 5434, dẫn tới tính sai hoàn toàn drift giữa Shadow và Master.

---

## 5. Kết Luận
Cần thực hiện giải pháp 2 tầng:
1. **Tầng vi mô (Defensive Code):** Sửa hàm `CountRows` trong `recon_dest_query.go` để rollback transaction khi lệnh 1 lỗi, mở transaction mới trước khi chạy fallback; đồng thời trong `scanExact`, khi `kind == "master"` thì truyền `pkColumn = ""` để chạy thẳng `SELECT COUNT(*)`.
2. **Tầng kiến trúc (Multi-Connection Recon):** Mở rộng `ReconCore` để tích hợp `ConnectionManager`, phân giải động `ReconDestAgent` theo `master_connection_key`, định danh target theo `kind:connKey:rel` trong toàn bộ pipeline Recon Smoke.

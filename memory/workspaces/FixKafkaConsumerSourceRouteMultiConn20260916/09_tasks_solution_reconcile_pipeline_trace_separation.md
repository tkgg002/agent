# 09_tasks_solution: HỒ SƠ GIẢI PHÁP PHÂN TÁCH PIPELINE SPAN THEO NGUỒN ĐA KẾT NỐI

## 1. Mổ xẻ Điểm Nghẽn Gộp Nhầm Trace Trong `recon_smoke.go`

### 1.1. Đoạn code gây lỗi gộp chùm
Tại `centralized-data-service/internal/service/recon/recon_smoke.go` (dòng 774–795):
```go
for _, e := range validEntries {
    key := e.TargetTable // CHỈ LÀ "trans_his"!
    if _, exists := tableCtxs[key]; !exists {
        tCtx, tSpan := observability.ChildSpan(ctx, "pipeline:"+key,
            attribute.String("pipeline.table", key),
        )
        tableCtxs[key] = tCtx
        ...
    }
}
for _, r := range validRefs {
    key := r.ShadowTable // CHỈ LÀ "trans_his"!
    if _, exists := tableCtxs[key]; !exists {
        tCtx, tSpan := observability.ChildSpan(ctx, "pipeline:"+key,
            attribute.String("pipeline.table", key),
        )
        tableCtxs[key] = tCtx
        ...
    }
}
```
Khi có nhiều kết nối nguồn (ví dụ: `traitestmongodevct` và kết nối khác) cùng có collection `trans_his`:
- Cả 2 đều có `TargetTable = "trans_his"`.
- Vòng lặp thấy `tableCtxs["trans_his"]` đã tồn tại từ pipeline thứ nhất, nên **BỎ QUA KHÔNG TẠO SPAN CHO PIPELINE THỨ HAI**!
- Hậu quả: Pipeline thứ hai bị nuốt chửng, chạy chung context của pipeline thứ nhất, trên SigNoz chỉ có 1 span duy nhất mang tên `pipeline:trans_his`.

---

## 2. Giải pháp Kiến trúc: Tách Riêng Từng Pipeline Theo Khóa Duy Nhất

### 2.1. Khóa phân lập Pipeline (Unique Pipeline Key)
Mỗi pipeline trong hệ thống CDC đa kết nối được định danh duy nhất bởi:
`pipelineKey = shadow_schema + "." + shadow_table` (ví dụ: `shadow_traitestmongodevct.trans_his`).
Nếu 2 kết nối nguồn khác nhau đẩy về cùng table name, chúng sẽ có 2 shadow schema khác nhau $\rightarrow$ 2 `pipelineKey` hoàn toàn độc lập!

### 2.2. Xây dựng cấu trúc dữ liệu và định dạng span name trực quan
```go
type PipelineFlowInfo struct {
	PipelineKey         string
	SourceEngine        string
	SourceConnectionKey string
	SourceDB            string
	SourceTable         string
	ShadowSchema        string
	ShadowTable         string
	MasterConnectionKey string
	MasterSchema        string
	MasterTable         string
}

func (p *PipelineFlowInfo) FormatSpanName() string {
	engine := strings.TrimSpace(p.SourceEngine)
	if strings.EqualFold(engine, "mongodb") || strings.EqualFold(engine, "mongo") {
		engine = "Mongo"
	} else if strings.EqualFold(engine, "postgresql") || strings.EqualFold(engine, "postgres") {
		engine = "Postgres"
	} else if strings.EqualFold(engine, "mysql") || strings.EqualFold(engine, "mariadb") {
		engine = "MySQL"
	} else if engine == "" {
		engine = "Source"
	}

	srcConn := p.SourceConnectionKey
	if srcConn == "" {
		srcConn = "default"
	}

	sourcePart := ""
	if p.SourceTable != "" {
		if p.SourceDB != "" {
			sourcePart = fmt.Sprintf("[%s:%s.%s.%s]", engine, srcConn, p.SourceDB, p.SourceTable)
		} else {
			sourcePart = fmt.Sprintf("[%s:%s.%s]", engine, srcConn, p.SourceTable)
		}
	}

	shadowSchema := p.ShadowSchema
	if shadowSchema == "" {
		shadowSchema = "public"
	}
	shadowPart := fmt.Sprintf("[Shadow:%s.%s]", shadowSchema, p.ShadowTable)

	masterPart := ""
	if p.MasterTable != "" {
		mConn := p.MasterConnectionKey
		if mConn == "" {
			mConn = "default"
		}
		mSchema := p.MasterSchema
		if mSchema == "" {
			mSchema = "public"
		}
		masterPart = fmt.Sprintf("[Master:%s:%s.%s]", mConn, mSchema, p.MasterTable)
	}

	if sourcePart != "" && masterPart != "" {
		return fmt.Sprintf("pipeline: %s -> %s -> %s", sourcePart, shadowPart, masterPart)
	} else if sourcePart != "" {
		return fmt.Sprintf("pipeline: %s -> %s", sourcePart, shadowPart)
	} else if masterPart != "" {
		return fmt.Sprintf("pipeline: %s -> %s", shadowPart, masterPart)
	}
	return fmt.Sprintf("pipeline: %s", shadowPart)
}
```

### 2.3. Khởi tạo Span độc lập trong `recon_smoke.go`
```go
	// ── KHỞI TẠO PIPELINE CONTEXTS PHÂN LẬP ──────────────────────────────────
	pipelineFlows := make(map[string]*PipelineFlowInfo)

	for _, e := range validEntries {
		key := e.QualifiedTarget() // "shadow_traitestmongodevct.trans_his"
		srcConn := ""
		if rc.metadata != nil {
			if route := rc.metadata.ResolveTargetRoute(e.TargetTable); route != nil {
				srcConn = route.SourceConnectionKey
			}
		}
		pipelineFlows[key] = &PipelineFlowInfo{
			PipelineKey:         key,
			SourceEngine:        e.SourceType,
			SourceConnectionKey: srcConn,
			SourceDB:            e.SourceDB,
			SourceTable:         e.SourceTable,
			ShadowSchema:        e.ShadowSchema,
			ShadowTable:         e.TargetTable,
		}
	}

	for _, r := range validRefs {
		key := r.ShadowRel() // "shadow_traitestmongodevct.trans_his"
		flow, exists := pipelineFlows[key]
		if !exists {
			flow = &PipelineFlowInfo{
				PipelineKey:         key,
				ShadowSchema:        r.ShadowSchema,
				ShadowTable:         r.ShadowTable,
				SourceEngine:        r.SourceEngine,
				SourceConnectionKey: r.SourceConnectionKey,
				SourceDB:            r.SourceDB,
				SourceTable:         r.SourceTable,
			}
			pipelineFlows[key] = flow
		}
		flow.MasterConnectionKey = r.MasterConnectionKey
		flow.MasterSchema = r.MasterSchema
			}
			pipelineFlows[key] = flow
		}
		flow.MasterConnectionKey = r.MasterConnectionKey
		flow.MasterSchema = r.MasterSchema
		flow.MasterTable = r.MasterTable

		// Bổ sung dữ liệu nguồn chính xác từ SQL JOIN của Segment B (cập nhật độc lập từng trường)
		if r.SourceConnectionKey != "" {
			flow.SourceConnectionKey = r.SourceConnectionKey
		}
		if r.SourceDB != "" {
			flow.SourceDB = r.SourceDB
		}
		if r.SourceTable != "" {
			flow.SourceTable = r.SourceTable
		}
		if r.SourceEngine != "" {
			flow.SourceEngine = r.SourceEngine
		}
	}

	pipelineCtxs := make(map[string]context.Context)
	pipelineEndFns := make([]func(), 0)

	for pKey, flow := range pipelineFlows {
		spanName := flow.FormatSpanName()
		pCtx, pSpan := observability.ChildSpan(ctx, spanName,
			attribute.String("pipeline.key", pKey),
			attribute.String("pipeline.flow", spanName),
			attribute.String("pipeline.source_engine", flow.SourceEngine),
			attribute.String("pipeline.source_conn", flow.SourceConnectionKey),
			attribute.String("pipeline.source_db", flow.SourceDB),
			attribute.String("pipeline.source_table", flow.SourceTable),
			attribute.String("pipeline.shadow_schema", flow.ShadowSchema),
			attribute.String("pipeline.shadow_table", flow.ShadowTable),
			attribute.String("pipeline.master_conn", flow.MasterConnectionKey),
			attribute.String("pipeline.master_schema", flow.MasterSchema),
			attribute.String("pipeline.master_table", flow.MasterTable),
		)
		pipelineCtxs[pKey] = pCtx
		sp := pSpan
		pipelineEndFns = append(pipelineEndFns, func() { observability.EndSpan(sp, nil) })
	}
	defer func() {
		for _, endFn := range pipelineEndFns {
			endFn()
		}
	}()
```

### 2.4. Khớp nối Context với `ScanTarget` và Execution
- Khi add target vào `scanTargets`: gán `PipelineKey: e.QualifiedTarget()` / `r.ShadowRel()`.
- Khi chạy prefetch: lấy `pCtx := pipelineCtxs[t.PipelineKey]`.
- Khi chạy Segment A: lấy `pipelineCtx := pipelineCtxs[entry.QualifiedTarget()]`.
- Khi chạy Segment B: lấy `pipelineCtx := pipelineCtxs[ref.ShadowRel()]`.
- Toàn bộ các pipeline trùng tên bảng nhưng khác kết nối/schema sẽ được **tách hoàn toàn thành các span riêng biệt**.

---

## 3. Khắc phục Điểm Nghẽn Runtime & Phương Án Kiểm Thử Tích Hợp Thật (DoD G3/G6)

### 3.1. Các điểm sửa đổi runtime cốt lõi
1. **`recon_smoke.go:848`:** Thay vì gọi `ResolveTargetRoute(e.TargetTable)` gây bốc nhầm route khi trùng tên bảng, ưu tiên gọi `ResolveTargetRoute(e.QualifiedTarget())` trước.
2. **`recon_smoke.go:881`:** Thay vì điều kiện gộp `if flow.SourceEngine == ""`, cập nhật độc lập từng trường nguồn từ `r` sang `flow`.
3. **`recon_smoke.go:803`:** Thêm deduplication map trong `validEntries` theo `QualifiedTarget()`.
4. **`metadata_registry_service.go:403`:** Đổi `ListTableConfigs()` sang duyệt `rs.idCache` thay vì `rs.targetCache`.

### 3.2. Đặc tả Kiểm thử Tích hợp Thực tế trên Database `cdc_dw` (Port 5433)
- Mở kết nối GORM thực tế tới `postgres://gpay_admin:gpay_pass@localhost:5433/cdc_dw`.
- Đăng ký `tracetest.NewInMemoryExporter()` vào OpenTelemetry TracerProvider.
- Khởi tạo `ReconCore` với DB thật và metadata thật.
- Gọi `rc.CheckAllUnified(ctx)`.
- Assert và in ra console toàn bộ spans được emit:
  * Span 1: `pipeline: [Mongo:traitestmongodevct.core-trans-proxy-history-service.trans-his] -> [Shadow:shadow_traitestmongodevct.trans_his] -> [Master:postgres-develop:public.cdc_transaction_history]`
  * Span 2: `pipeline: [Mongo:traitestctphs.core-trans-proxy-history-service.trans-his] -> [Shadow:shadow_traitestctphs.trans_his] -> [Master:default_master:master_core_trans_proxy_history_service.trans_his]`
- Khẳng định 100% bằng chứng spans thực tế trên dữ liệu thật, triệt tiêu hoàn toàn ảo tưởng test mock.

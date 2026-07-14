# Hồ Sơ Giải Pháp Kỹ Thuật: Cải Tiến Luồng Heal & Định Tuyến Theo Chế Độ Chọn (Window vs Full-Diff)

Hồ sơ này mô tả chi tiết các thay đổi mã nguồn cần thiết cho cả Frontend (FE) và Backend (BE) để thực hiện định tuyến heal theo tham số từ UI, ẩn input from/to khi chọn Window, hỗ trợ Hot/Cold lookback và thay thế cơ chế Debezium NATS signal bằng direct write.

---

## 1. Frontend (FE) - Modal Kích Hoạt Heal Trên CMS-Web

### Component JSX Mock Diff (React)
Thêm các controls radio buttons và DatePickers cho modal trigger heal:

```jsx
// src/components/ReconHealModal.jsx
import React, { useState, useEffect } from 'react';

export function ReconHealModal({ table, segment, onClose, onSubmit }) {
  const [mode, setMode] = useState('window'); // 'window' hoặc 'full_diff'
  const [lookback, setLookback] = useState('cold'); // 'hot' hoặc 'cold'
  const [startTime, setStartTime] = useState('');
  const [endTime, setEndTime] = useState('');
  const [errorMsg, setErrorMsg] = useState('');

  // Tự động tính toán 7 ngày gần nhất khi chuyển sang Window mode
  useEffect(() => {
    if (mode === 'window') {
      const to = new Date();
      const from = new Date();
      from.setDate(to.getDate() - 7);
      setStartTime(from.toISOString().slice(0, 16)); // Format: YYYY-MM-DDTHH:mm
      setEndTime(to.toISOString().slice(0, 16));
      setErrorMsg('');
    } else {
      setStartTime('');
      setEndTime('');
    }
  }, [mode]);

  // Validate range 30 ngày cho Full-diff
  const handleTimeChange = (startVal, endVal) => {
    if (!startVal || !endVal) {
      setErrorMsg('Vui lòng chọn đầy đủ thời gian bắt đầu và kết thúc!');
      return;
    }
    const start = new Date(startVal);
    const end = new Date(endVal);
    if (end < start) {
      setErrorMsg('Thời gian kết thúc không được nhỏ hơn thời gian bắt đầu!');
      return;
    }
    const diffDays = (end - start) / (1000 * 60 * 60 * 24);
    if (diffDays > 30) {
      setErrorMsg('Khoảng thời gian quét Full-diff không được vượt quá 30 ngày để bảo vệ DB!');
    } else {
      setErrorMsg('');
    }
  };

  const handleSubmit = () => {
    if (mode === 'full_diff' && errorMsg) return;
    onSubmit({
      table,
      segment,
      mode,
      lookback: mode === 'window' ? lookback : undefined,
      start_time: mode === 'full_diff' ? new Date(startTime).toISOString() : undefined,
      end_time: mode === 'full_diff' ? new Date(endTime).toISOString() : undefined,
    });
  };

  return (
    <div className="glass-modal">
      <h3>Kích Hoạt Chữa Lành Dữ Liệu (Heal)</h3>
      <div className="radio-group">
        <label>
          <input
            type="radio"
            value="window"
            checked={mode === 'window'}
            onChange={(e) => setMode(e.target.value)}
          />
          Chế độ Window (Đối soát cửa sổ)
        </label>
        <label>
          <input
            type="radio"
            value="full_diff"
            checked={mode === 'full_diff'}
            onChange={(e) => setMode(e.target.value)}
          />
          Chế độ Full-diff (So khớp toàn bảng theo khoảng thời gian)
        </label>
      </div>

      {mode === 'window' ? (
        <div className="lookback-options">
          <label>
            <input
              type="radio"
              value="hot"
              checked={lookback === 'hot'}
              onChange={() => setLookback('hot')}
            />
            Hot Mode (2 giờ gần nhất)
          </label>
          <label>
            <input
              type="radio"
              value="cold"
              checked={lookback === 'cold'}
              onChange={() => setLookback('cold')}
            />
            Cold Lookback (7 ngày gần nhất)
          </label>
        </div>
      ) : (
        <div className="time-pickers">
          <div>
            <label>Từ thời gian:</label>
            <input
              type="datetime-local"
              value={startTime}
              onChange={(e) => {
                setStartTime(e.target.value);
                handleTimeChange(e.target.value, endTime);
              }}
            />
          </div>
          <div>
            <label>Đến thời gian:</label>
            <input
              type="datetime-local"
              value={endTime}
              onChange={(e) => {
                setEndTime(e.target.value);
                handleTimeChange(startTime, e.target.value);
              }}
            />
          </div>
        </div>
      )}

      {errorMsg && <p className="error-text">{errorMsg}</p>}

      <div className="modal-actions">
        <button onClick={onClose}>Hủy</button>
        <button onClick={handleSubmit} disabled={mode === 'full_diff' && !!errorMsg}>
          Thực Hiện
        </button>
      </div>
    </div>
  );
}
```

### Chi Tiết Thay Đổi Mã Nguồn Frontend (Vite/React)

#### 1. [MODIFY] [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)
Cập nhật các mutation `useCheckTableMutation` và `useHealMutation` để chấp nhận thêm `lookback` cùng các tham số chế độ:
```diff
@@ -147,4 +147,4 @@
 export function useCheckTableMutation() {
-  return useMutation<void, Error, { table: string; tier: string; reason: string; sourceDatabase?: string; sourceTable?: string; shadowSchema?: string; shadowTable?: string }>({
-    mutationFn: async ({ table, tier, reason, sourceDatabase, sourceTable, shadowSchema, shadowTable }) => {
+  return useMutation<void, Error, { table: string; tier: string; reason: string; sourceDatabase?: string; sourceTable?: string; shadowSchema?: string; shadowTable?: string; lookback?: string }>({
+    mutationFn: async ({ table, tier, reason, sourceDatabase, sourceTable, shadowSchema, shadowTable, lookback }) => {
       await cmsApi.post(
@@ -156,2 +156,3 @@
           shadow_schema: shadowSchema || undefined,
           shadow_table: shadowTable || undefined,
+          lookback: lookback || undefined,
         },
@@ -190,5 +190,5 @@
 export function useHealMutation() {
-  return useMutation<void, Error, { table: string; reason: string; segment?: string; sourceDatabase?: string; sourceTable?: string; shadowSchema?: string; shadowTable?: string }>({
-    mutationFn: async ({ table, reason, segment, sourceDatabase, sourceTable, shadowSchema, shadowTable }) => {
+  return useMutation<void, Error, { table: string; reason: string; segment?: string; sourceDatabase?: string; sourceTable?: string; shadowSchema?: string; shadowTable?: string; mode?: string; startTime?: string; endTime?: string; lookback?: string }>({
+    mutationFn: async ({ table, reason, segment, sourceDatabase, sourceTable, shadowSchema, shadowTable, mode, startTime, endTime, lookback }) => {
       await cmsApi.post(
         '/api/reconciliation/heal',
         {
           reason,
           table,
           segment: segment || undefined,
           source_database: sourceDatabase || undefined,
           source_table: sourceTable || undefined,
           shadow_schema: shadowSchema || undefined,
           shadow_table: shadowTable || undefined,
+          mode: mode || undefined,
+          start_time: startTime || undefined,
+          end_time: endTime || undefined,
+          lookback: lookback || undefined,
         },
         { headers: auditHeaders(reason) },
       );
```

#### 2. [MODIFY] [ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx)
Thêm UI Controls đặc thù cho Heal và Check Tier 2 (radio button chọn Window/Full-diff, ẩn inputs from/to trong Window mode, thêm Hot/Cold lookback options và validation 30 ngày):
```diff
@@ -10,9 +10,11 @@
   actionLabel: string;
   danger?: boolean;
+  isHeal?: boolean;
+  isCheckTier2?: boolean;
-  onConfirm: (reason: string) => Promise<void> | void;
+  onConfirm: (reason: string, mode?: string, startTime?: string, endTime?: string, lookback?: string) => Promise<void> | void;
   onCancel: () => void;
   loading?: boolean;
 }
 
 export default function ConfirmDestructiveModal({
@@ -28,4 +30,6 @@
   danger = false,
+  isHeal = false,
+  isCheckTier2 = false,
   onConfirm,
   onCancel,
   loading = false,
@@ -33,4 +36,10 @@
   const [reason, setReason] = useState('');
   const [submitting, setSubmitting] = useState(false);
+  const [mode, setMode] = useState('window'); // 'window' hoặc 'full_diff'
+  const [lookback, setLookback] = useState('cold'); // 'hot' hoặc 'cold'
+  const [startTime, setStartTime] = useState('');
+  const [endTime, setEndTime] = useState('');
+  const [timeError, setTimeError] = useState('');
 
   useEffect(() => {
@@ -39,4 +47,21 @@
       setReason('');
       setSubmitting(false);
+      setMode('window');
+      setLookback('cold');
+      setTimeError('');
+      
+      // Tính toán mặc định 7 ngày cho Window mode
+      const to = new Date();
+      const from = new Date();
+      from.setDate(to.getDate() - 7);
+      setStartTime(from.toISOString().slice(0, 16));
+      setEndTime(to.toISOString().slice(0, 16));
     }
   }, [open]);
+
+  const handleTimeChange = (startVal: string, endVal: string) => {
+    if (!startVal || !endVal) {
+      setTimeError('Vui lòng chọn đầy đủ thời gian bắt đầu và kết thúc!');
+      return;
+    }
+    const start = new Date(startVal);
+    const end = new Date(endVal);
+    if (end < start) {
+      setTimeError('Thời gian kết thúc không được nhỏ hơn thời gian bắt đầu!');
+      return;
+    }
+    const diffDays = (end.getTime() - start.getTime()) / (1000 * 60 * 60 * 24);
+    if (diffDays > 30) {
+      setTimeError('Khoảng thời gian quét Full-diff không được vượt quá 30 ngày!');
+    } else {
+      setTimeError('');
+    }
+  };
 
   const trimmed = reason.trim();
   const isReasonValid = trimmed.length >= MIN_REASON_LENGTH;
-  const isFormValid = isReasonValid && !timeError;
+  const isFormValid = isReasonValid && (mode === 'window' || !timeError);
   const isBusy = loading || submitting;
 
   const handleOk = async () => {
-    if (!isReasonValid || isBusy) return;
+    if (!isFormValid || isBusy) return;
     try {
       setSubmitting(true);
-      await onConfirm(trimmed);
```diff
@@ -151,4 +151,5 @@
   | { kind: 'heal'; table: string; segment?: string; record?: ReconReport }
+  | { kind: 'heal'; table: string; segment?: string; record?: ReconReport; isHeal?: boolean }
 
@@ -220,5 +221,5 @@
   const openHeal = (record: ReconReport) =>
     setModalPlan({
-      action: { kind: 'heal', table: record.target_table, segment: record.segment, record },
+      action: { kind: 'heal', table: record.target_table, segment: record.segment, record, isHeal: true },
       title: `Chữa lành drift cho ${record.target_table}`,
       description:
@@ -277,4 +278,4 @@
 
-  const handleConfirm = async (reason: string) => {
+  const handleConfirm = async (reason: string, mode?: string, startTime?: string, endTime?: string, lookback?: string) => {
     if (!modalPlan) return;
     const action = modalPlan.action;
@@ -298,5 +299,5 @@
       } else if (action.kind === 'heal') {
         const row = action.record || reportByTarget.get(action.table);
         await heal.mutateAsync({
           table: action.table,
           reason,
           segment: action.segment,
           sourceDatabase: row?.source_db,
           sourceTable: row?.source_table || undefined,
           shadowSchema: row?.shadow_schema || undefined,
           shadowTable: row?.shadow_table || undefined,
+          mode,
+          startTime,
+          endTime,
+          lookback,
         });
         message.success(`Đang chữa lành ${action.table}…`);
@@ -1040,4 +1043,5 @@
           actionLabel={modalPlan.actionLabel}
           danger={modalPlan.danger}
+          isHeal={modalPlan.action.kind === 'heal'}
           loading={mutationPending}
```

---

## 2. Backend (BE) - Diffs Chi Tiết Cho Dịch Vụ Go

### 1. `internal/handler/recon/recon_handler_run.go`
Mở rộng payload struct unmarshal và truyền cấu hình vào `healSegmentA`:

```diff
@@ -204,9 +204,13 @@
 
 	var payload struct {
-		Table   string `json:"table"`
-		Segment string `json:"segment"` // ""/"source_shadow" = A; "shadow_master" = B
-		Legacy  bool   `json:"legacy"`  // true = ép đường bypass cũ (escape hatch, sẽ gỡ sau P4)
+		Table     string `json:"table"`
+		Segment   string `json:"segment"`
+		Legacy    bool   `json:"legacy"`
+		Mode      string `json:"mode"`       // "window" hoặc "full_diff"
+		StartTime string `json:"start_time"` // RFC3339 string
+		EndTime   string `json:"end_time"`   // RFC3339 string
+		Lookback  string `json:"lookback"`   // "hot" hoặc "cold"
 	}
 	json.Unmarshal(msg.Data, &payload)
 
@@ -225,3 +229,3 @@
 		if payload.Segment == "shadow_master" {
 			h.healSegmentB(ctx, msg, payload.Table)
 		} else {
-			h.healSegmentA(ctx, msg, payload.Table)
+			h.healSegmentA(ctx, msg, payload.Table, payload.Mode, payload.StartTime, payload.EndTime, payload.Lookback)
 		}
 		return
 	}
```

---

### 2. `internal/service/recon/recon_stream.go`
Thêm hàm mới `StreamIDsInTimeRange` hỗ trợ range query cho MongoDB và PostgreSQL source agents:

```go
// StreamIDsInTimeRange streams IDs matching a timestamp filter from the source (MongoDB or PostgreSQL).
func (sa *ReconSourceAgent) StreamIDsInTimeRange(ctx context.Context, sourceURL, database, collection, timestampField string, startTime, endTime time.Time) (<-chan string, <-chan error) {
	if isPostgres(sourceURL) {
		return sa.streamIDsPostgresInTimeRange(ctx, sourceURL, database, collection, timestampField, startTime, endTime)
	}
	idChan := make(chan string, 1000)
	errChan := make(chan error, 1)

	go func() {
		defer close(idChan)
		defer close(errChan)

		client, err := sa.getClient(ctx, sourceURL)
		if err != nil {
			errChan <- fmt.Errorf("get client: %w", err)
			return
		}
		coll := sa.secondaryColl(client, database, collection)

		batchSize := int64(sa.cfg.BatchSize)
		if batchSize <= 0 {
			batchSize = 1000
		}

		var lastID interface{} = nil

		for {
			if err := ctx.Err(); err != nil {
				errChan <- err
				return
			}

			// Lọc theo khoảng thời gian hỗ trợ cả kiểu Date và kiểu EpochMs
			filter := bson.M{
				"$or": []bson.M{
					{timestampField: bson.M{"$gte": startTime, "$lt": endTime}},
					{timestampField: bson.M{"$gte": startTime.UnixMilli(), "$lt": endTime.UnixMilli()}},
				},
			}
			if lastID != nil {
				filter = bson.M{
					"$and": []bson.M{
						filter,
						{"_id": bson.M{"$gt": lastID}},
					},
				}
			}

			opts := options.Find().
				SetProjection(bson.M{"_id": 1}).
				SetSort(bson.D{{Key: "_id", Value: 1}}).
				SetLimit(batchSize)

			result, err := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
				return coll.Find(ctx, filter, opts)
			})
			if err != nil {
				errChan <- fmt.Errorf("find range batch (after _id=%v): %w", lastID, err)
				return
			}

			cursor := result.(*mongo.Cursor)
			var batchCount int64

			for cursor.Next(ctx) {
				if err := sa.limiter.Wait(ctx); err != nil {
					cursor.Close(ctx)
					errChan <- fmt.Errorf("rate limiter: %w", err)
					return
				}

				var doc struct {
					ID interface{} `bson:"_id"`
				}
				if err := cursor.Decode(&doc); err != nil {
					cursor.Close(ctx)
					errChan <- fmt.Errorf("decode _id: %w", err)
					return
				}

				idStr := extractMongoID(doc.ID)
				if idStr != "" {
					idChan <- idStr
					lastID = doc.ID
					batchCount++
				}
			}
			cursor.Close(ctx)

			if batchCount < batchSize {
				break
			}
		}
	}()

	return idChan, errChan
}

func (sa *ReconSourceAgent) streamIDsPostgresInTimeRange(ctx context.Context, sourceURL, database, collection, timestampField string, startTime, endTime time.Time) (<-chan string, <-chan error) {
	idChan := make(chan string, 1000)
	errChan := make(chan error, 1)

	go func() {
		defer close(idChan)
		defer close(errChan)

		db, err := sa.getPGClient(ctx, sourceURL)
		if err != nil {
			errChan <- fmt.Errorf("get client: %w", err)
			return
		}
		tableName := collection
		if err := validateIdent(tableName); err != nil {
			errChan <- err
			return
		}

		var schemaName, bareTable string
		if i := strings.IndexByte(tableName, '.'); i > 0 {
			schemaName = tableName[:i]
			bareTable = tableName[i+1:]
		} else {
			schemaName = "public"
			bareTable = tableName
		}

		pkCol, err := getPrimaryKeyColumn(db, schemaName, bareTable)
		if err != nil {
			errChan <- err
			return
		}
		if err := validateIdent(pkCol); err != nil {
			errChan <- err
			return
		}

		batchSize := int64(sa.cfg.BatchSize)
		if batchSize <= 0 {
			batchSize = 1000
		}

		var lastID interface{} = nil

		for {
			if err := ctx.Err(); err != nil {
				errChan <- err
				return
			}

			var querySql string
			var args []interface{}
			if lastID != nil {
				querySql = fmt.Sprintf(`SELECT %s FROM %s WHERE %s > ? AND %s >= ? AND %s < ? ORDER BY %s LIMIT ?`,
					quoteIdent(pkCol), quoteRelation(tableName), quoteIdent(pkCol), quoteIdent(timestampField), quoteIdent(timestampField), quoteIdent(pkCol))
				args = []interface{}{lastID, startTime, endTime, batchSize}
			} else {
				querySql = fmt.Sprintf(`SELECT %s FROM %s WHERE %s >= ? AND %s < ? ORDER BY %s LIMIT ?`,
					quoteIdent(pkCol), quoteRelation(tableName), quoteIdent(timestampField), quoteIdent(timestampField), quoteIdent(pkCol))
				args = []interface{}{startTime, endTime, batchSize}
			}

			result, err := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
				rows, err := db.WithContext(ctx).Raw(querySql, args...).Rows()
				if err != nil {
					return nil, err
				}
				return rows, nil
			})
			if err != nil {
				errChan <- fmt.Errorf("find range pg batch (after id=%v): %w", lastID, err)
				return
			}

			rows := result.(*sql.Rows)
			var batchCount int64
			for rows.Next() {
				if err := sa.limiter.Wait(ctx); err != nil {
					rows.Close()
					errChan <- fmt.Errorf("rate limiter: %w", err)
					return
				}

				var idVal interface{}
				if err := rows.Scan(&idVal); err != nil {
					rows.Close()
					errChan <- fmt.Errorf("scan id: %w", err)
					return
				}

				idStr := fmt.Sprintf("%v", idVal)
				idChan <- idStr
				lastID = idVal
				batchCount++
			}
			rows.Close()

			if batchCount < batchSize {
				break
			}
		}
	}()

	return idChan, errChan
}
```

---

### 3. `internal/service/recon/recon_tier_a.go`
Định nghĩa hàm quét an toàn có bộ lọc thời gian `TimeBoundedDiffMissingFromShadow`:

```go
// TimeBoundedDiffMissingFromShadow performs comparison between MongoDB and shadow in a specific time range.
func (rc *ReconCore) TimeBoundedDiffMissingFromShadow(ctx context.Context, entry source.TableRegistry, startTime, endTime time.Time) ([]string, int, error) {
	if rc.shadowPlane == nil {
		return nil, 0, fmt.Errorf("shadowPlane not wired")
	}

	tsCol, err := rc.resolveSourceTSField(ctx, entry)
	if err != nil {
		return nil, 0, fmt.Errorf("resolve ts field: %w", err)
	}

	// Tải ID từ Postgres Shadow DB trong khoảng thời gian
	var shadowIDs []string
	if err := rc.shadowPlane.WithContext(ctx).Raw(
		fmt.Sprintf(`SELECT "_source_id"::text FROM %s WHERE NOT "_deleted" AND "_source_id" IS NOT NULL AND %s >= ? AND %s < ?`,
			quoteRelation(entry.QualifiedTarget()), quoteIdent(tsCol), quoteIdent(tsCol)),
		startTime, endTime,
	).Scan(&shadowIDs).Error; err != nil {
		return nil, 0, fmt.Errorf("shadow list ids in range: %w", err)
	}

	shadowSet := make(map[string]struct{}, len(shadowIDs))
	for _, id := range shadowIDs {
		shadowSet[id] = struct{}{}
	}

	// Stream ID từ Source MongoDB trong khoảng thời gian
	idChan, errChan := rc.sourceAgent.StreamIDsInTimeRange(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, tsCol, startTime, endTime)
	srcCount := 0
	var missing []string
	var streamErr error

	for idChan != nil || errChan != nil {
		select {
		case id, ok := <-idChan:
			if !ok {
				idChan = nil
				break
			}
			srcCount++
			if _, exists := shadowSet[id]; !exists {
				missing = append(missing, id)
			}
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
				break
			}
			if err != nil {
				streamErr = err
			}
		}
	}

	if streamErr != nil {
		return nil, srcCount, fmt.Errorf("stream range partial error: %w", streamErr)
	}

	return missing, srcCount, nil
}
```

---

### 4. `internal/handler/recon/recon_heal_v4.go`
Thay đổi chữ ký hàm `healSegmentA`, validate khoảng thời gian và phân luồng routing theo mode chọn và lookback param:

```diff
@@ -273,4 +273,4 @@
-func (h *ReconHandler) healSegmentA(ctx context.Context, msg *nats.Msg, table string) {
+func (h *ReconHandler) healSegmentA(ctx context.Context, msg *nats.Msg, table string, mode string, startTimeStr string, endTimeStr string, lookback string) {
 	startTime := time.Now()
 	const op = "recon-heal-a"
 
@@ -308,6 +308,21 @@
 	// Mỗi lần click heal = 1 fresh scan độc lập, không phụ thuộc report cũ.
-	coldCtx := context.WithValue(ctx, "cold_lookback", true)
-
-	h.logger.Info("[heal-a] starting fresh RunTier2 scan (cold_lookback=true, no cached report)",
-		zap.String("table", table),
-	)
-
-	newReport := h.reconCore.RunTier2(coldCtx, *entry)
+
+	// 🔸 NHÁNH B: Quét và chữa lành theo chế độ Full-diff trong khoảng thời gian giới hạn
+	if mode == "full_diff" {
+		start, err1 := time.Parse(time.RFC3339, startTimeStr)
+		end, err2 := time.Parse(time.RFC3339, endTimeStr)
+		if err1 != nil || err2 != nil || end.Before(start) || end.Sub(start) > 30*24*time.Hour {
+			err := fmt.Errorf("invalid time range for full-diff: must be bounded within 30 days")
+			h.logger.Error("[heal-a] validation failed", zap.Error(err))
+			h.logActivity(op, table, "error", 0, err)
+			h.respondErr(msg, err)
+			return
+		}
+
+		h.logger.Info("[heal-a] starting full-diff range comparison",
+			zap.String("table", table), zap.Time("start", start), zap.Time("end", end))
+
+		missing, srcTotal, err := h.reconCore.TimeBoundedDiffMissingFromShadow(ctx, *entry, start, end)
+		if err != nil {
+			h.logActivity(op, table, "error", 0, err)
+			h.respondErr(msg, err)
+			return
+		}
+
+		if len(missing) == 0 {
+			h.logActivity(op, table, "noop", 0, nil)
+			if msg.Reply != "" {
+				res, _ := json.Marshal(map[string]any{"status": "noop", "note": "range check không phát hiện missing"})
+				msg.Respond(res)
+			}
+			return
+		}
+
+		if h.healThresholdBlocked(msg, op, table, len(missing), int64(srcTotal), int64(len(missing))) {
+			return
+		}
+
+		written, dispatchErr := h.FetchAndWriteByIDs(ctx, entry, missing)
+		if dispatchErr != nil {
+			h.logActivity(op, table, "error", int64(written), dispatchErr)
+			h.respondErr(msg, dispatchErr)
+			return
+		}
+
+		h.logActivity(op, table, "healed", int64(written), nil)
+		if msg.Reply != "" {
+			res, _ := json.Marshal(map[string]any{
+				"status":        "healed",
+				"segment":       "source_shadow",
+				"healed_count":  written,
+				"missing_count": len(missing),
+				"src_total":     srcTotal,
+				"dispatch_path": "direct_fetch_write",
+			})
+			msg.Respond(res)
+		}
+		return
+	}
+
+	// 🔸 NHÁNH A: Chạy chế độ Window
+	var runCtx context.Context
+	if lookback == "hot" {
+		runCtx = ctx
+		h.logger.Info("[heal-a] starting fresh RunTier2 scan (Hot Mode lookback=2h, no cached report)",
+			zap.String("table", table),
+		)
+	} else {
+		runCtx = context.WithValue(ctx, "cold_lookback", true)
+		h.logger.Info("[heal-a] starting fresh RunTier2 scan (Cold Lookback=7d, no cached report)",
+			zap.String("table", table),
+		)
+	}
+	newReport := h.reconCore.RunTier2(runCtx, *entry)
 
 	if newReport == nil {
@@ -340,140 +402,20 @@
 	if newReport.MissingCount == 0 && newReport.StaleCount == 0 && newReport.OrphanCount == 0 {
 	}
 
 	report := *newReport
 	// ... (Gom missingIDs và staleObj)
 	healIDs := append(append(append([]string{}, missingIDs...), staleObj.Mismatched...), staleObj.MissingFromSrc...)
 
 	// ... (logic xác định entry từ table...)

	// 🔸 NHÁNH B: Quét và chữa lành theo chế độ Full-diff trong khoảng thời gian giới hạn
	if mode == "full_diff" {
		start, err1 := time.Parse(time.RFC3339, startTimeStr)
		end, err2 := time.Parse(time.RFC3339, endTimeStr)
		if err1 != nil || err2 != nil || end.Before(start) || end.Sub(start) > 30*24*time.Hour {
			err := fmt.Errorf("invalid time range for full-diff: must be bounded within 30 days")
			h.logger.Error("[heal-a] validation failed", zap.Error(err))
			h.logActivity(op, table, "error", 0, err)
			h.respondErr(msg, err)
			return
		}

		h.logger.Info("[heal-a] starting full-diff range comparison",
			zap.String("table", table), zap.Time("start", start), zap.Time("end", end))

		missing, srcTotal, err := h.reconCore.TimeBoundedDiffMissingFromShadow(ctx, *entry, start, end)
		if err != nil {
			h.logActivity(op, table, "error", 0, err)
			h.respondErr(msg, err)
			return
		}

		if len(missing) == 0 {
			h.logActivity(op, table, "noop", 0, nil)
			if msg.Reply != "" {
				res, _ := json.Marshal(map[string]any{"status": "noop", "note": "range check không phát hiện missing"})
				msg.Respond(res)
			}
			return
		}

		if h.healThresholdBlocked(msg, op, table, len(missing), int64(srcTotal), int64(len(missing))) {
			return
		}

		written, dispatchErr := h.FetchAndWriteByIDs(ctx, entry, missing)
		if dispatchErr != nil {
			h.logActivity(op, table, "error", int64(written), dispatchErr)
			h.respondErr(msg, dispatchErr)
			return
		}

		h.logActivity(op, table, "healed", int64(written), nil)
		if msg.Reply != "" {
			res, _ := json.Marshal(map[string]any{
				"status":        "healed",
				"segment":       "source_shadow",
				"healed_count":  written,
				"missing_count": len(missing),
				"src_total":     srcTotal,
				"dispatch_path": "direct_fetch_write",
			})
			msg.Respond(res)
		}
		return
	}

	// 🔸 NHÁNH A: Chạy chế độ Window
	var runCtx context.Context
	if lookback == "hot" {
		runCtx = context.WithValue(ctx, "manual_lookback", true)
		h.logger.Info("[heal-a] starting fresh RunTier2 scan (Hot Mode lookback=2h, no cached report)",
			zap.String("table", table),
		)
	} else {
		runCtx = context.WithValue(ctx, "cold_lookback", true)
		runCtx = context.WithValue(runCtx, "manual_lookback", true)
		h.logger.Info("[heal-a] starting fresh RunTier2 scan (Cold Lookback=7d, no cached report)",
			zap.String("table", table),
		)
	}
	newReport := h.reconCore.RunTier2(runCtx, *entry)

	if newReport == nil {
		return
	}

	report := *newReport
	// ... (Gom missingIDs và staleObj)
	healIDs := append(append(append([]string{}, missingIDs...), staleObj.Mismatched...), staleObj.MissingFromSrc...)

	if h.healThresholdBlocked(msg, op, table, len(healIDs), srcCount, report.Diff) {
		return
	}

	// logic thực thi direct write FetchAndWriteByIDs...
}

---

### 3. Giải quyết lỗi "kẹp upper lùi về quá khứ" khi chạy check/heal thủ công

#### 1. `internal/handler/recon/recon_handler_run.go`
Khi gọi `RunTier2`, luôn gán `"manual_lookback" = true` vào context:

```go
func (h *ReconHandler) HandleReconCheck(msg *nats.Msg) {
	// ...
	var report *recon.ReconciliationReport
	switch payload.Tier {
	case "2":
		tier2Ctx := context.WithValue(ctx, "manual_lookback", true)
		if payload.Lookback == "cold" {
			tier2Ctx = context.WithValue(tier2Ctx, "cold_lookback", true)
		}
		report = h.reconCore.RunTier2(tier2Ctx, *entry)
    // ...
```

#### 2. `internal/handler/recon/recon_heal_v4.go`
(Đã cập nhật ở mục 4 phía trên)

#### 3. `internal/service/recon/recon_tier_a.go`
Bỏ qua việc kẹp lùi `upper` về `srcMax` hay `dstMax` quá khứ trong `pickScanRangeWithLag` nếu `manual_lookback` được kích hoạt:

```go
func (rc *ReconCore) pickScanRangeWithLag(ctx context.Context, entry source.TableRegistry, srcMax, dstMax time.Time, ingestLagMs int64) (time.Time, time.Time) {
	nowFreeze := time.Now().UTC().Add(-rc.adaptiveFreeze(ingestLagMs))
	upper := nowFreeze
	
	isManualLookback := false
	if ctx != nil {
		if val, ok := ctx.Value("manual_lookback").(bool); ok && val {
			isManualLookback = true
		}
	}
	if !isManualLookback {
		if !srcMax.IsZero() && srcMax.Before(upper) {
			upper = srcMax.Add(time.Millisecond)
		}
		if !dstMax.IsZero() && dstMax.Before(upper) {
			upper = dstMax.Add(time.Millisecond)
		}
	}

	lower := upper.Add(-rc.effectiveLookback(ctx))
    // ...
}
```

---

### 4. Mở rộng NATS Command Payload ở API Gateway (`cdc-cms-service`)

#### 1. `internal/app/commands/recon/recon_check.go`
Mở rộng `ReconCheckCommand` struct nhận thêm `Lookback`:
```diff
@@ -27,4 +27,5 @@ type ReconCheckCommand struct {
 	ports.AsyncCommandMixin
 	Tier  string `json:"tier"`
 	Table string `json:"table"`
+	Lookback string `json:"lookback,omitempty"`
 }
```

#### 2. `internal/app/commands/recon/recon_async.go`
Mở rộng `ReconHealCommand` struct nhận thêm `Mode`, `StartTime`, `EndTime` và `Lookback`:
```diff
@@ -18,6 +18,10 @@ type ReconHealCommand struct {
 	ports.AsyncCommandMixin
 	Table string `json:"table"`
 	// Recon V4: ""/"source_shadow" = heal A (dbz signal); "shadow_master" = heal B (re-transmute).
...
	Segment string `json:"segment,omitempty"`
+	Mode      string `json:"mode,omitempty"`
+	StartTime string `json:"start_time,omitempty"`
+	EndTime   string `json:"end_time,omitempty"`
+	Lookback  string `json:"lookback,omitempty"`
}

### 5. Cấu Hình Unmarshal & Truyền Tham Số Tại API Handlers (`cdc-cms-service`)

#### 1. `internal/api/recon/reconciliation_handler_commands.go`
Sửa đổi `TriggerCheck` và `TriggerCheckAll` để unmarshal và truyền `Lookback`:
```diff
@@ -18,3 +18,8 @@ func (h *ReconciliationHandler) TriggerCheck(c *fiber.Ctx) error {
 	tier := c.Query("tier", "1")
 	table := strings.TrimSpace(c.Params("table"))
+	var req struct {
+		Lookback string `json:"lookback"`
+	}
+	_ = c.BodyParser(&req)
+
 	if table == "" {
@@ -37,3 +42,3 @@ func (h *ReconciliationHandler) TriggerCheck(c *fiber.Ctx) error {
 
-	cmd := reconCmd.ReconCheckCommand{Tier: tier, Table: table}
+	cmd := reconCmd.ReconCheckCommand{Tier: tier, Table: table, Lookback: req.Lookback}
 	user := middleware.GetUsername(c)
@@ -63,4 +68,8 @@ func (h *ReconciliationHandler) TriggerCheck(c *fiber.Ctx) error {
 func (h *ReconciliationHandler) TriggerCheckAll(c *fiber.Ctx) error {
-	var scope reconScopeRequest
-	_ = c.BodyParser(&scope)
+	var req struct {
+		reconScopeRequest
+		Lookback string `json:"lookback"`
+	}
+	_ = c.BodyParser(&req)
+	scope := req.reconScopeRequest
 	tier := c.Query("tier", "1")
@@ -71,3 +80,3 @@ func (h *ReconciliationHandler) TriggerCheckAll(c *fiber.Ctx) error {
 	if err == nil && table != "" {
-		if _, derr := h.bus.Dispatch(ctx, reconCmd.ReconCheckCommand{Tier: tier, Table: table}); derr != nil {
+		if _, derr := h.bus.Dispatch(ctx, reconCmd.ReconCheckCommand{Tier: tier, Table: table, Lookback: req.Lookback}); derr != nil {
 			return c.Status(500).JSON(fiber.Map{"error": derr.Error()})
```

#### 2. `internal/api/recon/reconciliation_handler_heal.go`
Sửa đổi `TriggerHeal` để unmarshal và truyền `Mode`, `StartTime`, `EndTime`, `Lookback`:
```diff
@@ -35,5 +35,9 @@ func (h *ReconciliationHandler) TriggerHeal(c *fiber.Ctx) error {
 	}
 
-	var segReq struct {
-		Segment string `json:"segment"`
-	}
-	_ = c.BodyParser(&segReq)
+	var req struct {
+		Segment   string `json:"segment"`
+		Mode      string `json:"mode"`
+		StartTime string `json:"start_time"`
+		EndTime   string `json:"end_time"`
+		Lookback  string `json:"lookback"`
+	}
+	_ = c.BodyParser(&req)
 
 	user := middleware.GetUsername(c)
 	ctx := messaging.WithMetadata(c.UserContext(), user, c.Get("X-Correlation-Id"), c.Get("Idempotency-Key"))
-	res, derr := h.bus.Dispatch(ctx, reconCmd.ReconHealCommand{Table: table, Segment: segReq.Segment})
+	res, derr := h.bus.Dispatch(ctx, reconCmd.ReconHealCommand{
+		Table:     table,
+		Segment:   req.Segment,
+		Mode:      req.Mode,
+		StartTime: req.StartTime,
+		EndTime:   req.EndTime,
+		Lookback:  req.Lookback,
+	})
```

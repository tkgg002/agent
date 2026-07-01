# Hồ Sơ Giải Pháp Kỹ Thuật: Cải Tiến Luồng Heal & Định Tuyến Theo Chế Độ Chọn (Window vs Full-Diff)

Hồ sơ này mô tả chi tiết các thay đổi mã nguồn cần thiết cho cả Frontend (FE) và Backend (BE) để thực hiện định tuyến heal theo tham số từ UI và thay thế cơ chế Debezium NATS signal bằng direct write.

---

## 1. Frontend (FE) - Modal Kích Hoạt Heal Trên CMS-Web

### Component JSX Mock Diff (React)
Thêm các controls radio buttons và DatePickers cho modal trigger heal:

```jsx
// src/components/ReconHealModal.jsx
import React, { useState, useEffect } from 'react';

export function ReconHealModal({ table, segment, onClose, onSubmit }) {
  const [mode, setMode] = useState('window'); // 'window' hoặc 'full_diff'
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
      start_time: new Date(startTime).toISOString(),
      end_time: new Date(endTime).toISOString(),
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
          Chế độ Window (7 ngày gần nhất, an toàn)
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

      <div className="time-pickers">
        <div>
          <label>Từ thời gian:</label>
          <input
            type="datetime-local"
            value={startTime}
            disabled={mode === 'window'}
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
            disabled={mode === 'window'}
            onChange={(e) => {
              setEndTime(e.target.value);
              handleTimeChange(startTime, e.target.value);
            }}
          />
        </div>
      </div>

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
Cập nhật mutation để chấp nhận thêm `mode`, `startTime` và `endTime`:
```diff
@@ -190,5 +190,5 @@
 export function useHealMutation() {
-  return useMutation<void, Error, { table: string; reason: string; segment?: string; sourceDatabase?: string; sourceTable?: string; shadowSchema?: string; shadowTable?: string }>({
-    mutationFn: async ({ table, reason, segment, sourceDatabase, sourceTable, shadowSchema, shadowTable }) => {
+  return useMutation<void, Error, { table: string; reason: string; segment?: string; sourceDatabase?: string; sourceTable?: string; shadowSchema?: string; shadowTable?: string; mode?: string; startTime?: string; endTime?: string }>({
+    mutationFn: async ({ table, reason, segment, sourceDatabase, sourceTable, shadowSchema, shadowTable, mode, startTime, endTime }) => {
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
         },
         { headers: auditHeaders(reason) },
       );
```

#### 2. [MODIFY] [ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx)
Thêm UI Controls đặc thù cho Heal (radio button chọn Window/Full-diff, inputs và validation 30 ngày):
```diff
@@ -10,9 +10,11 @@
   actionLabel: string;
   danger?: boolean;
+  isHeal?: boolean;
-  onConfirm: (reason: string) => Promise<void> | void;
+  onConfirm: (reason: string, mode?: string, startTime?: string, endTime?: string) => Promise<void> | void;
   onCancel: () => void;
   loading?: boolean;
 }
 
 export default function ConfirmDestructiveModal({
@@ -28,4 +30,5 @@
   danger = false,
+  isHeal = false,
   onConfirm,
   onCancel,
   loading = false,
@@ -33,4 +36,9 @@
   const [reason, setReason] = useState('');
   const [submitting, setSubmitting] = useState(false);
+  const [mode, setMode] = useState('window'); // 'window' hoặc 'full_diff'
+  const [startTime, setStartTime] = useState('');
+  const [endTime, setEndTime] = useState('');
+  const [timeError, setTimeError] = useState('');
 
   useEffect(() => {
@@ -39,4 +47,19 @@
       setReason('');
       setSubmitting(false);
+      setMode('window');
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
+      setTimeError('Vui lòng chọn đầy đủ thời gian!');
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
+      await onConfirm(
+        trimmed,
+        isHeal ? mode : undefined,
+        isHeal ? new Date(startTime).toISOString() : undefined,
+        isHeal ? new Date(endTime).toISOString() : undefined
+      );
     } finally {
       setSubmitting(false);
     }
   };
@@ -76,5 +98,5 @@
       okButtonProps={{
         danger,
-        disabled: !isReasonValid || isBusy,
+        disabled: !isFormValid || isBusy,
         loading: isBusy,
       }}
@@ -87,4 +109,47 @@
       <Paragraph>{description}</Paragraph>
 
+      {isHeal && (
+        <div style={{ marginBottom: 16, padding: 12, background: '#f5f5f5', borderRadius: 8 }}>
+          <div style={{ marginBottom: 8 }}>
+            <Text strong>Chế độ quét & chữa lành:</Text>
+          </div>
+          <div style={{ marginBottom: 12 }}>
+            <label style={{ marginRight: 16, cursor: 'pointer' }}>
+              <input
+                type="radio"
+                name="heal_mode"
+                value="window"
+                checked={mode === 'window'}
+                onChange={() => {
+                  setMode('window');
+                  setTimeError('');
+                }}
+              />{' '}
+              Window (Cửa sổ 7 ngày gần nhất)
+            </label>
+            <label style={{ cursor: 'pointer' }}>
+              <input
+                type="radio"
+                name="heal_mode"
+                value="full_diff"
+                checked={mode === 'full_diff'}
+                onChange={() => {
+                  setMode('full_diff');
+                  handleTimeChange(startTime, endTime);
+                }}
+              />{' '}
+              Full-diff (Quét theo thời gian, tối đa 30 ngày)
+            </label>
+          </div>
+          
+          <div style={{ display: 'flex', gap: 12 }}>
+            <div style={{ flex: 1 }}>
+              <div style={{ fontSize: 12, color: '#666' }}>Từ thời gian:</div>
+              <Input
+                type="datetime-local"
+                value={startTime}
+                disabled={mode === 'window'}
+                onChange={(e) => {
+                  setStartTime(e.target.value);
+                  handleTimeChange(e.target.value, endTime);
+                }}
+              />
+            </div>
+            <div style={{ flex: 1 }}>
+              <div style={{ fontSize: 12, color: '#666' }}>Đến thời gian:</div>
+              <Input
+                type="datetime-local"
+                value={endTime}
+                disabled={mode === 'window'}
+                onChange={(e) => {
+                  setEndTime(e.target.value);
+                  handleTimeChange(startTime, e.target.value);
+                }}
+              />
+            </div>
+          </div>
+          {mode === 'full_diff' && timeError && (
+            <div style={{ color: '#ff4d4f', fontSize: 12, marginTop: 8 }}>
+              {timeError}
+            </div>
+          )}
+        </div>
+      )}
+
       <Paragraph>
```

#### 3. [MODIFY] [DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx)
Khai báo prop `isHeal` khi render modal và gửi params sang mutation:
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
+  const handleConfirm = async (reason: string, mode?: string, startTime?: string, endTime?: string) => {
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
@@ -204,9 +204,12 @@
 
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
 	}
 	json.Unmarshal(msg.Data, &payload)
 
@@ -225,3 +228,3 @@
 		if payload.Segment == "shadow_master" {
 			h.healSegmentB(ctx, msg, payload.Table)
 		} else {
-			h.healSegmentA(ctx, msg, payload.Table)
+			h.healSegmentA(ctx, msg, payload.Table, payload.Mode, payload.StartTime, payload.EndTime)
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

			// Lọc theo khoảng thời gian
			filter := bson.M{
				timestampField: bson.M{
					"$gte": startTime,
					"$lt":  endTime,
				},
			}
			if lastID != nil {
				filter["_id"] = bson.M{"$gt": lastID}
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

	tsCol := rc.resolveSourceTSField(ctx, entry)

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
	idChan, errChan := rc.sourceAgent.StreamIDsInTimeRange(ctx, entry.SourceURL, entry.SourceDB, entry.SourceTable, entry.TimestampField, startTime, endTime)
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
Thay đổi chữ ký hàm `healSegmentA`, validate khoảng thời gian và phân luồng routing theo mode chọn:

```diff
@@ -273,4 +273,4 @@
-func (h *ReconHandler) healSegmentA(ctx context.Context, msg *nats.Msg, table string) {
+func (h *ReconHandler) healSegmentA(ctx context.Context, msg *nats.Msg, table string, mode string, startTimeStr string, endTimeStr string) {
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
+	// 🔸 NHÁNH A: Chạy chế độ Window mặc định
+	coldCtx := context.WithValue(ctx, "cold_lookback", true)
+	newReport := h.reconCore.RunTier2(coldCtx, *entry)
 
 	if newReport == nil {
@@ -340,140 +392,20 @@
 	if newReport.MissingCount == 0 && newReport.StaleCount == 0 && newReport.OrphanCount == 0 {
-		h.logger.Info("[heal-a] RunTier2 found no drift — running full ID diff safety net (catches records outside time window)",
-			zap.String("table", table),
-		)
-
-		// Safety net: RunTier2 is time-windowed (cold_lookback=7d).
-		// Records deleted from shadow with old timestamp are invisible to hash_window.
-		// Full ID diff catches ALL missing records regardless of timestamp.
-		fullMissing, srcTotal, fullErr := h.reconCore.FullIDDiffMissingFromShadow(ctx, *entry)
-		if fullErr != nil {
-			h.logger.Warn("[heal-a] full ID diff failed — treating as noop",
-				zap.String("table", table),
-				zap.Error(fullErr),
-			)
-			h.logActivity(op, table, "noop", 0, nil)
-			if msg.Reply != "" {
-				res, _ := json.Marshal(map[string]any{
-					"status":     "noop",
-					"note":       "RunTier2 clean, full_id_diff error: " + fullErr.Error(),
-					"checked_at": newReport.CheckedAt,
-				})
-				msg.Respond(res)
-			}
-			return
-		}
-
-		if len(fullMissing) == 0 {
-			h.logger.Info("[heal-a] full ID diff confirms clean — genuine noop",
-				zap.String("table", table),
-				zap.Int("src_total", srcTotal),
-			)
-			h.logActivity(op, table, "noop", 0, nil)
-			if msg.Reply != "" {
-				res, _ := json.Marshal(map[string]any{
-					"status":    "noop",
-					"note":      "RunTier2 + full_id_diff đều sạch — data genuinely clean",
-					"src_total": srcTotal,
-					"checked_at": newReport.CheckedAt,
-				})
-				msg.Respond(res)
-			}
-			return
-		}
-
-		// Found IDs missing from shadow but outside time window — heal them directly.
-		h.logger.Info("[heal-a] full ID diff found records missing from shadow (outside time window)",
-			zap.String("table", table),
-			zap.Int("missing_count", len(fullMissing)),
-			zap.Int("src_total", srcTotal),
-		)
-
-		// Reuse heal dispatch path below by injecting into missingIDs.
-		// Build a synthetic report-like path via the existing dispatch code.
-		if h.healThresholdBlocked(msg, op, table, len(fullMissing), int64(srcTotal), int64(len(fullMissing))) {
-			return
-		}
-
-		var written int
-		var dispatchErr error
-
-		if h.eventHandler != nil {
-			// Direct path: fetch từ MongoDB → upsert vào shadow.
-			// KHÔNG dùng Debezium signal vì connector có thể không có
-			// signal.data.collection configured.
-			h.logger.Info("[heal-a] dispatching via FetchAndWriteByIDs (direct MongoDB→shadow)",
-				zap.String("table", table),
-				zap.Int("missing_count", len(fullMissing)),
-			)
-			written, dispatchErr = h.FetchAndWriteByIDs(ctx, entry, fullMissing)
-			if dispatchErr != nil {
-				h.logActivity(op, table, "error", int64(written), dispatchErr)
-				h.respondErr(msg, fmt.Errorf("FetchAndWriteByIDs: %w", dispatchErr))
-				return
-			}
-		} else {
-			// Fallback: publish Debezium signal (legacy path).
-			// Chỉ hoạt động nếu connector có signal.data.collection hoặc
-			// signal.kafka.topic configured.
-			h.logger.Warn("[heal-a] eventHandler not wired — falling back to Debezium signal (may be no-op if connector not configured)",
-				zap.String("table", table),
-			)
-			dispatched := 0
-			for start := 0; start < len(fullMissing); start += healChunkSize {
-				if start > 0 {
-					time.Sleep(healDelayMs)
-				}
-				// ...
-			}
-		}
+		h.logActivity(op, table, "noop", 0, nil)
+		if msg.Reply != "" {
+			res, _ := json.Marshal(map[string]any{"status": "noop", "note": "RunTier2 window clean"})
+			msg.Respond(res)
+		}
+		return
 	}
 
 	report := *newReport
 	// ... (Gom missingIDs và staleObj)
 	healIDs := append(append(append([]string{}, missingIDs...), staleObj.Mismatched...), staleObj.MissingFromSrc...)
 
 	if h.healThresholdBlocked(msg, op, table, len(healIDs), srcCount, report.Diff) {
 		return
 	}
 
-	// [MODIFIED]: Thay thế hoàn toàn việc bắn Debezium NATS signal
-	// Gọi trực tiếp FetchAndWriteByIDs để thực hiện direct write từ Mongo sang Shadow
-	h.logger.Info("[heal-a] Window drift detected! Dispatching via FetchAndWriteByIDs (direct write path)",
-		zap.String("table", table), zap.Int("ids", len(healIDs)))
-
-	written, dispatchErr := h.FetchAndWriteByIDs(ctx, entry, healIDs)
-	if dispatchErr != nil {
-		h.logActivity(op, table, "error", int64(written), dispatchErr)
-		h.respondErr(msg, dispatchErr)
-		return
-	}
-
-	now := time.Now().UTC()
-	healedDurationMs := int(time.Since(startTime).Milliseconds())
-	_ = h.reportRepo.UpdateByID(ctx, report.ID, map[string]any{
-		"healed_at":          now,
-		"healed_count":       written,
-		"healed_duration_ms": healedDurationMs,
-		"status":             "healed",
-	})
-
-	h.logActivity(op, table, "healed", int64(written), nil)
-	if msg.Reply != "" {
-		res, _ := json.Marshal(map[string]any{
-			"status":             "healed",
-			"segment":            "source_shadow",
-			"healed_count":       written,
-			"healed_at":          now,
-			"healed_duration_ms": healedDurationMs,
-			"checked_at":         report.CheckedAt,
-			"missing_count":      len(missingIDs),
-			"mismatched_count":   len(staleObj.Mismatched),
-			"orphan_count":       len(staleObj.MissingFromSrc),
-			"note":               "chữa lành trực tiếp từ MongoDB sang Shadow DB thành công",
-		})
-		msg.Respond(res)
-	}
+	// logic thực thi direct write FetchAndWriteByIDs...
 }
 ```

## 3. Tích Hợp Tracing (OpenTelemetry)

Để tăng khả năng quan sát (observability), OpenTelemetry parent và child spans được tích hợp vào các điểm nút chính trong luồng đối soát và heal.

### 1. [MODIFY] [recon_stream.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_stream.go)
Bổ sung child span cho các goroutine chạy Find/Select batch:
```diff
@@ -9,4 +9,5 @@
 	"time"
 
 	"centralized-data-service/internal/service/source"
+	"centralized-data-service/pkgs/observability"
 	"github.com/kamva/mgm/v3"
@@ -107,3 +108,6 @@
 
 			opts := options.Find().
 				SetProjection(bson.M{"_id": 1}).
 				SetSort(bson.D{{Key: "_id", Value: 1}}).
 				SetLimit(batchSize)
 
+			_, batchSpan := observability.ChildSpan(ctx, "mongo.find_batch", attribute.String("collection", collection))
 			result, err := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
 				return coll.Find(ctx, filter, opts)
 			})
+			observability.EndSpan(batchSpan, &err)
 			if err != nil {
@@ -213,3 +217,6 @@
 
 			result, err := sa.getBreaker(sourceURL).Execute(func() (interface{}, error) {
+				_, batchSpan := observability.ChildSpan(ctx, "pg.select_batch", attribute.String("table", tableName))
+				defer observability.EndSpan(batchSpan, nil)
 				rows, err := db.WithContext(ctx).Raw(querySql, args...).Rows()
 				if err != nil {
```

### 2. [MODIFY] [recon_tier_a.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service/recon/recon_tier_a.go)
Bổ sung parent và child spans cho đối soát range:
```diff
@@ -11,4 +11,6 @@
 	"centralized-data-service/internal/model/recon"
 	"centralized-data-service/internal/service/source"
+	"centralized-data-service/pkgs/observability"
+	"go.opentelemetry.io/otel/attribute"
 	"go.uber.org/zap"
@@ -620,3 +622,7 @@
 	if rc.shadowPlane == nil {
 		return nil, 0, fmt.Errorf("shadowPlane not wired")
 	}
+
+	ctx, span := observability.ChildSpan(ctx, "cdc.recon.time_bounded_diff", attribute.String("table", entry.QualifiedTarget()))
+	var finalErr error
+	defer func() { observability.EndSpan(span, &finalErr) }()
 
 	tsCol := rc.resolveSourceTSField(ctx, entry)
 
 	// Tải ID từ Postgres Shadow DB trong khoảng thời gian
 	var shadowIDs []string
+	ctxPg, spanPg := observability.ChildSpan(ctx, "pg.query.shadow_ids", attribute.String("table", entry.QualifiedTarget()))
 	if err := rc.shadowPlane.WithContext(ctxPg).Raw(
 		fmt.Sprintf(`SELECT "_source_id"::text FROM %s WHERE NOT "_deleted" AND "_source_id" IS NOT NULL AND %s >= ? AND %s < ?`,
 			quoteRelation(entry.QualifiedTarget()), quoteIdent(tsCol), quoteIdent(tsCol)),
 		startTime, endTime,
-	).Scan(&shadowIDs).Error; err != nil {
+	).Scan(&shadowIDs).Error; err != nil {
+		observability.EndSpan(spanPg, &err)
+		finalErr = err
 		return nil, 0, fmt.Errorf("shadow list ids in range: %w", err)
 	}
+	observability.EndSpan(spanPg, nil)
 
 	shadowSet := make(map[string]struct{}, len(shadowIDs))
 	for _, id := range shadowIDs {
 		shadowSet[id] = struct{}{}
 	}
 
 	// Stream ID từ Source MongoDB trong khoảng thời gian
+	ctxStream, spanStream := observability.ChildSpan(ctx, "mongo.stream.source_ids", attribute.String("table", entry.SourceTable))
 	idChan, errChan := rc.sourceAgent.StreamIDsInTimeRange(ctxStream, entry.SourceURL, entry.SourceDB, entry.SourceTable, entry.TimestampField, startTime, endTime)
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
+	observability.EndSpan(spanStream, &streamErr)
 
 	if streamErr != nil {
+		finalErr = streamErr
 		return nil, srcCount, fmt.Errorf("stream range partial error: %w", streamErr)
 	}
 
 	return missing, srcCount, nil
```

### 3. [MODIFY] [recon_heal_v4.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_heal_v4.go)
Tách các child spans cho các giai đoạn scan và write trong `healSegmentA`:
```diff
@@ -315,6 +315,9 @@
 		h.logger.Info("[heal-a] starting full-diff range comparison",
 			zap.String("table", table), zap.Time("start", start), zap.Time("end", end))
 
-		missing, srcTotal, err := h.reconCore.TimeBoundedDiffMissingFromShadow(ctx, *entry, start, end)
+		ctxDiff, spanDiff := observability.ChildSpan(ctx, "cdc.recon.heal.full_diff_scan")
+		missing, srcTotal, err := h.reconCore.TimeBoundedDiffMissingFromShadow(ctxDiff, *entry, start, end)
+		observability.EndSpan(spanDiff, &err)
 		if err != nil {
 			h.logActivity(op, table, "error", 0, err)
 			h.respondErr(msg, err)
@@ -335,6 +338,9 @@
 		}
 
-		written, dispatchErr := h.FetchAndWriteByIDs(ctx, entry, missing)
+		ctxWrite, spanWrite := observability.ChildSpan(ctx, "cdc.recon.heal.direct_write")
+		written, dispatchErr := h.FetchAndWriteByIDs(ctxWrite, entry, missing)
+		observability.EndSpan(spanWrite, &dispatchErr)
 		if dispatchErr != nil {
 			h.logActivity(op, table, "error", int64(written), dispatchErr)
```

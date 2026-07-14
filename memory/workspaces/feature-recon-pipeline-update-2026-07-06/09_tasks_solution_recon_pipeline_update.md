# Hồ sơ giải pháp kỹ thuật - Dọn dẹp Deep Check Payload và Sửa lỗi Segment UI Routing

Tài liệu này định nghĩa chi tiết mã nguồn sẽ thay đổi trên Frontend (`cdc-cms-web`), API Gateway (`cdc-cms-service`) và Core Engine (`centralized-data-service`).

---

## 1. Frontend (cdc-cms-web)

### [MODIFY] [ConfirmDestructiveModal.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/components/ConfirmDestructiveModal.tsx)

#### Thay đổi 1: Prop types & interface
```diff
export interface ConfirmDestructiveModalProps {
  open: boolean;
  title: string;
  description: string;
  targetName: string;
  actionLabel: string;
  danger?: boolean;
  isHeal?: boolean;
  isManualRecon?: boolean;
+  initialSegment?: string;
  onConfirm: (
    reason: string,
-    mode?: string,
-    startTime?: string,
-    endTime?: string,
-    lookback?: string,
-    segment?: string,
-    deep?: boolean,
-    startTimeMs?: number | null,
-    endTimeMs?: number | null
+    typeRecon: string,
+    lookback?: string,
+    segment?: string,
+    startTimeMs?: number | null,
+    endTimeMs?: number | null
  ) => Promise<void> | void;
  onCancel: () => void;
  loading?: boolean;
}
```

#### Thay đổi 2: Khởi tạo state & logic submit
```diff
export default function ConfirmDestructiveModal({
  open,
  title,
  description,
  targetName,
  actionLabel,
  danger = false,
  isHeal: _isHeal = false,
  isManualRecon = false,
+  initialSegment,
  onConfirm,
  onCancel,
  loading = false,
}: ConfirmDestructiveModalProps) {
  const [reason, setReason] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [checkMode, setCheckMode] = useState<'lookback' | 'full_diff' | 'deep'>('lookback');
  const [lookback, setLookback] = useState('cold'); // 'hot' hoặc 'cold'
  const [customRange, setCustomRange] = useState<[dayjs.Dayjs | null, dayjs.Dayjs | null] | null>(null);
-  const [segment, setSegment] = useState(''); // '' (cả 2), 'source_shadow', 'shadow_master'
+  const [segment, setSegment] = useState(initialSegment || ''); // '' (cả 2), 'source_shadow', 'shadow_master'

  const handleCheckModeChange = (mode: 'lookback' | 'full_diff' | 'deep') => {
    setCheckMode(mode);
    if (mode === 'full_diff' || mode === 'deep') {
      setCustomRange([dayjs().subtract(30, 'day'), dayjs()]);
    } else {
      setCustomRange(null);
    }
  };

  useEffect(() => {
    if (open) {
      setReason('');
      setSubmitting(false);
      setLookback('cold');
      setCheckMode('lookback');
      setCustomRange(null);
-      setSegment('');
+      setSegment(initialSegment || '');
    }
-  }, [open]);
+  }, [open, initialSegment]);

  const trimmed = reason.trim();
  const isReasonValid = trimmed.length >= MIN_REASON_LENGTH;
  
  const rangeDurationDays = checkMode !== 'lookback' && customRange && customRange[0] && customRange[1]
    ? customRange[1].diff(customRange[0], 'day', true)
    : 0;
  const isRangeTooLong = checkMode !== 'lookback' && rangeDurationDays > 30;
  const isCustomTimeValid = checkMode === 'lookback' || (customRange && customRange[0] && customRange[1] && !isRangeTooLong);

  const isFormValid = isReasonValid && isCustomTimeValid;
  const isBusy = loading || submitting;

  const handleOk = async () => {
    if (!isFormValid || isBusy) return;
    try {
      setSubmitting(true);
      if (isManualRecon) {
        const startMs = checkMode !== 'lookback' && customRange?.[0] ? customRange[0].valueOf() : null;
        const endMs = checkMode !== 'lookback' && customRange?.[1] ? customRange[1].valueOf() : null;
        const finalLookback = checkMode === 'lookback' ? lookback : '';
-        const deepValue = checkMode === 'deep';
+        let typeRecon = 'hash_window';
+        if (checkMode === 'deep') {
+          typeRecon = 'deep_check';
+        } else if (checkMode === 'full_diff') {
+          typeRecon = 'full_diff';
+        }

        await onConfirm(
          trimmed,
-          undefined,
-          undefined,
-          undefined,
+          typeRecon,
          finalLookback,
          segment,
-          deepValue,
          startMs,
          endMs
        );
      } else {
        await onConfirm(
          trimmed,
-          undefined,
-          undefined,
-          undefined,
-          undefined
+          'smoke'
        );
      }
    } finally {
      setSubmitting(false);
    }
  };
```

---

### [MODIFY] [DataIntegrity.tsx](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/pages/DataIntegrity.tsx)

#### Thay đổi 1: Callback handleConfirm
```diff
  const handleConfirm = async (
    reason: string,
-    _mode?: string,
-    _startTime?: string,
-    _endTime?: string,
+    typeRecon: string,
    lookback?: string,
    segment?: string,
-    deep?: boolean,
    startTimeMs?: number | null,
    endTimeMs?: number | null
  ) => {
    if (!modalPlan) return;
    const action = modalPlan.action;
    try {
      if (action.kind === 'check-all') {
        await checkAll.mutateAsync({ reason });
        message.success('Đã kích hoạt kiểm tra — kết quả sẽ cập nhật trong vài phút.');
        setTimeout(invalidateReports, 5000);
      } else if (action.kind === 'check-table') {
        const row = action.record || reportByTarget.get(action.table);
-        let typeRecon = 'hash_window';
-        if (deep) {
-          typeRecon = 'deep_check';
-        } else if (startTimeMs || endTimeMs) {
-          typeRecon = 'full_diff';
-        }

        await checkTable.mutateAsync({
          table: action.table,
          typeRecon,
          reason,
          sourceDatabase: row?.source_db,
          sourceTable: row?.source_table || undefined,
          shadowSchema: row?.shadow_schema || undefined,
          shadowTable: row?.shadow_table || undefined,
          lookback: lookback || undefined,
-          segment: segment || undefined,
-          deep: deep || undefined,
+          segment: segment || undefined,
          start_time: startTimeMs || undefined,
          end_time: endTimeMs || undefined,
        });
        message.success(`Đang kiểm tra ${typeRecon} cho ${action.table}…`);
        setTimeout(invalidateReports, 10000);
```

#### Thay đổi 2: Truyền prop initialSegment vào modal confirm
```diff
      {modalPlan && (
        <ConfirmDestructiveModal
          open={!!modalPlan}
          title={modalPlan.title}
          description={modalPlan.description}
          targetName={modalPlan.targetName}
          actionLabel={modalPlan.actionLabel}
          danger={modalPlan.danger}
          isHeal={modalPlan.action.kind === 'heal'}
          isManualRecon={modalPlan.action.kind === 'check-table' && modalPlan.action.isManualRecon}
+          initialSegment={modalPlan.action.record?.segment}
          loading={mutationPending}
          onConfirm={handleConfirm}
          onCancel={() => setModalPlan(null)}
        />
      )}
```

---

### [MODIFY] [useReconStatus.ts](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-web/src/hooks/useReconStatus.ts)

#### Thay đổi 1: Xóa deep khỏi useCheckTableMutation
```diff
export function useCheckTableMutation() {
  return useMutation<
    void,
    Error,
    {
      table: string;
      typeRecon: string;
      reason: string;
      sourceDatabase?: string;
      sourceTable?: string;
      shadowSchema?: string;
      shadowTable?: string;
      lookback?: string;
      segment?: string;
-      deep?: boolean;
      start_time?: number | null;
      end_time?: number | null;
    }
  >({
    mutationFn: async ({
      table,
      typeRecon,
      reason,
      sourceDatabase,
      sourceTable,
      shadowSchema,
      shadowTable,
      lookback,
      segment,
-      deep,
      start_time,
      end_time,
    }) => {
      await cmsApi.post(
        `/api/reconciliation/check?type_recon=${encodeURIComponent(typeRecon)}`,
        {
          reason,
          table,
          source_database: sourceDatabase || undefined,
          source_table: sourceTable || undefined,
          shadow_schema: shadowSchema || undefined,
          shadow_table: shadowTable || undefined,
          lookback: lookback || undefined,
          segment: segment || undefined,
-          deep: deep || undefined,
          start_time: start_time || undefined,
          end_time: end_time || undefined,
        },
        { headers: auditHeaders(reason) },
      );
    },
    retry: 0,
  });
}
```

---

## 2. API Gateway (cdc-cms-service)

### [MODIFY] [recon_check.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/app/commands/recon/recon_check.go)
```diff
type ReconCheckCommand struct {
	ports.AsyncCommandMixin
	TypeRecon string `json:"type_recon"`
	Table     string `json:"table"`
	Lookback  string `json:"lookback,omitempty"`
	Segment   string `json:"segment,omitempty"`
-	Deep      bool   `json:"deep,omitempty"`
	StartTime *int64 `json:"start_time,omitempty"`
	EndTime   *int64 `json:"end_time,omitempty"`
}
```

---

### [MODIFY] [reconciliation_handler_commands.go](file:///Users/trainguyen/Documents/work/data-hub/cdc-cms-service/internal/api/recon/reconciliation_handler_commands.go)

#### Thay đổi 1: TriggerCheck
```diff
	var req struct {
		Lookback  string `json:"lookback"`
		Segment   string `json:"segment"`
-		Deep      bool   `json:"deep"`
		StartTime *int64 `json:"start_time"`
		EndTime   *int64 `json:"end_time"`
	}
	_ = c.BodyParser(&req)
 
	cmd := reconCmd.ReconCheckCommand{
		TypeRecon: typeRecon,
		Table:     table,
		Lookback:  req.Lookback,
		Segment:   req.Segment,
-		Deep:      req.Deep,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}
```

#### Thay đổi 2: TriggerCheckAll
```diff
	var req struct {
		reconScopeRequest
		Lookback  string `json:"lookback"`
		Segment   string `json:"segment"`
-		Deep      bool   `json:"deep"`
		StartTime *int64 `json:"start_time"`
		EndTime   *int64 `json:"end_time"`
	}
	_ = c.BodyParser(&req)
	scope := req.reconScopeRequest
	typeRecon := c.Query("type_recon", "smoke")
	user := middleware.GetUsername(c)
	ctx := messaging.WithMetadata(c.UserContext(), user, c.Get("X-Correlation-Id"), c.Get("Idempotency-Key"))
 
	table, err := h.resolveTargetTable(c, scope)
	if err == nil && table != "" {
		cmd := reconCmd.ReconCheckCommand{
			TypeRecon: typeRecon,
			Table:     table,
			Lookback:  req.Lookback,
			Segment:   req.Segment,
-			Deep:      req.Deep,
			StartTime: req.StartTime,
			EndTime:   req.EndTime,
		}
```

---

## 3. Core Engine (centralized-data-service)

### [MODIFY] [recon_check_handler.go](file:///Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/handler/recon/recon_check_handler.go)

#### Thay đổi 1: Payload struct
```diff
type reconCheckPayload struct {
	TypeRecon string `json:"type_recon"`
	Table     string `json:"table"`
	Segment   string `json:"segment"`
-	Deep      bool   `json:"deep"`
	StartTime *int64 `json:"start_time"`
	EndTime   *int64 `json:"end_time"`
	Lookback  string `json:"lookback"`
}
```

#### Thay đổi 2: Span attributes & routing logic
```diff
	ctx, span = observability.ChildSpan(ctx, "cdc.recon.check",
		attribute.String("recon.type_recon", payload.TypeRecon),
		attribute.String("recon.table", payload.Table),
		attribute.String("recon.segment", payload.Segment),
-		attribute.Bool("recon.deep", payload.Deep),
	)
	defer observability.EndSpan(span, nil)
 
	// 4. Routing Logic
	if payload.Segment == SegmentShadowMaster {
-		isDeep := payload.TypeRecon == TypeReconDeepCheck || payload.Deep
+		isDeep := payload.TypeRecon == TypeReconDeepCheck
		h.handleReconCheckSegmentB(ctx, msg, payload.Table, isDeep)
		return
	}
```

#### Thay đổi 3: validateAndEnrichContext
```diff
func (h *CheckHandler) validateAndEnrichContext(ctx context.Context, payload *reconCheckPayload) (context.Context, error) {
	hasTimeRange := payload.StartTime != nil || payload.EndTime != nil
	hasLookback := payload.Lookback != ""
 
	if hasTimeRange && hasLookback {
		return ctx, fmt.Errorf("invalid_parameters: time range and lookback are mutually exclusive")
	}
 
-	if hasLookback && payload.Deep {
+	if hasLookback && payload.TypeRecon == TypeReconDeepCheck {
		return ctx, fmt.Errorf("invalid_parameters: lookback and deep are mutually exclusive")
	}
```

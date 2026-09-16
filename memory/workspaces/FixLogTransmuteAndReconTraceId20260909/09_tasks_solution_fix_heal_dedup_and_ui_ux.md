# 09 Tasks Solution: Phương án Kỹ thuật Sửa lỗi Nhân bản Trùng lặp ID Recon, Lệch Số lượng Heal & Cơ chế Dừng Khẩn cấp Recon Lớn

**Mã Workspace:** `FixLogTransmuteAndReconTraceId20260909`  
**Ngày cập nhật:** 2026-09-09 16:20:00  

---

## 1. Backend Go Engine: `centralized-data-service`

### File 1: `internal/service/recon/recon_stream_bucket_engine.go`

#### A. Thêm method `DeduplicateAll()` chuẩn hóa thứ tự xử lý
Khử trùng lặp `MissingFromDest`, `MissingFromSrc`, `Mismatched` trước khi chạy phantom drift để tránh phình to và duyệt lặp:
```go
// DeduplicateAll loại bỏ toàn bộ các ID trùng lặp nội bộ trong từng nhóm
// và reclassify các ID phantom drift giữa MissingFromDest và MissingFromSrc.
func (s *StaleIDsPayload) DeduplicateAll() {
	if s == nil {
		return
	}
	// 1. Làm sạch trùng lặp từng mảng trước
	s.MissingFromDest = deduplicateStringSlice(s.MissingFromDest)
	s.MissingFromSrc = deduplicateStringSlice(s.MissingFromSrc)
	s.Mismatched = deduplicateStringSlice(s.Mismatched)

	// 2. Chạy reclassify phantom drift
	s.deduplicatePhantomDrift()

	// 3. Khử trùng lặp lại Mismatched nếu có ID mới chuyển sang
	s.Mismatched = deduplicateStringSlice(s.Mismatched)
}

func deduplicateStringSlice(slice []string) []string {
	if len(slice) <= 1 {
		return slice
	}
	seen := make(map[string]struct{}, len(slice))
	out := make([]string, 0, len(slice))
	for _, item := range slice {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			out = append(out, item)
		}
	}
	return out
}
```

#### B. Kích hoạt trong `ExecuteSegment` và `executeSegmentB`
```go
	if res.StaleIDs != nil {
		res.StaleIDs.DeduplicateAll()
	}
```

#### C. Thêm cơ chế Kiểm tra Dừng khẩn cấp (Cancellation Check) trong vòng lặp ngày
Tại đầu vòng lặp chunk ngày của cả Segment A và Segment B:
```go
	jobID, _ := ctx.Value(ReconJobIDKey).(string)

	for curr := startTime.UTC(); curr.Before(endTime.UTC()); {
		// Kiểm tra tín hiệu huỷ từ context hoặc từ DB status
		if err := ctx.Err(); err != nil {
			e.logger.Warn("recon job context cancelled", zap.String("job_id", jobID), zap.Error(err))
			return res, err
		}
		if e.isJobCancelled(ctx, jobID) {
			e.logger.Info("recon job was cancelled by user, breaking execution early", zap.String("job_id", jobID))
			return res, fmt.Errorf("recon job cancelled by user")
		}

		next := curr.Add(24 * time.Hour)
        // ...
```

Helper `isJobCancelled`:
```go
func (e *ChunkStreamBucketEngine) isJobCancelled(ctx context.Context, jobID string) bool {
	if e.jobRepo == nil || jobID == "" {
		return false
	}
	job, err := e.jobRepo.GetByID(ctx, jobID)
	if err != nil || job == nil {
		return false
	}
	return job.Status == "CANCELLED"
}
```

---

### File 2: `internal/service/recon/recon_job_worker.go`

#### A. Kiểm tra Cancelled trước khi thực thi
Tại `HandleJobEvent` (sau Step 1):
```go
	// Kiểm tra nếu job đã bị huỷ trước khi worker bốc lên
	if job.Status == "CANCELLED" {
		w.logger.Info("recon job already cancelled, skipping execution", zap.String("job_id", job.JobID))
		return nil
	}
```

#### B. Cập nhật Status `CANCELLED` khi engine trả về lỗi huỷ
Tại Step 5:
```go
	errMsg := ""
	if engineErr != nil {
		if strings.Contains(strings.ToLower(engineErr.Error()), "cancelled") {
			status = "CANCELLED"
		} else {
			status = StatusFailed
		}
		errMsg = engineErr.Error()
		finalErr = engineErr
	}
```

---

### File 3: `internal/handler/recon/recon_execute_heal_handler.go`

#### A. Khóa trần biên (Capping) & Auto-Realign cho CẢ Segment A và Segment B
- **Segment A** (`executeHealSegA`):
```go
	// Chuẩn hóa lại tổng số lỗi cần heal theo số lượng unique IDs thực tế
	if len(staleA.Mismatched) > 0 && rpt.StaleCount > len(staleA.Mismatched) {
		rpt.StaleCount = len(staleA.Mismatched)
	}
	if len(missingIDs) > 0 && rpt.MissingCount > len(missingIDs) {
		rpt.MissingCount = len(missingIDs)
	}

	healed := 0

	if opts.HealMismatched && len(staleA.Mismatched) > 0 {
		start := time.Now()
		written := h.fetchAndWriteChunked(ctx, entry, staleA.Mismatched, "mismatched")
		rpt.HealedMismatchedCount = min(written, len(staleA.Mismatched))
		rpt.HealedMismatchedDurationMs = int(time.Since(start).Milliseconds())
		healed += rpt.HealedMismatchedCount
	}
	if opts.HealMissingDest && len(missingIDs) > 0 {
		start := time.Now()
		written := h.fetchAndWriteChunked(ctx, entry, missingIDs, "missing_dest")
		rpt.HealedMissingDestCount = min(written, len(missingIDs))
		rpt.HealedMissingDestDurationMs = int(time.Since(start).Milliseconds())
		healed += rpt.HealedMissingDestCount
	}
```

- **Segment B** (`executeHealSegB`):
```go
	// Chuẩn hóa lại tổng số lỗi cần heal theo số lượng unique IDs thực tế cho Segment B
	if len(staleB.Mismatched) > 0 && rpt.StaleCount > len(staleB.Mismatched) {
		rpt.StaleCount = len(staleB.Mismatched)
	}
	if len(missingGpayIDs) > 0 && rpt.MissingCount > len(missingGpayIDs) {
		rpt.MissingCount = len(missingGpayIDs)
	}
	if len(staleB.MissingFromSrc) > 0 && rpt.OrphanCount > len(staleB.MissingFromSrc) {
		rpt.OrphanCount = len(staleB.MissingFromSrc)
	}

	healed := 0
	if opts.HealMismatched && len(staleB.Mismatched) > 0 {
		processed, err := h.publishTransmuteChunked(...)
		if err == nil {
			rpt.HealedMismatchedCount = min(processed, len(staleB.Mismatched))
			healed += rpt.HealedMismatchedCount
		}
	}
	if opts.HealMissingDest && len(missingGpayIDs) > 0 {
		processed, err := h.publishTransmuteChunked(...)
		if err == nil {
			rpt.HealedMissingDestCount = min(processed, len(missingGpayIDs))
			healed += rpt.HealedMissingDestCount
		}
	}
```

#### B. Cập nhật `finalizeReport`
Ghi đè `missing_count`, `stale_count`, `orphan_count` đã chuẩn hóa vào database:
```go
	updates := map[string]any{
		"missing_count":                   rpt.MissingCount,
		"stale_count":                     rpt.StaleCount,
		"orphan_count":                    rpt.OrphanCount,
        // ...
	}
```

---

## 2. CMS Backend API: `cdc-cms-service`

### File 1: `internal/infra/persistence/recon/recon_read_repo_gorm.go`
Thêm hàm `CancelReconJob`:
```go
func (r *reconReadRepoGorm) CancelReconJob(ctx context.Context, jobID string) error {
	return r.db.WithContext(ctx).
		Table("cdc_system.recon_jobs").
		Where("job_id = ? AND status IN ?", jobID, []string{"PENDING", "RUNNING"}).
		Updates(map[string]interface{}{
			"status":        "CANCELLED",
			"error_message": "Đã hủy bởi người dùng qua CMS",
		}).Error
}
```

### File 2: `internal/api/recon/reconciliation_handler_reports.go` & `internal/router/router.go`
Thêm endpoint API:
```go
func (h *ReconciliationHandler) CancelActiveJob(c *fiber.Ctx) error {
	jobID := strings.TrimSpace(c.Params("id"))
	if jobID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "job_id is required"})
	}
	if err := h.reader.CancelReconJob(c.UserContext(), jobID); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "job cancelled", "job_id": jobID})
}
```
Route trong `router.go`:
```go
dual("POST", shared, "/reconciliation/jobs/:id/cancel", h.Recon.CancelActiveJob)
```

---

## 3. Frontend React: `cdc-cms-web`

### File 1: `src/hooks/useReconStatus.ts`
Thêm hook hủy job:
```ts
export function useCancelReconJobMutation() {
  const queryClient = useQueryClient();
  return useMutation<void, Error, { jobId: string }>({
    mutationFn: async ({ jobId }) => {
      await cmsApi.post(`/api/reconciliation/jobs/${encodeURIComponent(jobId)}/cancel`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['active-recon-jobs'] });
    },
    retry: 0,
  });
}
```

### File 2: `src/components/ReconPipelineGrid.tsx`
Thêm nút `[Dừng]` cạnh mỗi job trong danh sách `active_jobs`:
```tsx
const cancelJobMutation = useCancelReconJobMutation();

// Trong renderItem của List<ActiveReconJob>:
{(job.status === 'RUNNING' || job.status === 'PENDING') && (
  <Popconfirm
    title="Dừng tiến trình đối soát này?"
    description="Bạn có chắc chắn muốn dừng tiến trình? Hệ thống sẽ dừng ngay sau chunk ngày hiện tại và giải phóng worker."
    onConfirm={() => cancelJobMutation.mutate({ jobId: job.job_id })}
    okText="Dừng tiến trình"
    cancelText="Đóng"
    okButtonProps={{ danger: true, loading: cancelJobMutation.isPending }}
  >
    <Button size="small" danger icon={<StopOutlined />} style={{ fontSize: 11 }}>
      Dừng
    </Button>
  </Popconfirm>
)}
```

### File 3: `src/components/ExecuteHealModal.tsx`
Cập nhật render các cột "Thiếu", "Lệch", "Thừa" trong bảng `unhealedReportColumns`:
```tsx
    {
      title: 'Thiếu', dataIndex: 'missing_count', width: 110,
      render: (v: number, record: any) => {
        const healed = record.healed_missing_dest_count || 0;
        const remaining = Math.max(0, v - healed);
        if (healed > 0 && remaining === 0) {
          return <Tag color="success" style={{ margin: 0 }}>Đã xong ({healed}/{v})</Tag>;
        }
        if (healed > 0) {
          return (
            <Space direction="vertical" size={0}>
              <Text type="danger" style={{ fontWeight: 600 }}>{healed}/{v}</Text>
              <Text type="secondary" style={{ fontSize: 10 }}>còn lại {remaining}</Text>
            </Space>
          );
        }
        return v > 0 ? <Text type="danger" style={{ fontWeight: 600 }}>{v}</Text> : <Text type="secondary">0</Text>;
      }
    },
    {
      title: 'Lệch', dataIndex: 'stale_count', width: 110,
      render: (v: number, record: any) => {
        const healed = record.healed_mismatched_count || 0;
        const remaining = Math.max(0, v - healed);
        if (healed > 0 && remaining === 0) {
          return <Tag color="success" style={{ margin: 0 }}>Đã xong ({healed}/{v})</Tag>;
        }
        if (healed > 0) {
          return (
            <Space direction="vertical" size={0}>
              <Text type="warning" style={{ fontWeight: 600 }}>{healed}/{v}</Text>
              <Text type="secondary" style={{ fontSize: 10 }}>còn lại {remaining}</Text>
            </Space>
          );
        }
        return v > 0 ? <Text type="warning" style={{ fontWeight: 600 }}>{v}</Text> : <Text type="secondary">0</Text>;
      }
    },
    {
      title: 'Thừa', dataIndex: 'orphan_count', width: 110,
      render: (v: number, record: any) => {
        const healed = record.pruned_missing_src_count || 0;
        const remaining = Math.max(0, v - healed);
        if (healed > 0 && remaining === 0) {
          return <Tag color="success" style={{ margin: 0 }}>Đã xong ({healed}/{v})</Tag>;
        }
        if (healed > 0) {
          return (
            <Space direction="vertical" size={0}>
              <Text style={{ color: '#fa8c16', fontWeight: 600 }}>{healed}/{v}</Text>
              <Text type="secondary" style={{ fontSize: 10 }}>còn lại {remaining}</Text>
            </Space>
          );
        }
        return v > 0 ? <Text style={{ color: '#ff4d4f', fontWeight: 600 }}>{v}</Text> : <Text type="secondary">0</Text>;
      }
    },
```

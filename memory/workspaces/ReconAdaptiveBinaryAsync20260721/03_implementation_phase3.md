# 03 — Thiết Kế Kỹ Thuật Chi Tiết Phase 3

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Phase:** 3 — Control Plane API Integration  

---

## 1. Thiết Kế Single Adaptive Endpoint (`POST /api/reconciliation/check`)

```go
func (h *CheckHandler) HandleReconCheck(msg *nats.Msg) {
    // 1. Unmarshal payload
    var payload reconCheckPayload
    ...
    // 2. Resolve time range
    startTime, endTime := h.resolveTimeRange(payload)
    rangeDuration := endTime.Sub(startTime)

    // 3. ADAPTIVE BRANCHING
    if rangeDuration <= 2*time.Hour {
        // --- SYNC FAST-PATH (< 300ms) ---
        drifts, err := h.bisectionEngine.ExecuteDrillDown(ctx, payload.Table, startTime, endTime)
        ...
        h.RespondJSON(msg, map[string]interface{}{
            "status": "success",
            "mode":   "sync_fast_path",
            "drifts": drifts,
        })
        return
    }

    // --- ASYNC JOB PATH (< 50ms) ---
    jobID := fmt.Sprintf("job_%d", time.Now().UnixNano())
    job := &repository.ReconJob{
        JobID:       jobID,
        TargetTable: payload.Table,
        StartTime:   startTime,
        EndTime:     endTime,
        Status:      "PENDING",
    }
    _ = h.jobRepo.Create(ctx, job)
    _ = h.natsPublisher.Publish("cdc.event.recon.job_created", jobID)

    h.RespondJSON(msg, map[string]interface{}{
        "status":     "accepted",
        "mode":       "async_job",
        "job_id":     jobID,
        "status_url": fmt.Sprintf("/api/reconciliation/jobs/%s", jobID),
    })
}
```

---

## 2. Thiết Kế Polling Handler (`GET /api/reconciliation/jobs/:job_id`)

```go
func (h *JobHandler) HandleGetJobStatus(c *gin.Context) {
    jobID := c.Param("job_id")
    job, err := h.jobRepo.GetByID(c.Request.Context(), jobID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "job_not_found"})
        return
    }
    c.JSON(http.StatusOK, gin.H{
        "job_id":           job.JobID,
        "target_table":     job.TargetTable,
        "status":           job.Status,
        "progress_percent": job.ProgressPercent,
        "total_diff_count": job.TotalDiffCount,
        "checkpoint_ts":    job.CheckpointTS,
        "result_summary":   job.ResultSummary,
        "error_message":    job.ErrorMessage,
    })
}
```

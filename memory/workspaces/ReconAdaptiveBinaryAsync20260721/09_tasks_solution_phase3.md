# 09 — Hồ Sơ Giải Pháp Kỹ Thuật Chi Tiết Phase 3

> **Workspace:** `ReconAdaptiveBinaryAsync20260721`  
> **Phase:** 3 — Control Plane API Integration  

---

## 1. Phân Tích Luồng Xử Lý Của Single Adaptive Endpoint Pattern

```
                       POST /api/reconciliation/check
                                     │
                     Payload Validation & Range Calculation
                                     │
                        Is Range Duration <= 2 Hours?
                                     │
             ┌───────────────────────┴───────────────────────┐
             ▼                                               ▼
          [ YES ]                                         [ NO ]
             │                                               │
   Sync Fast-Path Execution                        Async Job Stateful Execution
             │                                               │
  BinaryDrillDownEngine (Sync)                     Insert ReconJob (PENDING)
             │                                               │
   Return HTTP 200 OK + Data                      Publish NATS Event (job_created)
   (Execution Time < 300ms)                                  │
                                                  Return HTTP 202 Accepted + JobID
                                                  (Execution Time < 50ms)
                                                             │
                                                             ▼
                                                   Client Polling Status
                                               GET /api/reconciliation/jobs/:id
```

---

## 2. DTO Specifications

### Sync Response (Range $\le$ 2h):
```json
{
  "status": "success",
  "mode": "sync_fast_path",
  "table": "payment_bills",
  "drifts": []
}
```

### Async Response (Range $>$ 2h):
```json
{
  "status": "accepted",
  "mode": "async_job",
  "job_id": "job_1771500000000",
  "status_url": "/api/reconciliation/jobs/job_1771500000000"
}
```

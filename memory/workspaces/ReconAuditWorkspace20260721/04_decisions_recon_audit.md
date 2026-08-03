# 04 — Quyết Định Kiến Trúc Audit (Architectural Decision Records - ADRs)

> **Workspace:** `ReconAuditWorkspace20260721`  

---

## ADR-AUDIT-01: Phê Duyệt Mô Hình Chunk-Based Stream-to-Bucket
- **Bối cảnh:** Mô hình cũ Merkle Tree đệ quy có nguy cơ OOM khi tải dữ liệu lớn lên RAM và gây I/O Read Amplification.
- **Quyết định:** Phê duyệt kiến trúc **Chunk-Based Stream-to-Bucket Engine** (Outer loop duyệt chunk 1 ngày + Inner loop 96 RAM buckets 15-phút).
- **Hệ quả:** Duy trì RAM hằng số $O(1)$, triệt tiêu rủi ro OOM và tối ưu hóa việc truy vấn dữ liệu từ PostgreSQL/MongoDB.

## ADR-AUDIT-02: Đồng Bộ NATS Async State Machine & CMS Report
- **Bối cảnh:** Các scan dữ liệu lớn (> 2h) cần chạy bất đồng bộ để tránh HTTP Request Timeout.
- **Quyết định:** Áp dụng luồng định tuyến tự động Single Adaptive Endpoint. Lưu trạng thái job vào `cdc_system.recon_jobs` và ghi kết quả đối soát vào `cdc_system.cdc_reconciliation_report`.
- **Hệ quả:** Cho phép giao diện CMS UI truy vấn trạng thái job qua REST API hoặc NATS một cách linh hoạt.

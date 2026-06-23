# Kế hoạch chi tiết: Chiến dịch Refactor các file lớn trong centralized-data-service

Chiến dịch này được chia thành các đợt (Phases) dựa trên nhóm chức năng (Domains) để quản lý rủi ro và kiểm thử hiệu quả.

## Danh sách 19 file lớn (> 500 LoC) cần refactor:

| STT | File Path | LoC | Nhóm chức năng (Domain) |
|-----|-----------|-----|-------------------------|
| 1 | `internal/handler/orchestration/snapshot_runner_handler.go` | 981 | Orchestration |
| 2 | `internal/service/master/transmuter.go` | 902 | Master & Shadow |
| 3 | `internal/service/shadow/schema_adapter.go` | 900 | Master & Shadow |
| 4 | `internal/service/recon/recon_heal.go` | 899 | Recon |
| 5 | `internal/service/source/metadata_registry_service.go` | 898 | Source |
| 6 | `internal/service/recon/provisioning_orchestrator.go` | 872 | Recon |
| 7 | `internal/handler/recon/recon_handler.go` | 843 | Recon |
| 8 | `internal/service/recon/recon_tier_a.go` | 803 | Recon |
| 9 | `internal/server/worker_server_init.go` | 789 | Server Init |
| 10 | `internal/handler/recon/scan_handler.go` | 765 | Recon |
| 11 | `internal/service/master/master_ddl_generator.go` | 741 | Master & Shadow |
| 12 | `internal/service/recon/recon_engine.go` | 730 | Recon |
| 13 | `internal/handler/shadow/schema_ddl_handler.go` | 625 | Master & Shadow |
| 14 | `internal/handler/source/discover_handler.go` | 617 | Source |
| 15 | `internal/service/governance/masking_service.go` | 615 | Governance |
| 16 | `internal/handler/shadow/batch_buffer.go` | 552 | Master & Shadow |
| 17 | `internal/service/governance/partition_dropper.go` | 530 | Governance |
| 18 | `internal/service/governance/backfill_source_ts.go` | 526 | Governance |
| 19 | `internal/handler/source/source_register.go` | 515 | Source |

---

## Kế hoạch triển khai theo đợt (Phases):

### Giai đoạn 1: Recon Module (Phần còn lại)
Tập trung vào 6 file lớn thuộc module đối soát dữ liệu (`recon`):
1.  `recon_heal.go` (899 dòng) -> **Mục tiêu đầu tiên của Phase 1**.
2.  `provisioning_orchestrator.go` (872 dòng)
3.  `recon_tier_a.go` (803 dòng)
4.  `recon_engine.go` (730 dòng)
5.  `recon_handler.go` (843 dòng)
6.  `scan_handler.go` (765 dòng)

### Giai đoạn 2: Master & Shadow Module
Tập trung vào các logic biến đổi cấu trúc, sinh DDL và buffer:
1.  `transmuter.go` (902 dòng)
2.  `schema_adapter.go` (900 dòng)
3.  `master_ddl_generator.go` (741 dòng)
4.  `schema_ddl_handler.go` (625 dòng)
5.  `batch_buffer.go` (552 dòng)

### Giai đoạn 3: Orchestration & Source Module
Tập trung vào snapshot runner và đăng ký metadata nguồn:
1.  `snapshot_runner_handler.go` (981 dòng)
2.  `metadata_registry_service.go` (898 dòng)
3.  `discover_handler.go` (617 dòng)
4.  `source_register.go` (515 dòng)

### Giai đoạn 4: Governance & Server Init
Tập trung vào dọn dẹp phân vùng, masking dữ liệu và khởi tạo hệ thống:
1.  `masking_service.go` (615 dòng)
2.  `worker_server_init.go` (789 dòng)
3.  `partition_dropper.go` (530 dòng)
4.  `backfill_source_ts.go` (526 dòng)

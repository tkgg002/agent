# Active Plans Registry

> **Maintained by**: Brain (Antigravity)
> **Last Updated**: 2026-03-16
> **Purpose**: Registry để Brain biết workspace nào đang active → load đúng context khi bắt đầu phiên mới. KHÔNG phải cơ chế agent communication.

| Workspace | Project | Status | Last Active |
|-----------|---------|--------|-------------|
| upgrade-core-system | Upgrade Core Brain/Muscle System | ✅ Done | 2026-02-25 |
| feature-refactor-2026 | GooPay Core Refactor 2026 | ✅ Done — sẵn sàng tiếp tục | 2026-02-25 |
| optimize-brain-muscle-models | Tối ưu hóa model cho Brain/Muscle | ✅ Done (V2 Quota & Multi-Muscle) | 2026-02-25 |
| compare-disbursement-export | So sánh logic DisbursementTicketExport | ⏸ Paused | 2026-02-25 |
| compare-disbursement-trans-his-export | So sánh logic DisbursementTransHisExport | ✅ Done | 2026-02-27 |
| feature-merchant-export-activation-info | Bổ sung thông tin kích hoạt Merchant Export | ✅ Done | 2026-03-12 |
| feature-id-expired-notification-log-export | Tạo IDExpiredNotificationLogExport type | ✅ Done | 2026-02-27 |
| feature-fee-configuration | Cấu hình phí dịch vụ (Fee Configuration) | ⏸ Paused | 2026-03-03 |
| feature-cdc-integration | CDC Integration (Hybrid Debezium + Airbyte) | 🟡 Active | 2026-04-06 |
| feature-export-driver-search | Driver Info & Approximate Search in Exports | ✅ Done | 2026-03-24 |
| upgrade-agent-infrastructure | Nâng cấp hạ tầng Agent v1.10.0 (Brain/Muscle) | ✅ Done | 2026-04-06 |
| feature-trans-his-collection-export | Export TransHis Collection | 🟡 Active | 2026-04-09 |


---

## Notes
- **Active** 🟡: Đang làm trong phiên hiện tại hoặc phiên gần nhất
- **Paused** ⏸: Tạm dừng, sẽ tiếp tục sau
- **Done** ✅: Hoàn thành, archived
- Khi bắt đầu phiên mới: Brain đọc bảng này → load workspace có status Active đầu tiên
|-----------|---------|--------|-------------|

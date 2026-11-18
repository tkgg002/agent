# 01. REQUIREMENTS — Rà soát Critical Issues trên Git Diff của 3 Repo CDC

tier: T0
workspace: audit-cdc-3repos-diff-2026-11-16
suffix: cdc_diff_audit
date: 2026-11-16

## 1. Yêu cầu từ User
- Rà soát `git diff` cục bộ trên 3 repo CDC (`centralized-data-service`, `cdc-cms-service`, `cdc-cms-web`) khi mất OpenVPN.
- **Cập nhật yêu cầu:** Sửa dứt điểm toàn bộ 4 vấn đề **CRITICAL** (và 1 vấn đề **HIGH** ở test) đã phát hiện trong diff của 3 repo CDC.

## 2. Ràng buộc Thực thi (Technical Constraints từ Lessons)
- **L004 (Rule-Flag Driven Metadata Column Detection):** Tránh các heuristic quá rộng khi phát hiện cột metadata hoặc lọc cột row-diff MongoDB.
- **L082 (JetStream FIFO & Poison Pill Guard):** Mọi luồng auto-resume job mồ côi phải có giới hạn số lần thử (`MaxOrphanAutoResumeAttempts`) để chống vòng lặp OOM vô tận.
- **L086 (Spec-Driven PK & Multi-Source Routing):** Đồng bộ chuẩn kiểu dữ liệu ClickHouse/PostgreSQL giữa Frontend và Backend.
- **L002 / L095 (Host-Bound Secret Merge):** Bảo toàn secret gắn chặt với host/port/username khi Update/Recover/Introspect.

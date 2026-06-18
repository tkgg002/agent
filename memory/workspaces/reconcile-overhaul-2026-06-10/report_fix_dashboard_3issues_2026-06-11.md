# report_fix_dashboard_3issues_2026-06-11.md — Fix connector/schema + time + counts

> **Agent**: Muscle:Claude-Opus-4.8 | 2026-06-11 | User: "thêm connector + master schema; Recon cuối lúc sai time; counts vẫn chưa đúng"

## 1. Ba root cause (verify trước khi sửa)
| Issue | Root cause |
|---|---|
| Counts vẫn sai | **CMS đang chạy binary build TRƯỚC khi model có `total_*`** → API không trả field → FE fallback window; + đa số bảng chưa có report mới sau migration 084 |
| Time sai | `checked_at` = `timestamp WITHOUT time zone` nhưng worker ghi `time.Now()` (local +07) → DB lưu 17:32 khi UTC là 10:33 → API marshal "Z" → FE cộng +7h lần nữa |
| Thiếu connector/schema | report không mang — cần enrich read-side |

## 2. Fix
1. **Enrich lineage (read-side, không đổi write)**: `listLatestPrimary` + LATERAL `master_binding.master_schema` (row Segment B) + JOIN `connection_registry` qua `source_object_registry.source_connection_id` → cột `source_connection_code`. Read model +2 fields. FE: tag connector (cyan) dưới source; master hiển thị FQN `schema.table`.
   - *Sự cố trong lúc làm*: lần đầu lấy `shadow_binding.source_connection_code` — **cột không tồn tại** (FE type là dữ liệu enriched). Bắt được nhờ **chạy SQL trên DB thật trước khi build**; sửa đúng đường `so → connection_registry`.
2. **Time**: 5× `CheckedAt` + 2× `healed_at` → `time.Now().UTC()`. **Data-fix 1 lần**: UPDATE 34 rows local-naive (đang "tương lai" so UTC) −7h — bắt buộc, nếu không rows cũ đè `DISTINCT ON … DESC` latest suốt 7 tiếng (đã chứng kiến thật).
3. **Counts**: rebuild + restart CMS; trigger full recon A+B. *Phát hiện vận hành*: trigger B ngay sau A bị **leader-lock** chặn (A giữ `recon:leader` suốt run) → phải chờ leader free.

## 3. Phát hiện thêm + xử lý
- Rows `check_type='count'` totals-NULL đè latest = `errorReport` Tier1 **SRC_TIMEOUT** (Mongo `source max ts` deadline) — vì **schedule reconcile đã được bật với interval 1 phút** (bật song song ngoài phiên này) → CheckAll (spread 5') chồng vòng, nghẹt Mongo định kỳ. Code đúng (lỗi thật phải hiện); chỉnh **interval 1' → 30'** (đúng default thiết kế). Đây là config vận hành có chủ đích + ghi nhận, không phải cheat để đẹp số.
- Working tree worker có **code mới của bên khác** (orphan-prune: `PruneAllOrphans`/`RunOrphanPrune`, tier="prune") — build chung PASS, không đụng.

## 4. Verify (SQL thật)
- **Segment B 6/6 totals + UTC**: aaa 170/170 · aaaa2 170/170 · b3 465/464 · export_jobs_mt 170/163 · mt_02 170/332 · wallet_capsets 11101/11101.
- **Segment A** `count_windowed` có totals (vd export_jobs_4: 456/465); connector ra thật (dev000/aaaaa/goopay-lc-ws), master_schema đủ 6 row B.
- `checked_at` mới khớp NOW-UTC; build worker+cms+FE PASS; worker p4e (PID 75596) + CMS p4d + FE 5173 RUNNING.

## 5. Files đã sửa (git)
worker: `recon_core.go` (5× UTC + totals đoạn trước), `recon_heal_v4.go` (2× UTC); cms: `recon_read_repo_gorm.go` (3 edits enrich), `recon_read_models.go` (+2), (model total_* từ đợt trước); web: `useReconStatus.ts` (+2 fields), `ReconPipelineGrid.tsx` (connector tag + master FQN). +2 data ops có ghi nhận: UPDATE 34 rows timezone, interval 30'.

## 6. Note trung thực
- Bảng nào Mongo timeout đúng lúc check → counts "—" + trạng thái Cảnh báo: **trung thực** (không đếm được thì không bịa số); tần suất sẽ giảm mạnh với interval 30'.
- Heal-A signal row-level vẫn pending staging (từ P2).

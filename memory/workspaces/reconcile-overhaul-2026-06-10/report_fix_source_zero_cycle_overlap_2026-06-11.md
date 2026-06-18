# report_fix_source_zero_cycle_overlap_2026-06-11.md — Source=0 tràn lan (đợt 2)

> Muscle:Claude-Opus-4.8 | 2026-06-11 | User: "Source (recs) không đúng cái gì cả" + activity error_count tăng 2→12

## 1. Root cause (KHÁC đợt trước — verify bằng số thật)
9/12 bảng A latest = error 23505 `recon_runs_one_running`, do **2 yếu tố cộng hưởng**:
1. **Interval bị chỉnh 30'→10'** (ngoài phiên này) trong khi 1 cycle 19 bảng × stagger mất **12-16'** → vòng N+1 due khi vòng N còn chạy → chồng vòng.
2. **Worker được dev restart liên tục** (PID 17863→28402, `instance_id` đổi mỗi lần) → run 'running' mồ côi **"tươi" <15'** → self-heal cũ (threshold 15') không dám cancel → 23505 mỗi vòng → error row đè latest → totals NULL → FE hiện 0.

## 2. Fix 3 tầng
| # | Fix | File |
|---|---|---|
| 1 | `beginRun` self-heal: threshold 15'→**5'** + **cancel theo `instance_id <> mình`** — an toàn tuyệt đối vì: vào được beginRun = mình ĐANG giữ advisory lock bảng đó = không instance nào khác đang chạy thật bảng này (advisory lock chết theo connection khi process cũ bị kill) | `recon_core.go` |
| 2 | **In-flight guard**: `reconCycleInFlight atomic.Bool` — tick due khi cycle trước chưa xong → SKIP + Warn + activity row. Interval ngắn (10') trở nên **vô hại** (cycle kế chạy ngay tick sau khi xong) — không ép đổi config 10' của bên khác | `worker_server.go` |
| 3 | Data-fix: cancel orphan running hiện tại | recon_runs |

## 3. Verify
- Build PASS.
- **Không kill process dev của bên kia** — code fix sẽ ăn từ lần restart kế của họ (họ restart liên tục); orphan đã dọn nên **Source hồi NGAY trên worker hiện hành**: `export_jobs` **168/170** ✅, `users` 323/326, `export_jobs_testid1` 456/456, `wallet_capsets*` 11111/11111 ✅. 4 bảng còn error-row cũ (events, ej_2/_4/_test) đã trigger re-check — cùng cơ chế.
- Lưu ý đọc số mới: `wallet_capsets` 11111 (tăng từ 11101 — source đang nhận data mới, realtime đuổi kịp), `export_jobs_testid` shadow=0 (binding mới chưa ingest — Cảnh báo đúng).

## 4. Ghi chú vận hành cho Boss
- Interval 10' giữ nguyên theo ý người chỉnh — với in-flight guard sẽ không còn chồng vòng; nếu muốn cycle ngắn thật sự thì giảm stagger spread (hiện 5'/cycle 19 bảng).
- Mỗi lần dev restart worker giữa vòng vẫn tạo orphan tươi ≤5' → tối đa 5' sau tự hồi (trước là vĩnh viễn → rồi 15').

# 01_requirements

## R1 — Dọn recon_runs treo + chặn tái diễn (root cause)
- R1.1: 7 dòng `recon_runs` `status='running'` mồ côi (từ instance_id worker cũ đã chết) phải được đóng (status='cancelled', finished_at, error_message rõ).
- R1.2: Worker restart giữa vòng recon KHÔNG được để lại run treo vĩnh viễn → cần reaper **proactive** (không chỉ reactive theo từng table như `beginRun` hiện tại).
- R1.3: Reaper phải dọn cả run của **bảng không còn active** (vd `wallet_capsets_1`, `export_jobs_2/5` đã rename/bỏ) — nơi cơ chế reactive hiện tại không vươn tới.
- R1.4: Không được nhầm-giết run đang chạy THẬT (giữ ngưỡng stale an toàn, tái dùng 15' của beginRun).
- R1.5 (DoD): sau fix, `SELECT count(*) FROM recon_runs WHERE status='running' AND started_at < now()-15min` = 0; không còn `tier1 beginRun failed ... 23505` trong log mới.

## R2 — Hết warn "cannot resolve source URI" cho default_master (đúng root cause)
- R2.1: Warn phát sinh vì loop resolve source-URI quét **mọi** connection kể cả role không phải source. `default_master` role=master, 0 nguồn bind → không có source-URI là ĐÚNG, warn là false-positive.
- R2.2: Fix = chỉ resolve+warn cho connection thực sự được `source_object_registry` tham chiếu; bỏ qua (im lặng/ debug) connection không có nguồn. KHÔNG fabricate DSN cho default_master (không có bằng chứng nó cần, master writes segment B vẫn chạy OK).
- R2.3: Nếu một connection có nguồn THẬT mà resolve fail → vẫn warn (giữ tín hiệu lỗi thật).
- R2.4 (DoD): log worker mới không còn warn default_master; warn chỉ xuất hiện cho connection có source bị lỗi thật (nếu có).

## Non-goals
- Không sửa cơ chế leader-election / scheduler interval.
- Không đụng luồng Source→Shadow ngoài 2 fix trên.
- Không xoá/đổi connection_registry rows (chỉ sửa logic enumerate trong code).

# 04_decisions

## D1 — Reaper theo TUỔI (age-based 15') thay vì theo instance_id
- Cân nhắc: (a) cancel run của instance_id ≠ hiện tại (dọn ngay khi restart); (b) cancel theo tuổi >15'.
- Chọn **(b) age-based 15'** (tái dùng ngưỡng beginRun) vì: an toàn với multi-worker (không giết run của peer mới start), đồng nhất semantics sẵn có, đủ để dọn 7 row cũ (đều > vài giờ). Run mồ côi <15' không chặn gì gấp (beginRun reactive + chu kỳ kế tiếp xử lý). Tránh giả định "1 worker" cứng.
- Chạy reaper ở **startup + mỗi chu kỳ reconcile** để vừa dọn ngay vừa tự lành định kỳ.

## D1-revised — Thêm startup reaper INSTANCE-BASED (sau khi quan sát realtime)
- Quan sát: hệ thống đang có 5 run 'running' là orphan **<15'** từ restart gần đây (instance `5a71a016` chết, `1db294c6` lên); row >15' cũ đã được beginRun reactive cancel. Dev rebuild worker LIÊN TỤC (p4d→p4g) → mỗi restart orphan run <15' → reaper age-based 15' có **độ trễ 15'**, trong đó table bị 23505 mỗi vòng.
- Quyết bổ sung: startup gọi `ReapOrphanRunsFromDeadInstances` (cancel `status='running' AND instance_id IS DISTINCT FROM current`) → dọn NGAY mọi orphan của instance cũ bất kể tuổi. Giữ `ReapStaleRuns` age-based cho **periodic** (bắt hung run của chính instance hiện tại, an toàn không phụ thuộc instance).
- An toàn multi-worker: reconcile chạy dưới Redis leader-election (1 reconciler) → row instance khác lúc ta startup = chắc chắn mồ côi. Kể cả không leader-election, xấu nhất là cosmetic (finishRun `WHERE id=?` của peer ghi đè lại trạng thái thật). `IS DISTINCT FROM` quét luôn row instance_id NULL (legacy).

## D2 — KHÔNG fabricate DSN cho default_master
- `default_master`: role=master, 0 source bind (`count=0`), host/port/db=∅, options_json=`{}`. Warn đến từ recon loop resolve source-URI cho MỌI connection.
- Bịa DSN cho 1 connection master không nguồn = "thay đổi config để fake kết quả" (vi phạm ràng buộc user) + sai bản chất (nó không phải source). Master writes (segment B) trong log vẫn chạy (`master_rows:11`) ⇒ DSN rỗng của default_master KHÔNG chặn master writes ⇒ không cần.
- Quyết: **fix logic** — chỉ resolve source-URI cho connection có nguồn tham chiếu. Đúng root cause, không đụng dữ liệu config.

## D3 — Dọn 7 row treo bằng chính reaper (không SQL tay)
- Để startup reaper (FIX 1) tự cancel 7 row → chứng minh fix hoạt động end-to-end (red→green), tránh "sửa DB tay" tách rời code. Nếu không restart được worker → fallback cancel tay + ghi rõ.

## D4 — Phạm vi
- Chỉ 2 fix; không đổi scheduler/leader-election/interval. Backup file trước sửa (Rule 18).

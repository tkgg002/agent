# 06 — Validation Evidence: `pentagi_integration`

## 1. Đối chiếu 8 Cổng Chất lượng (Definition of Done — Gates G1–G8)

| Gate | Hạng mục | Trạng thái | Bằng chứng vật lý |
|------|----------|------------|-------------------|
| **G1** | Truy vết Requirement (`R1–R6`) | ✅ PASS | Đối chiếu đầy đủ `R1–R6` trong `01_requirements_pentagi_integration.md` với 4 trụ cột đã triển khai. |
| **G2** | Reproduce / Test Verification | ✅ PASS | `test_29` (H4b S5->S3 Re-plan Lock), `test_30` (H6 Loop Escalation Counter), và `test_01..05` trong `test_pentagi_bridge.py` đều PASS. |
| **G3** | Test & Runtime thật | ✅ PASS | Chạy 30/30 tests (`hooks/agy/tests`) + 25/25 tests (`tooling/tests`) + chạy CLI `pentagi_bridge.py trigger` thực tế. |
| **G4** | Edge-case & Boundary Check | ✅ PASS | Kiểm tra chặn subdomain spoofing (`http://localhost.evil.com:8080/api`), chống overwrite file cũ (`FileExistsError`), cô lập `.loop` theo `<conv_id>.<u_idx>.loop`. |
| **G5** | Regression Test Suite | ✅ PASS | Toàn bộ 55/55 unit tests cũ và mới của `hooks` và `tooling` đều `OK` (0 regression). |
| **G6** | Output Correctness | ✅ PASS | Hồ sơ S1 sinh từ `pentagi_bridge.py intake` khớp 100% format `LOG_LINE_RE` của `verify_governance.py` và mask 100% secret/credential. |
| **G7** | Adversarial Audit & `DG3` | ✅ PASS | Đã lập `audit_report_pentagi_integration.md` (đủ 6 phần) và chạy `DG3` sau lần sửa cuối đạt `"gate": "PASS"` (`06_laya_summary_dg3_20261020T134233.json`). |
| **G8** | Bằng chứng vật lý | ✅ PASS | Lưu đầy đủ trong `06_validation_pentagi_integration.md`, `11_report_pentagi_integration.md`, `audit_report_pentagi_integration.md`, `05_progress_pentagi_integration.md`. |

---

## 2. Kết quả Chạy Test Suite & Runtime CLI Thực tế

### 2.1. Unit Tests Component `hooks` (`hooks/agy/tests`) & `tooling` (`tooling/tests`)
```text
$ python3 -m unittest discover -s /Users/trainguyen/Documents/work/agent/hooks/agy/tests -p 'test_*.py' -v
...
test_29_h4b_blocks_source_edit_after_s5_audit_until_s3_replan_and_new_approve ... ok
test_30_h6_loop_prevention_escalates_on_three_consecutive_failures_and_resets_on_pass ... ok
----------------------------------------------------------------------
Ran 30 tests in 0.871s
OK

$ python3 -m unittest discover -s /Users/trainguyen/Documents/work/agent/tooling/tests -p 'test_*.py' -v
...
test_01_mask_secrets_masks_tokens_passwords_and_db_uris ... ok
test_02_validate_target_url_blocks_external_and_subdomain_spoofing ... ok
test_03_generate_s1_intake_creates_valid_docs_and_masks_secrets ... ok
test_04_trigger_pentagi_flow_blocks_unlisted_target_and_supports_dry_run ... ok
test_05_load_allowed_targets_from_manifest ... ok
----------------------------------------------------------------------
Ran 25 tests in 1.440s
OK
```

### 2.2. Kiểm chứng Runtime CLI `tooling/pentagi_bridge.py`
```text
$ python3 /Users/trainguyen/Documents/work/agent/tooling/pentagi_bridge.py trigger --target "http://localhost:8080/api/v1/orders" --dry-run
{
  "status": "DRY_RUN_VALIDATED",
  "target": "http://localhost:8080/api/v1/orders",
  "api_url": "https://localhost:8443",
  "payload": {
    "target": "http://localhost:8080/api/v1/orders",
    "prompt": "Scoped DAST check on modified endpoints"
  }
}

$ python3 /Users/trainguyen/Documents/work/agent/tooling/pentagi_bridge.py trigger --target "https://prod.example.com/api" --dry-run
{"status": "BLOCKED", "error": "Target URL 'https://prod.example.com/api' bị chặn bởi cổng an toàn: không nằm trong allowed_pentest_targets."}
Blocked exit code: 2
```

---

## 3. Security Gate Report (`/security-agent` — 5 Tầng)

| Hạng mục | Kết quả | Chi tiết |
|----------|---------|----------|
| Secret & PII Scan (Code + Memory) | PASS | `grep -rnE` trên toàn bộ workspace và diff trả về `0 matches`. |
| Input & Query Injection Check | PASS | `pentagi_bridge.py` dùng `urllib.request` với JSON body, không nối chuỗi vào shell (`shell=True` không được dùng trong `pentagi_bridge.py`). |
| AuthN / AuthZ & Target Whitelist Guard | PASS | `_matches_prefix_boundary()` chặn cả domain ngoài lẫn subdomain spoofing (`http://localhost.evil.com`). |
| VCS Hygiene (No secret staged, No push) | PASS | Chỉ tạo local restore-point commit, tuyệt đối không chạy `git push` (Rule #15). |
| Dynamic Pentest Gate (DAST / PentAGI) | PASS | Đã tích hợp và kiểm chứng CLI `pentagi_bridge.py trigger` & `intake`. |
- **Verdict:** ✅ PASS

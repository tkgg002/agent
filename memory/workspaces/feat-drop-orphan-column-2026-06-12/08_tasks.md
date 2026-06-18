# 08_tasks

- [ ] T1. Backup 4 file (worker x2, CMS x2) + FE x1
- [ ] T2. worker master_ddl_generator.go: +DropColumn (guard _*)
- [ ] T3. worker master_ddl_handler.go: +Action field + drop branch
- [ ] T4. CMS master_mapping_rule_handler.go: +publishMasterDropColumn +DropColumn +DropRejectedColumns
- [ ] T5. CMS router.go: +2 registerDestructive route
- [ ] T6. FE MasterMappingFieldsPage.tsx: nút per-row "Drop field" (status=rejected) + "Drop all rejected" + Modal.confirm
- [ ] T7. go build CMS + worker = 0 ; tsc FE
- [ ] T8. Deploy + verify drop 1 cột thật (fileUrl) → biến mất; đối soát before/after
- [ ] T9. report_*.md (file đổi + LOC) + 06_validation + 05_progress
- [ ] T10. Pre-flight Rule 14 + service work trước khi báo done

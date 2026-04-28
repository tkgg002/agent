# Implementation — Phase 17 cdc_system Namespace Bridge

## Audit kết luận

CMS backend còn ít nhất 2 runtime model/repo trỏ sai namespace:

- `Source.TableName()` -> `cdc_internal.sources`
- `WizardSession.TableName()` -> `cdc_internal.cdc_wizard_sessions`
- `WizardRepo.AppendProgress()` còn raw SQL update `cdc_internal.cdc_wizard_sessions`

Trong khi migration end-state đã move các bảng này sang `cdc_system`.

## Thay đổi đã áp dụng

- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/model/source.go`
  - `TableName()` -> `cdc_system.sources`
- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/model/wizard_session.go`
  - `TableName()` -> `cdc_system.cdc_wizard_sessions`
- `/Users/trainguyen/Documents/work/cdc-system/cdc-cms-service/internal/repository/wizard_repo.go`
  - raw SQL `AppendProgress()` -> `UPDATE cdc_system.cdc_wizard_sessions ...`

## Kết quả

- `v1/sources` và wizard session backing store không còn bám namespace `cdc_internal`
- màn `Sources & Connectors` vừa refactor không còn đọc từ schema cũ

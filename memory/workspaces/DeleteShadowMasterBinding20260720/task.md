# Task: Xoá Shadow & Xoá Master

## Backend — cdc-cms-service

- [ ] Extend `ShadowBindingRepo` interface (ports/repository.go): thêm GetByID, ListMasterBindingsByShadowID, DeleteShadowBinding
- [ ] Thêm struct `ShadowBindingInfo` vào ports
- [ ] Implement 3 method mới trong shadowBindingRepoGorm
- [ ] NEW: commands/shadow/delete_shadow_binding.go
- [ ] NEW: commands/master/delete_master_binding.go
- [ ] MODIFY: api/shadow/shadow_binding_actions_handler.go — thêm Delete handler
- [ ] MODIFY: api/master/master_registry_handler.go — thêm Delete handler
- [ ] MODIFY: router/router.go — thêm 2 route DELETE
- [ ] MODIFY: server/server.go — wire DeleteShadowBindingHandler, DeleteMasterBindingHandler
- [ ] go build ./... — verify không lỗi

## Frontend — cdc-cms-web

- [ ] MODIFY: pages/TableRegistry.tsx — nút Xoá Shadow ở bindingColumns + handler
- [ ] MODIFY: pages/MasterRegistry.tsx — nút Xoá master + handler
- [ ] npm run build — verify không lỗi

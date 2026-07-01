# TODO - ReconHealStaleReport

- [ ] Modify `healSegmentA` in `internal/handler/recon/recon_heal_v4.go` to use `GetLatestByTable`.
- [ ] Add unit test verifying that healSegmentA returns `noop` if the latest report is healthy.
- [ ] Run existing and new unit tests to ensure stability.

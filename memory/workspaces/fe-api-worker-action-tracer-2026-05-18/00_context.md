# Context: FE API Worker Action Tracer

## Scope
- FE repo: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-web`
- API repo: `/Users/trainguyen/Documents/work/data-hub/cdc-cms-service`
- Worker repo: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`
- User-reported missing flows:
  - FE click `Sync Fields to Shadow` does not trigger centralized-data-service worker to create fields selected for sync.
  - FE click `Snapshot Now` does not trigger centralized-data-service worker to sync data into shadow.

## Constraints
- Fix the two missing flows first.
- Add a tracer for FE -> API -> centralized-data-service so action flow can be checked end-to-end.
- Do not cheat through DB/config mutation to fake success.
- Do not revert existing dirty worktree changes.


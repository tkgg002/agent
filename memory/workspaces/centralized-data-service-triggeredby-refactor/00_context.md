# Context: centralized-data-service TriggeredBy Refactor

## Scope
- Repository: `/Users/trainguyen/Documents/work/data-hub/centralized-data-service`
- Input note: `/Users/trainguyen/Documents/work/data-hub/Rft worker.ini`
- Goal: restructure worker source so TriggeredBy types are easier to manage/debug, and extend kafka-consumer so follow-up actions can run immediately after Kafka consumption.

## Constraints
- Follow `/Users/trainguyen/Documents/work/agent/GEMINI.md` and global lessons.
- Do not change DB/config to fake successful results.
- Do not revert existing user changes in the repo.
- Only touch code needed for TriggeredBy management, Kafka consumer follow-up action, tests, and required reports.


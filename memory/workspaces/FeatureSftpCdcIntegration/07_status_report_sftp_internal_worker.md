# 07_status_report — SFTP CDC Integration (Kafka Connect kafka-connect-fs)
**Ngày cập nhật:** 2026-08-11 10:52 | **Trạng thái:** WAITING FOR DEVOPS

## Tóm tắt

| Hạng mục | Trạng thái | Ghi chú |
|:---|:---:|:---|
| Workspace & Docs Setup | DONE | Full Doc Set 13 file |
| Audit Luồng | DONE | report_kafka_connect_fs_audit.md |
| Pivot Internal Worker to Kafka Connect | DONE | ADR-005, kafka-connect-fs Apache 2.0 |
| config-local.yml topicPrefix | DONE | Bo cdc.goopay, them cdc.sftplocal |
| SourceConnectors.tsx connector class + keys | DONE | FsSourceConnector, policy.sleepy.sleep, file_reader.delimited.settings.header |
| SourceConnectors.tsx form UI | DONE | fs.uris field, an host/port/user/pass/database |
| system_connectors_handler.go | DONE | parseFingerprint FsSourceConnector + extractCredentials fs.uris |
| event_handler.go isSFTP | DONE | nhan ca sftp.* va cdc.sftplocal.* |
| topic_helper.go isSFTPTopic | DONE | strings.Contains(topic, sftp) |
| DevOps cai kafka-connect-fs JAR | PENDING | v1.3.0 vao /opt/kafka/plugins/ |
| E2E verification | PENDING | sau khi JAR duoc cai |

## Build Status
- centralized-data-service: BUILD PASS (exit 0)
- cdc-cms-service: BUILD PASS (exit 0)
- cdc-cms-web TypeScript: TSC PASS (exit 0)

## Files thay doi

| File | Thay doi |
|:---|:---|
| centralized-data-service/config/config-local.yml | Bo cdc.goopay, them cdc.sftplocal |
| centralized-data-service/internal/handler/shadow/event_handler.go | isSFTP detection + db/table parsing |
| centralized-data-service/internal/handler/shadow/topic_helper.go | isSFTPTopic = Contains sftp |
| cdc-cms-service/internal/api/source/system_connectors_handler.go | parseFingerprint + extractCredentials |
| cdc-cms-web/src/pages/SourceConnectors.tsx | connector class, config keys, form UI |

## Config JSON chuan (kafka-connect-fs)

connector.class = com.github.mmolimar.kafka.connect.fs.FsSourceConnector
tasks.max = 1
fs.uris = sftp://user:pass@host:2022/path
topic = cdc.sftplocal.reconcile.final
policy.class = ...SleepyPolicy
policy.sleepy.sleep = 30000
policy.regexp = ^reconcile_final_.*\.csv$
policy.recursive = false
file_reader.class = ...CsvFileReader
file_reader.delimited.settings.header = true
value.converter = org.apache.kafka.connect.json.JsonConverter
value.converter.schemas.enable = false
key.converter = org.apache.kafka.connect.storage.StringConverter

## Next Actions
1. DevOps cai JAR restart Kafka Connect
2. Verify: curl http://10.200.186.203:8083/connector-plugins | jq .[].class | grep FsSource
3. Tao connector qua CMS UI
4. Check topic cdc.sftplocal.reconcile.final nhan messages
5. E2E: consume shadow DB transmute master DB

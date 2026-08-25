# 11 - Change Report (Final Clean)

## 1. Tổng quan thay đổi
- **File sửa đổi:** `cdc-cms-web/src/pages/SourceConnectors.tsx`
- **Tác động:** 
  1. Loại bỏ duplicate segment cho MongoDB/Postgres/MySQL.
  2. Xóa bỏ fallback thừa `|| TOPIC_PREFIX_POSTGRESQL` ở dòng 435.
  3. Khóa cứng trường `Topic Prefix` trên UI.

## 2. Chi tiết Git Diff đối soát
```diff
diff --git a/src/pages/SourceConnectors.tsx b/src/pages/SourceConnectors.tsx
index 2e97bf4..fef3934 100644
--- a/src/pages/SourceConnectors.tsx
+++ b/src/pages/SourceConnectors.tsx
@@ -391,7 +391,7 @@ function parseConnectionSeed(source: SourceFingerprint, connector?: ConnectorVie
     return {
       dbKind,
       connectorName: source.connector_name,
-      topicPrefix: source.topic_prefix || cfg['topic.prefix'] || `cdc.goopay.${slugifyForShadow(source.connector_name || 'connector')}`,
+      topicPrefix: source.topic_prefix || cfg['topic.prefix'] || TOPIC_PREFIX_MONGODB,
       connectionUrl,
       database,
       collectionNames,
@@ -432,7 +432,7 @@ function parseConnectionSeed(source: SourceFingerprint, connector?: ConnectorVie
   return {
     dbKind,
     connectorName: source.connector_name,
-    topicPrefix: source.topic_prefix || cfg['topic.prefix'] || (source.connector_name ? `${TOPIC_PREFIX_BY_DB[dbKind] || TOPIC_PREFIX_POSTGRESQL}.${source.connector_name}` : ''),
+    topicPrefix: source.topic_prefix || cfg['topic.prefix'] || TOPIC_PREFIX_BY_DB[dbKind],
     host: cfg['database.hostname'] || source.server_address?.split(':')[0] || 'localhost',
     port: Number(cfg['database.port'] || source.server_address?.split(':')[1] || (dbKind === 'mysql' ? 3306 : 5432)),
     database: cfg['database.include.list'] || cfg['database.dbname'] || source.database_include_list || '',
@@ -481,13 +481,15 @@ export default function SourceConnectors() {
     if (!editorOpen || editorMode !== 'create') return;
     const name = slugifyForShadow(String(connectorNameValue || 'connector'));
     if (dbKind === 'sftp') {
+      // SFTP (kafka-connect-fs): cần connector name trong prefix vì không tự append database.collection
       form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_SFTP}.${name}`);
     } else if (dbKind === 'mongodb') {
-      form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_MONGODB}.${name}`);
+      // Debezium tự append {database}.{collection} → prefix chỉ cần base (e.g. cdc.goopay)
       form.setFieldValue('topicPrefix', TOPIC_PREFIX_MONGODB);
     } else if (dbKind === 'mysql') {
-      form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_MYSQL}.${name}`);
+      form.setFieldValue('topicPrefix', TOPIC_PREFIX_MYSQL);
     } else if (dbKind === 'postgresql') {
-      form.setFieldValue('topicPrefix', `${TOPIC_PREFIX_POSTGRESQL}.${name}`);
+      form.setFieldValue('topicPrefix', TOPIC_PREFIX_POSTGRESQL);
     }
   }, [dbKind, editorOpen, editorMode, form, connectorNameValue]);
```

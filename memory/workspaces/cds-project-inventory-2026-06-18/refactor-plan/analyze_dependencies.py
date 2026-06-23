import os
import re

service_dir = "/Users/trainguyen/Documents/work/data-hub/centralized-data-service/internal/service"
files = [f for f in os.listdir(service_dir) if f.endswith(".go") and not f.endswith("_test.go")]

# Map symbol -> defining file
symbol_to_file = {}
file_defs = {}

struct_re = re.compile(r"type\s+([A-Z][a-zA-Z0-9_]*)\s+(struct|interface)")
func_re = re.compile(r"func\s+([A-Z][a-zA-Z0-9_]*)\(")

# Target sub-packages mapping
subpackages = {
    # source
    "metadata_registry_service.go": "source",
    "registry_service.go": "source",
    "connection_manager.go": "source",
    "connection_overrides.go": "source",
    "connector_resolver.go": "source",
    "source_router.go": "source",
    "mongo_introspection.go": "source",
    "scan_service.go": "source",
    # shadow
    "schema_adapter.go": "shadow",
    "dynamic_mapper.go": "shadow",
    "child_explode.go": "shadow",
    "enrichment_service.go": "shadow",
    "bridge_service.go": "shadow",
    "type_resolver.go": "shadow",
    "text_sanitizer.go": "shadow",
    # master
    "master_ddl_generator.go": "master",
    "transmuter.go": "master",
    "transmute_scheduler.go": "master",
    "child_explode_master.go": "master",
    "job_monitor.go": "master",
    "transform_registry.go": "master",
    # governance
    "masking_service.go": "governance",
    "schema_inspector.go": "governance",
    "schema_validator.go": "governance",
    "activity_logger.go": "governance",
    "partition_dropper.go": "governance",
    "wal_monitor.go": "governance",
    "full_count_aggregator.go": "governance",
    "debezium_signal.go": "governance",
    "timestamp_detector.go": "governance",
    "backfill_source_ts.go": "governance",
    # recon
    "recon_engine.go": "recon",
    "recon_tier_a.go": "recon",
    "recon_tier_b.go": "recon",
    "recon_source_agent.go": "recon",
    "recon_dest_agent.go": "recon",
    "recon_heal.go": "recon",
    "recon_alert.go": "recon",
    "dlq_worker.go": "recon",
    "provisioning_orchestrator.go": "recon",
    "provisioning_state_machine.go": "recon"
}

for fname in files:
    path = os.path.join(service_dir, fname)
    with open(path, "r", encoding="utf-8") as f:
        content = f.read()
    
    structs = struct_re.findall(content)
    funcs = func_re.findall(content)
    
    defined = [s[0] for s in structs] + funcs
    file_defs[fname] = defined
    for d in defined:
        symbol_to_file[d] = fname

print(f"Total symbols defined: {len(symbol_to_file)}")

# Now parse references
dependencies = {}
for fname in files:
    path = os.path.join(service_dir, fname)
    with open(path, "r", encoding="utf-8") as f:
        content = f.read()
    
    # Strip comments to avoid false references
    content_clean = re.sub(r"//[^\n]*", "", content)
    content_clean = re.sub(r"/\*.*?\*/", "", content_clean, flags=re.DOTALL)
    
    deps = set()
    for symbol, def_file in symbol_to_file.items():
        if def_file == fname:
            continue
        # Check if symbol is used as a word
        if re.search(r"\b" + symbol + r"\b", content_clean):
            deps.add(def_file)
    dependencies[fname] = deps

# Print summary
print("FILE LEVEL DEPENDENCIES:")
for fname, deps in sorted(dependencies.items()):
    if deps:
        print(f"  {fname} ({subpackages.get(fname, 'unknown')}) -> {sorted([f'{d} ({subpackages.get(d, None)})' for d in deps])}")

print("\nPACKAGE LEVEL DEPENDENCIES:")
pkg_deps = {}
for fname, deps in dependencies.items():
    src_pkg = subpackages.get(fname, "unknown")
    if src_pkg not in pkg_deps:
        pkg_deps[src_pkg] = set()
    for d in deps:
        dst_pkg = subpackages.get(d, "unknown")
        if dst_pkg != src_pkg:
            pkg_deps[src_pkg].add(dst_pkg)

for pkg, deps in sorted(pkg_deps.items()):
    print(f"  {pkg} -> {sorted(list(deps))}")

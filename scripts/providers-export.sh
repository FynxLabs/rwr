#!/usr/bin/env bash
# Export the CUE provider sources to the committed JSON the binary embeds.
# CUE is build-time only; the exported JSON is what ships.
set -euo pipefail

# CUE_BIN lets CI run cue via `go run` instead of a separate installer.
CUE_BIN="${CUE_BIN:-cue}"

out_dir="internal/system/definitions/providers"
export_json="$($CUE_BIN export ./providers/cue -e providers)"

mkdir -p "$out_dir"
python3 - "$out_dir" <<'PY' "$export_json"
import json, os, sys
out_dir = sys.argv[1]
providers = json.loads(sys.argv[2])
for name, prov in sorted(providers.items()):
    path = os.path.join(out_dir, name + ".json")
    with open(path, "w") as f:
        json.dump(prov, f, indent=2, ensure_ascii=False, sort_keys=True)
        f.write("\n")
print(f"exported {len(providers)} providers to {out_dir}")
PY

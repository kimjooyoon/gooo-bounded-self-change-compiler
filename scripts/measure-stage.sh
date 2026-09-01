#!/usr/bin/env bash
set -euo pipefail

stage=${1:?stage name is required}
measurement=${2:?measurement output path is required}
shift 2
if [ "$#" -eq 0 ]; then
	echo "a command is required" >&2
	exit 64
fi

mkdir -p "$(dirname "$measurement")"
time_log="$measurement.time-v"
set +e
/usr/bin/time -v -o "$time_log" "$@"
command_status=$?
set -e

elapsed=$(awk -F': ' '/Elapsed \(wall clock\) time/ {print $2}' "$time_log")
rss=$(awk -F': ' '/Maximum resident set size/ {print $2}' "$time_log")
python3 - "$stage" "$elapsed" "$rss" "$measurement" <<'PY'
import json
import re
import sys

stage, elapsed, rss, destination = sys.argv[1:]
parts = elapsed.strip().split(":")
if len(parts) == 3:
    seconds = int(parts[0]) * 3600 + int(parts[1]) * 60 + float(parts[2])
elif len(parts) == 2:
    seconds = int(parts[0]) * 60 + float(parts[1])
else:
    seconds = float(parts[0])
wall_ms = int(round(seconds * 1000))
peak_rss_kib = int(re.sub(r"[^0-9-]", "", rss))
with open(destination, "w", encoding="utf-8") as output:
    json.dump({"stage": stage, "wall_ms": wall_ms, "peak_rss_kib": peak_rss_kib}, output, indent=2, sort_keys=True)
    output.write("\n")
PY
exit "$command_status"

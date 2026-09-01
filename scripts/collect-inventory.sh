#!/usr/bin/env bash
set -euo pipefail

root=${1:?repository root is required}
generated=${2:?generated artifact directory is required}
output=${3:?inventory output path is required}

go_files=$(find "$root" -type f -name '*.go' ! -path "$root/.git/*" -print)
gooo_files=$(find "$root" -type f -name '*.gooo' ! -path "$root/.git/*" -print)
go_count=$(printf '%s\n' "$go_files" | awk 'NF {count++} END {print count+0}')
gooo_count=$(printf '%s\n' "$gooo_files" | awk 'NF {count++} END {print count+0}')
go_lines=$(find "$root" -type f -name '*.go' ! -path "$root/.git/*" -exec awk 'END {print NR}' {} + | awk '{total += $1} END {print total+0}')
gooo_lines=$(find "$root" -type f -name '*.gooo' ! -path "$root/.git/*" -exec awk 'END {print NR}' {} + | awk '{total += $1} END {print total+0}')
descendant_dirs=$(find "$root" -type d ! -path "$root" ! -path "$root/.git" ! -path "$root/.git/*" | awk 'END {print NR+0}')
regular_files=$(find "$root" -type f ! -path "$root/.git/*" ! -path "$root/README.md" | awk 'END {print NR+0}')
generated_artifacts=$(find "$generated" -type f | awk 'END {print NR+0}')
generated_bytes=$(find "$generated" -type f -exec stat -c '%s' {} + | awk '{total += $1} END {print total+0}')

jq -n \
	--argjson go_files "$go_count" \
	--argjson go_physical_lines "$go_lines" \
	--argjson gooo_files "$gooo_count" \
	--argjson gooo_physical_lines "$gooo_lines" \
	--argjson descendant_dirs "$descendant_dirs" \
	--argjson regular_files "$regular_files" \
	--argjson generated_artifacts "$generated_artifacts" \
	--argjson generated_bytes "$generated_bytes" \
	'{root_readme_excluded:true,go:{files:$go_files,physical_lines:$go_physical_lines},gooo:{files:$gooo_files,physical_lines:$gooo_physical_lines},descendant_dirs:$descendant_dirs,regular_files:$regular_files,generated_artifacts:{count:$generated_artifacts,bytes:$generated_bytes}}' > "$output"

#!/usr/bin/env bash
# Shared region env setup for start_server.sh / start_collector.sh.
# Sets: REGION, DATA_DIR, LISTEN (web), REGION_ENV_REMAINING_ARGS
# Sources .env.{id} when present so godotenv cannot refill wrong-region keys from .env.local.

region_env_setup() {
  local mode="$1" # web|collector
  shift

  local region=""
  local -a rest=()
  if [[ $# -gt 0 ]]; then
    case "$1" in
      -* ) rest=("$@") ;;
      * )
        if region_id_known "$1"; then
          region="$1"
          shift
          rest=("$@")
        else
          echo "Unknown region '$1' (not in config/regions.yaml). Known: $(region_ids_csv)" >&2
          exit 1
        fi
        ;;
    esac
  fi
  if [[ -z "$region" ]]; then
    region="pacific"
  fi

  export_dotenv_file ".env.${region}"

  export REGION="$region"
  if [[ -z "${DATA_DIR:-}" ]]; then
    export DATA_DIR="./data/${region}"
  fi
  if [[ "$mode" == "web" ]]; then
    if [[ -z "${LISTEN:-}" ]]; then
      local listen
      listen="$(region_dev_listen "$region")"
      export LISTEN="${listen:-:8082}"
    fi
  fi

  echo "region_env: REGION=${REGION} DATA_DIR=${DATA_DIR}${LISTEN:+ LISTEN=${LISTEN}} PUBLIC_SITE_URL=${PUBLIC_SITE_URL:-}"
  REGION_ENV_REMAINING_ARGS=("${rest[@]+"${rest[@]}"}")
}

region_ids_csv() {
  # Prefer yq-less parse: grep id: lines under regions.yaml
  awk '
    /^[[:space:]]*-[[:space:]]*id:[[:space:]]*/ {
      gsub(/^[[:space:]]*-[[:space:]]*id:[[:space:]]*/, "")
      gsub(/[[:space:]]+$/, "")
      gsub(/^["'\'']|["'\'']$/, "")
      if (NR && $0 != "") { if (n++) printf ","; printf "%s", $0 }
    }
  ' config/regions.yaml
}

region_id_known() {
  local want="$1" id
  while IFS= read -r id; do
    [[ "$id" == "$want" ]] && return 0
  done < <(awk '
    /^[[:space:]]*-[[:space:]]*id:[[:space:]]*/ {
      gsub(/^[[:space:]]*-[[:space:]]*id:[[:space:]]*/, "")
      gsub(/[[:space:]]+$/, "")
      gsub(/^["'\'']|["'\'']$/, "")
      if ($0 != "") print $0
    }
  ' config/regions.yaml)
  return 1
}

region_dev_listen() {
  local want="$1" id="" listen=""
  while IFS= read -r line; do
    if [[ "$line" =~ ^[[:space:]]*-[[:space:]]*id:[[:space:]]*(.*)$ ]]; then
      id="${BASH_REMATCH[1]}"
      id="${id%\"}"; id="${id#\"}"
      id="${id%\'}"; id="${id#\'}"
      id="${id%"${id##*[![:space:]]}"}"
    elif [[ -n "$id" && "$id" == "$want" && "$line" =~ ^[[:space:]]*dev_listen:[[:space:]]*(.*)$ ]]; then
      listen="${BASH_REMATCH[1]}"
      listen="${listen%\"}"; listen="${listen#\"}"
      listen="${listen%\'}"; listen="${listen#\'}"
      echo "$listen"
      return 0
    fi
  done < config/regions.yaml
  echo ""
}

# Export KEY=VAL from a dotenv-style file (no export of comments/blank). Does not override existing env.
export_dotenv_file() {
  local f="$1"
  [[ -f "$f" ]] || return 0
  local line key val
  while IFS= read -r line || [[ -n "$line" ]]; do
    line="${line#"${line%%[![:space:]]*}"}"
    [[ -z "$line" || "$line" == \#* ]] && continue
    if [[ "$line" =~ ^([A-Za-z_][A-Za-z0-9_]*)=(.*)$ ]]; then
      key="${BASH_REMATCH[1]}"
      val="${BASH_REMATCH[2]}"
      if [[ -n "${!key+x}" ]]; then
        continue
      fi
      # Strip optional surrounding quotes
      if [[ "$val" =~ ^\"(.*)\"$ ]]; then
        val="${BASH_REMATCH[1]}"
      elif [[ "$val" =~ ^\'(.*)\'$ ]]; then
        val="${BASH_REMATCH[1]}"
      fi
      export "$key=$val"
    fi
  done < "$f"
}

#!/usr/bin/env bash
# bitbucket-tool.sh — example shell script tool for the clikit shell bridge.
# Pipe output through toolkit-fmt to get the same JSON output contract as
# native Go tools (structured errors, schema transformation, binary guard).
#
# Usage (registered as a toolkit tool):
#   toolkit bitbucket repo list [--project KEY]
#   toolkit bitbucket pr list  [--project KEY] [--repo SLUG]

set -euo pipefail

SUBCMD="${1:-}"
RESOURCE="${2:-}"
ACTION="${3:-}"
shift 3 2>/dev/null || true

BITBUCKET_URL="${BITBUCKET_URL:-http://localhost:7990}"
BITBUCKET_TOKEN="${BITBUCKET_TOKEN:-}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

auth_header() {
  if [[ -n "$BITBUCKET_TOKEN" ]]; then
    echo "-H" "Authorization: Bearer $BITBUCKET_TOKEN"
  fi
}

case "$RESOURCE $ACTION" in
  "repo list")
    PROJECT=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --project) PROJECT="$2"; shift 2 ;;
        *) shift ;;
      esac
    done
    URL="$BITBUCKET_URL/rest/api/1.0/projects/${PROJECT}/repos"
    curl -sf $(auth_header) "$URL" \
      | toolkit-fmt output --schema "$SCRIPT_DIR/schemas/bitbucket-repos.json"
    ;;
  "pr list")
    PROJECT=""
    REPO=""
    while [[ $# -gt 0 ]]; do
      case "$1" in
        --project) PROJECT="$2"; shift 2 ;;
        --repo)    REPO="$2";    shift 2 ;;
        *) shift ;;
      esac
    done
    URL="$BITBUCKET_URL/rest/api/1.0/projects/${PROJECT}/repos/${REPO}/pull-requests"
    curl -sf $(auth_header) "$URL" \
      | toolkit-fmt output --schema "$SCRIPT_DIR/schemas/bitbucket-prs.json"
    ;;
  *)
    toolkit-fmt error \
      --message "unknown command: $RESOURCE $ACTION" \
      --suggestion "available: repo list, pr list" \
      --exit-code 1
    exit 1
    ;;
esac

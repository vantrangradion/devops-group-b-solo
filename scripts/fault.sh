#!/usr/bin/env bash
# Flip the app's runtime fault toggles (app/README.md#fault-toggles) on every
# devops-app pod, or on one pod. Each pod keeps its own state, so the script
# port-forwards to each pod's admin port in turn and sends the change there.
#
#   scripts/fault.sh error <0..1> [pod]        chance that /api/work returns 500
#   scripts/fault.sh latency <0..2000> [pod]   ms added to every /api/work
#   scripts/fault.sh ready <true|false> [pod]  what /ready reports (readinessProbe)
#   scripts/fault.sh reset [pod]               back to the defaults
#   scripts/fault.sh show [pod]                print the current state
#
# Needs kubectl and curl. KUBE_CONTEXT (default k3d-devops-solo) picks the
# cluster.
set -euo pipefail

CONTEXT="${KUBE_CONTEXT:-k3d-devops-solo}"
NAMESPACE=devops-app
SELECTOR=app.kubernetes.io/name=devops-app
LOCAL_PORT_BASE=18181

usage() {
  sed -n '2,13p' "$0" | sed 's/^# \{0,1\}//'
  exit 2
}

[ $# -ge 1 ] || usage
cmd=$1
shift

method=PUT
body=
case "$cmd" in
  error)
    [ $# -ge 1 ] || usage
    body="{\"error_rate\": $1}"; shift ;;
  latency)
    [ $# -ge 1 ] || usage
    body="{\"extra_latency_ms\": $1}"; shift ;;
  ready)
    [ $# -ge 1 ] || usage
    case "$1" in true|false) ;; *) echo "ready takes true or false" >&2; exit 2 ;; esac
    body="{\"ready\": $1}"; shift ;;
  reset) method=DELETE ;;
  show) method=GET ;;
  *) usage ;;
esac

kc() { kubectl --context "$CONTEXT" -n "$NAMESPACE" "$@"; }

if [ $# -ge 1 ]; then
  pods=$1
else
  pods=$(kc get pods -l "$SELECTOR" --field-selector=status.phase=Running -o jsonpath='{.items[*].metadata.name}')
fi
[ -n "$pods" ] || { echo "no running devops-app pods in $NAMESPACE" >&2; exit 1; }

i=0
status=0
for pod in $pods; do
  port=$((LOCAL_PORT_BASE + i))
  i=$((i + 1))
  kc port-forward "pod/$pod" "$port:admin" >/dev/null 2>&1 &
  pf=$!
  # Wait for the forward to accept connections (up to 5 s).
  for _ in 1 2 3 4 5 6 7 8 9 10; do
    curl -s -o /dev/null "http://127.0.0.1:$port/faults" && break
    sleep 0.5
  done
  if [ "$method" = PUT ]; then
    out=$(curl -sS -X PUT -H 'Content-Type: application/json' -d "$body" -w '\n%{http_code}' "http://127.0.0.1:$port/faults" || printf '\nfailed')
  else
    out=$(curl -sS -X "$method" -w '\n%{http_code}' "http://127.0.0.1:$port/faults" || printf '\nfailed')
  fi
  kill "$pf" 2>/dev/null || true
  wait "$pf" 2>/dev/null || true
  # The last line is the status code; the JSON body before it ends in a newline.
  code=${out##*$'\n'}
  resp=${out%$'\n'*}
  resp=${resp%$'\n'}
  [ "$code" = 200 ] || status=1
  echo "$pod: $resp${code:+ ($code)}"
done
exit $status

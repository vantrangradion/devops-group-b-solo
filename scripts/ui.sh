#!/usr/bin/env bash
# Keeps the demo's web UIs reachable on localhost: Grafana, Argo CD and
# Prometheus run inside the cluster, so each needs a kubectl port-forward.
# Each forward runs in the background in a loop, so it comes back by itself
# when it drops (a pod restart, a Docker restart, a laptop sleep).
#
#   scripts/ui.sh start    start the forwards (safe to rerun) and print the URLs
#   scripts/ui.sh status   show which forwards answer
#   scripts/ui.sh stop     stop them
#
# Needs kubectl and curl. KUBE_CONTEXT (default k3d-devops-solo) picks the
# cluster. Passwords are never printed; see the commands at the end of start.
set -euo pipefail

CONTEXT="${KUBE_CONTEXT:-k3d-devops-solo}"
STATE="${TMPDIR:-/tmp}/devops-demo-ui"

# name namespace service local-port remote-port scheme path
FORWARDS="grafana monitoring kps-grafana 3000 80 http /login
argocd argocd argocd-server 8443 443 https /
prometheus monitoring kps-kube-prometheus-stack-prometheus 9090 9090 http /-/ready"

answers() { # scheme port path
  curl -sk -o /dev/null --max-time 2 "$1://127.0.0.1:$2$3"
}

start() {
  mkdir -p "$STATE"
  echo "$FORWARDS" | while read -r name ns svc lport rport scheme path; do
    if answers "$scheme" "$lport" "$path"; then
      echo "$name: already up"
      continue
    fi
    # The loop restarts the forward whenever kubectl exits.
    nohup bash -c "while true; do kubectl --context '$CONTEXT' -n '$ns' port-forward 'svc/$svc' '$lport:$rport' >>'$STATE/$name.log' 2>&1; sleep 1; done" \
      >/dev/null 2>&1 &
    echo $! >"$STATE/$name.pid"
  done
  # Wait for each to answer (up to 30 s).
  status=0
  echo "$FORWARDS" | while read -r name ns svc lport rport scheme path; do
    for _ in $(seq 30); do answers "$scheme" "$lport" "$path" && break; sleep 1; done
    answers "$scheme" "$lport" "$path" || { echo "$name: not answering yet, see $STATE/$name.log"; exit 1; }
  done || status=1
  show
  cat <<EOF

Log in as admin. Passwords (run in a terminal that is not on screen):
  Grafana: kubectl --context $CONTEXT -n monitoring get secret kps-grafana -o jsonpath='{.data.admin-password}' | base64 -d; echo
  Argo CD: kubectl --context $CONTEXT -n argocd get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d; echo
Argo CD uses a self-signed certificate: accept the browser warning once.
EOF
  return $status
}

show() {
  echo "$FORWARDS" | while read -r name ns svc lport rport scheme path; do
    if answers "$scheme" "$lport" "$path"; then state=up; else state=DOWN; fi
    url="$scheme://localhost:$lport"
    [ "$name" = grafana ] && url="$url/d/devops-app"
    printf '%-11s %-5s %s\n' "$name" "$state" "$url"
  done
}

stop() {
  echo "$FORWARDS" | while read -r name ns svc lport rport scheme path; do
    if [ -f "$STATE/$name.pid" ]; then
      kill "$(cat "$STATE/$name.pid")" 2>/dev/null || true
      rm -f "$STATE/$name.pid"
    fi
    # The kubectl started by the loop outlives it; stop that too.
    pkill -f "port-forward svc/$svc $lport:$rport" 2>/dev/null || true
    echo "$name: stopped"
  done
}

case "${1:-}" in
  start) start ;;
  status) show ;;
  stop) stop ;;
  *) sed -n '2,13p' "$0" | sed 's/^# \{0,1\}//'; exit 2 ;;
esac

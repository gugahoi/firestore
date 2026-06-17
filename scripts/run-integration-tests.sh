#!/usr/bin/env bash
# Boots the Firestore emulator, runs integration tests, then tears it down.
set -euo pipefail

PORT="${PORT:-8765}"
LOG=$(mktemp)
PGID=""

cleanup() {
    [[ -n "$PGID" ]] && kill -- "-$PGID" 2>/dev/null || true
    rm -f "$LOG"
}
trap cleanup EXIT

# ── prerequisites ──────────────────────────────────────────────────────────────
if ! command -v gcloud &>/dev/null; then
    echo "error: gcloud not found — install the Google Cloud SDK" >&2
    exit 1
fi
if ! gcloud components list --filter="id=cloud-firestore-emulator" \
        --format="value(state.name)" 2>/dev/null | grep -qi "installed"; then
    echo "error: cloud-firestore-emulator component not installed" >&2
    echo "       Run: gcloud components install cloud-firestore-emulator" >&2
    exit 1
fi

# ── start emulator ─────────────────────────────────────────────────────────────
echo "Starting Firestore emulator on localhost:${PORT} ..."
# set -m assigns each background job its own process group so kill -- -$PGID
# also terminates the Java subprocess spawned by the gcloud wrapper.
set -m
gcloud emulators firestore start --host-port="localhost:${PORT}" 2>"$LOG" &
EMULATOR_PID=$!
PGID=$(ps -o pgid= -p "$EMULATOR_PID" 2>/dev/null | tr -d ' ') || PGID=$EMULATOR_PID
set +m

# ── wait for ready (up to 30 s) ────────────────────────────────────────────────
for i in $(seq 1 30); do
    if grep -q "Dev App Server is now running" "$LOG" 2>/dev/null; then
        echo "Emulator ready."
        break
    fi
    if ! kill -0 "$EMULATOR_PID" 2>/dev/null; then
        echo "error: emulator exited before becoming ready. Log:" >&2
        cat "$LOG" >&2
        exit 1
    fi
    if [[ $i -eq 30 ]]; then
        echo "error: emulator did not become ready within 30 s. Log:" >&2
        cat "$LOG" >&2
        exit 1
    fi
    sleep 1
done

# ── run tests ──────────────────────────────────────────────────────────────────
export FIRESTORE_EMULATOR_HOST="localhost:${PORT}"
go test -tags integration -v ./... "$@"

#!/bin/sh
set -e

needs_install() {
  if [ ! -x node_modules/.bin/vite ]; then
    return 0
  fi
  if [ ! -d node_modules/@rollup/rollup-linux-x64-musl ]; then
    return 0
  fi
  return 1
}

if needs_install; then
  echo "Installing npm dependencies for Linux..."
  attempt=1
  while [ "$attempt" -le 3 ]; do
    if npm install; then
      break
    fi
    if [ "$attempt" -eq 3 ]; then
      echo "Failed to install npm dependencies after 3 attempts."
      exit 1
    fi
    echo "npm install failed (attempt ${attempt}/3). Retrying in 5s..."
    attempt=$((attempt + 1))
    sleep 5
  done
fi

cat <<'EOF'

========================================
  NoteHub is ready

  App:      http://localhost:5173
  API:      http://localhost:8081
  Health:   http://localhost:8081/health

========================================

EOF

exec npm run dev -- --host 0.0.0.0 --port 5173

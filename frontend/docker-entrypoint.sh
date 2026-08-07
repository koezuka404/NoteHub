#!/bin/sh
set -e

cat <<'EOF'

========================================
  NoteHub is ready

  App:      http://localhost:5173
  API:      http://localhost:8081
  Health:   http://localhost:8081/health

========================================

EOF

exec npm run dev -- --host 0.0.0.0 --port 5173

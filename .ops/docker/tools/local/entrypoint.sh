#!/usr/bin/env bash

log() {
  echo "START-SCRIPT: $1"
}

build() {
  log "Building server binary"
  go mod vendor
  go build -gcflags "all=-N -l" -mod vendor -o /go/src/app/build/clickhouse-tools /go/src/app/cmd/tools/app.go
  cp .env.dist ./build/.env
}

rebuild() {
  log "Rerun server"
  build
}

hotReloading() {
  log "Run hotReloading"
  inotifywait -e "CREATE,MODIFY,DELETE,MOVED_TO,MOVED_FROM" -m -r ./ | (
    while true; do
      read path action file
      ext=${file: -3}
      if [[ "$ext" == ".go" ]]; then
        echo "$file"
      fi
    done
  ) | (
    WAITING=""
    while true; do
      file=""
      read -t 1 file
      if test -z "$file"; then
        if test ! -z "$WAITING"; then
          echo "CHANGED"
          WAITING=""
        fi
      else
        log "File ${file} changed" >>/tmp/filechanges.log
        WAITING=1
      fi
    done
  ) | (
    while true; do
      read TMP
      log "File Changed. Reloading..."
      rebuild
    done
  )
}

initFileChangeLogger() {
  echo "" > /tmp/filechanges.log
  tail -f /tmp/filechanges.log &
}

main() {
  initFileChangeLogger
  build
  hotReloading
}

main

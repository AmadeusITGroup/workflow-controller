#!/usr/bin/env bash
set -e

[ -z "$COVER" ] && COVER=.cover
profile="$COVER/cover.out"
mode=atomic

OS=$(uname)
race_flag="-race"
if [ "$OS" = "Linux" ]; then
  # check Alpine - alpine does not support race test
  if [ -f "/etc/alpine-release" ]; then
    race_flag=""
  fi
fi
if [ ! -z "$SKIP_RACE" ]; then
  race_flag=""
fi

generate_cover_data() {
  [ -d "${COVER}" ] && rm -rf "${COVER:?}/*"
  [ -d "${COVER}" ] || mkdir -p "${COVER}"

  pkgs=($(go list -f '{{if .TestGoFiles}}{{ .ImportPath }}{{end}}' ./... | grep -v vendor | grep -v test/e2e))

  for pkg in "${pkgs[@]}"; do
    f="${COVER}/$(echo $pkg | tr / -).cover"
    tout="${COVER}/$(echo $pkg | tr / -)_tests.out"
    go test -v $race_flag -covermode="$mode" -coverprofile="$f" "$pkg" | tee "$tout"
  done

  echo "mode: $mode" >"$profile"
  grep -h -v "^mode:" "${COVER}"/*.cover >>"$profile"
}

generate_cover_report() {
  go tool cover -${1}="$profile" -o "${COVER}/coverage.html"
}

generate_cover_data 
generate_cover_report html


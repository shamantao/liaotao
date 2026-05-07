#!/usr/bin/env bash
# java-env.sh - resolves Java runtime for local scripts.
# Responsibilities: detect a usable JDK, including Homebrew OpenJDK layouts,
# and export JAVA_HOME/PATH when Java is installed but not in shell PATH.

resolve_java_runtime() {
  if java -version >/dev/null 2>&1; then
    return 0
  fi

  local candidates=(
    "/opt/homebrew/opt/openjdk@21"
    "/usr/local/opt/openjdk@21"
    "/opt/homebrew/opt/openjdk"
    "/usr/local/opt/openjdk"
  )

  local root
  for root in "${candidates[@]}"; do
    if [[ -x "$root/bin/java" ]]; then
      export JAVA_HOME="$root"
      export PATH="$JAVA_HOME/bin:$PATH"
      if java -version >/dev/null 2>&1; then
        return 0
      fi
    fi
  done

  return 1
}

java_major_version() {
  local first_line
  first_line="$(java -version 2>&1 | head -n1)"

  if [[ "$first_line" =~ \"([0-9]+)(\.[0-9]+)*\" ]]; then
    echo "${BASH_REMATCH[1]}"
    return 0
  fi

  return 1
}

ensure_java_21_runtime() {
  if ! resolve_java_runtime; then
    return 1
  fi

  local major
  major="$(java_major_version || echo "")"
  [[ "$major" == "21" ]]
}

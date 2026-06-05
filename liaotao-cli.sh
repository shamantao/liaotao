#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
JAR="$SCRIPT_DIR/app-cli/build/libs/liaotao-1.0.0.jar"
JAVA="${JAVA_HOME:-/opt/homebrew/opt/openjdk@21/libexec/openjdk.jdk/Contents/Home}/bin/java"

build() {
    echo "==> Building shadowJar..."
    (cd "$SCRIPT_DIR" && ./gradlew :app-cli:shadowJar -q)
    echo "==> Done"
}

case "${1:-help}" in
    build|b)
        build
        ;;
    launch|l)
        shift
        exec "$JAVA" -jar "$JAR" chat "$@"
        ;;
    chat)
        shift
        exec "$JAVA" -jar "$JAR" chat "$@"
        ;;
    config)
        shift
        exec "$JAVA" -jar "$JAR" config "$@"
        ;;
    conversations|conv)
        shift
        exec "$JAVA" -jar "$JAR" conversations "$@"
        ;;
    status|st)
        exec "$JAVA" -jar "$JAR"
        ;;
    help|h|--help|-h)
        echo "liaotao-cli.sh — Liaotao CLI launcher"
        echo ""
        echo "Usage:"
        echo "  liaotao-cli.sh build          Rebuild the JAR"
        echo "  liaotao-cli.sh launch         Interactive chat (same as 'chat')"
        echo "  liaotao-cli.sh chat [msg]     Chat (one-shot or interactive)"
        echo "  liaotao-cli.sh config <cmd>   Manage providers (list|add|edit|delete|test)"
        echo "  liaotao-cli.sh conversations  Manage conversations (list|show|delete)"
        echo "  liaotao-cli.sh status         Show CLI status"
        echo "  liaotao-cli.sh help           This help"
        ;;
    *)
        exec "$JAVA" -jar "$JAR" "$@"
        ;;
esac

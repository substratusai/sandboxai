#!/bin/bash

set -eu

repo_root=$(git rev-parse --show-toplevel)
echo "Repo root: $repo_root"
export PATH="$PATH:$repo_root/bin"

# Start sandboxaid in the background.
SANDBOXAID_PORT=8080 SANDBOXAID_HOST=localhost $repo_root/bin/sandboxaid &
sandboxaid_pid=$!

# Shutdown sandboxaid on script exit.
cleanup() {
	echo "Stopping sandboxaid (pid $sandboxaid_pid)..."
	kill $sandboxaid_pid
}
trap cleanup EXIT

# Wait for sandboxaid to start.
echo -n "Waiting for sandboxaid to start"
for i in {1..10}; do
	echo -n "."
	set +e
	curl http://localhost:8080/v1/healthz >/dev/null 2>&1
	curl_exit_code=$?
	set -e
	if [ $curl_exit_code -eq 0 ]; then
		echo ""
		echo "sandboxaid started."
		break
	elif [ $i -eq 10 ]; then
		echo ""
		echo "sandboxaid failed to start after 10 seconds."
		exit 1
	fi
	sleep 1
done

echo "Starting tests..."
export SANDBOXAI_BASE_URL="http://localhost:8080/v1"

export TEST_SANDBOX_PATH="${repo_root}/test/e2e/sandbox.json"
export TEST_IPYTHON_CASES_PATH="${repo_root}/test/e2e/cases/run_ipython_cell.json"
export TEST_SHELL_CASES_PATH="${repo_root}/test/e2e/cases/run_shell_command.json"

cd $repo_root/go/test/e2e
go clean -testcache
go test -v .
cd $repo_root

cd $repo_root/python
uv run pytest -o log_cli=true
cd $repo_root

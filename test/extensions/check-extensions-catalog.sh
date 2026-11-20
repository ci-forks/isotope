#!/usr/bin/env bash
# scripts/extensions-catalog.sh against a fake registry: the catalog names
# every published version of every extension under extensions/, pinned by
# digest, follows the registry's pages, and leaves out the image tags and the
# tags of another extension whose name starts the same way.
set -euo pipefail

repo=$(cd "$(dirname "$0")/../.." && pwd)
T=$(mktemp -d)
trap 'kill "${server:-}" 2>/dev/null || true; rm -rf "$T"' EXIT
fail=0
expect() {
    if [ "$2" = "$3" ]; then echo "ok   $1"; else echo "FAIL $1: got [$2], want [$3]"; fail=1; fi
}

# A copy of the tree with three extensions: htop (published twice), htop-extra
# (whose tags start with htop's prefix) and new (nothing published).
mkdir -p "$T/src/scripts" "$T/src/extensions"
cp "$repo/scripts/extensions-catalog.sh" "$T/src/scripts/"
for n in htop htop-extra new; do
    mkdir "$T/src/extensions/$n"
    echo 'FROM scratch' >"$T/src/extensions/$n/Dockerfile"
done
echo '{"title": "htop", "description": "Interactive process viewer"}' >"$T/src/extensions/htop/extension.json"

# The fake registry: two pages of tags; a manifest's digest is the hash of
# its tag.
cat >"$T/registry.py" <<'EOF'
import hashlib, http.server, json, sys
pages = [
    ["sway-v0.0.24", "sysext-htop-3.5.2-amd64", "sha256-abc"],
    ["sysext-htop-3.5.3-amd64", "sysext-htop-extra-1.0-amd64", "sysext-htop-3.5.3-arm64", "niri-vm-v0.0.24"],
]
class H(http.server.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        if self.path.startswith("/token"):
            self.send_response(404); self.end_headers(); return
        page = 1 if "last=" in self.path else 0
        body = json.dumps({"name": "o/r", "tags": pages[page]}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        if page == 0:
            self.send_header("Link", '</v2/o/r/tags/list?n=1000&last=sha256-abc>; rel="next"')
        self.end_headers(); self.wfile.write(body)
    def do_HEAD(self):
        tag = self.path.rsplit("/", 1)[1]
        self.send_response(200)
        self.send_header("Docker-Content-Digest", "sha256:" + hashlib.sha256(tag.encode()).hexdigest())
        self.end_headers()
s = http.server.HTTPServer(("127.0.0.1", 0), H)
print(s.server_port, flush=True)
s.serve_forever()
EOF
python3 "$T/registry.py" >"$T/port" & server=$!
for _ in $(seq 50); do [ -s "$T/port" ] && break; sleep 0.1; done
port=$(cat "$T/port")

IMAGE_REPO=ghcr.io/o/r REGISTRY_URL="http://127.0.0.1:$port" \
    "$T/src/scripts/extensions-catalog.sh" "$T/catalog.json" 2>"$T/log"
c="$T/catalog.json"
digest() { printf 'ghcr.io/o/r@sha256:%s' "$(printf '%s' "$1" | sha256sum | cut -d' ' -f1)"; }

expect "repo is the image repository" "$(jq -r .repo "$c")" "o/r"
expect "published extensions only, by name" "$(jq -r '[.layers[].name] | join(",")' "$c")" "htop,htop-extra"
expect "title and description from extension.json" "$(jq -r '.layers[0] | .title + "|" + .description' "$c")" "htop|Interactive process viewer"
expect "title defaults to the name" "$(jq -r '.layers[1] | .title + "|" + .description' "$c")" "htop-extra|"
expect "latest is the highest version" "$(jq -r '.layers[0].latest' "$c")" "3.5.3"
expect "versions newest first, from both pages" "$(jq -r '[.layers[0].tags[].tag] | join(",")' "$c")" "3.5.3,3.5.2"
expect "htop does not take htop-extra's tag" "$(jq -r '.layers[1].tags[0].tag' "$c")" "1.0"
expect "amd64 pinned by digest" "$(jq -r '.layers[0].tags[0].sysext.amd64.oci' "$c")" "$(digest sysext-htop-3.5.3-amd64)"
expect "arm64 pinned by digest" "$(jq -r '.layers[0].tags[0].sysext.arm64.oci' "$c")" "$(digest sysext-htop-3.5.3-arm64)"
expect "an older version has only its own arch" "$(jq -r '.layers[0].tags[1].sysext | keys | join(",")' "$c")" "amd64"
expect "an unpublished extension is reported" "$(grep -c 'new: nothing published yet' "$T/log")" 1

[ "$fail" -eq 0 ] && echo "all extensions catalog tests passed"
exit "$fail"

#!/usr/bin/env bash
# Write Isotope's system extension catalog: the JSON file `isotope ext` and
# `kairos-agent sysext` read (Kairos format, kairos-io/kairos
# sdk/extensions/catalog.go, plus the title and description `isotope ext`
# shows).
#
#   scripts/extensions-catalog.sh OUT.json
#
# The extensions are the directories under extensions/ (see
# extensions/README.md). Their versions are the registry tags
# sysext-<name>-<version>-<arch> that .github/workflows/extensions.yml pushes,
# each pinned here by digest. The image repository is public, so this reads
# the registry anonymously and needs no token.
#
# Environment:
#   IMAGE_REPO    the repository the extensions are pushed to
#                 (default ghcr.io/kairos-community/isotope)
#   REGISTRY_URL  where its registry API is (default https://<registry host>);
#                 the test points it at a local fake
set -euo pipefail

out="${1:?usage: $0 OUT.json}"
cd "$(dirname "$0")/.."

IMAGE_REPO="${IMAGE_REPO:-ghcr.io/kairos-community/isotope}"
registry_host="${IMAGE_REPO%%/*}"
repo_path="${IMAGE_REPO#*/}"
REGISTRY_URL="${REGISTRY_URL:-https://${registry_host}}"

# An anonymous pull token, as `docker pull` gets one. A registry that does not
# ask for one (the test's fake) returns no token, and no header is sent.
token=$(curl -fsS --retry 3 "${REGISTRY_URL}/token?scope=repository:${repo_path}:pull" 2>/dev/null |
    jq -r '.token // empty' 2>/dev/null || true)
auth=()
[ -n "$token" ] && auth=(-H "Authorization: Bearer ${token}")

# Every tag of the repository, following the Link header across pages.
tags=""
next="/v2/${repo_path}/tags/list?n=1000"
while [ -n "$next" ]; do
    headers=$(mktemp)
    body=$(curl -fsS --retry 3 "${auth[@]}" -D "$headers" "${REGISTRY_URL}${next}")
    tags+=$(jq -r '.tags // [] | .[]' <<<"$body")$'\n'
    next=$(sed -n 's/^[Ll]ink: *<\([^>]*\)>.*rel="next".*/\1/p' "$headers" | tr -d '\r')
    rm -f "$headers"
done

digest_of() {
    curl -fsS --retry 3 -I "${auth[@]}" \
        -H 'Accept: application/vnd.oci.image.manifest.v1+json, application/vnd.oci.image.index.v1+json, application/vnd.docker.distribution.manifest.v2+json' \
        "${REGISTRY_URL}/v2/${repo_path}/manifests/$1" |
        tr -d '\r' | sed -n 's/^[Dd]ocker-[Cc]ontent-[Dd]igest: *//p'
}

layers='[]'
for dir in extensions/*/; do
    name=$(basename "$dir")
    [ -f "${dir}Dockerfile" ] || continue
    meta="${dir}extension.json"
    title=$(jq -r '.title // empty' "$meta" 2>/dev/null || true)
    description=$(jq -r '.description // empty' "$meta" 2>/dev/null || true)

    # sysext-<name>-<version>-<arch>. The version starts with a digit, so
    # extension "foo" does not take the tags of an extension "foo-bar".
    entries='[]'
    while IFS= read -r tag; do
        [ -n "$tag" ] || continue
        rest="${tag#"sysext-${name}-"}"
        arch="${rest##*-}"
        version="${rest%-*}"
        digest=$(digest_of "$tag")
        if [ -z "$digest" ]; then
            echo "no digest for ${IMAGE_REPO}:${tag}" >&2
            exit 1
        fi
        entries=$(jq --arg v "$version" --arg a "$arch" --arg oci "${IMAGE_REPO}@${digest}" \
            '. + [{version: $v, arch: $a, oci: $oci}]' <<<"$entries")
    done < <(grep -E "^sysext-${name}-[0-9][^ ]*-(amd64|arm64)\$" <<<"$tags" || true)

    if [ "$entries" = '[]' ]; then
        echo "  ${name}: nothing published yet, left out" >&2
        continue
    fi

    versions=$(jq -r '[.[].version] | unique | .[]' <<<"$entries" | sort -V)
    latest=$(tail -n 1 <<<"$versions")
    layer=$(jq --arg name "$name" --arg title "${title:-$name}" --arg description "$description" \
        --arg latest "$latest" --arg versions "$versions" '
        . as $e
        | {name: $name, title: $title, description: $description, latest: $latest,
           tags: [($versions | split("\n") | reverse)[] as $v
                  | {tag: $v,
                     sysext: (reduce ($e[] | select(.version == $v)) as $x
                              ({}; .[$x.arch] = {oci: $x.oci}))}]}' <<<"$entries")
    layers=$(jq --argjson l "$layer" '. + [$l]' <<<"$layers")
    echo "  ${name}: $(tr '\n' ' ' <<<"$versions")(latest ${latest})" >&2
done

jq -n --arg repo "$repo_path" --argjson layers "$layers" \
    '{repo: $repo, generated: (now | todate), layers: ($layers | sort_by(.name))}' >"$out"
echo "wrote $(jq '.layers | length' "$out") extension(s) to ${out}" >&2

#!/usr/bin/env sh
set -eu

service_id=${1:?usage: deploy-render-service.sh <service-id> <commit-sha>}
commit_sha=${2:?usage: deploy-render-service.sh <service-id> <commit-sha>}
: "${RENDER_API_KEY:?RENDER_API_KEY is required}"

response=$(curl --fail-with-body --retry 3 --retry-all-errors --silent --show-error \
    --request POST "https://api.render.com/v1/services/$service_id/deploys" \
    --header "Authorization: Bearer $RENDER_API_KEY" \
    --header 'Content-Type: application/json' \
    --data "{\"commitId\":\"$commit_sha\"}")
deploy_id=$(printf '%s' "$response" | jq -r '.id // .deploy.id // empty')
if [ -z "$deploy_id" ]; then
    echo "Render did not return a deployment ID for service $service_id" >&2
    printf '%s\n' "$response" >&2
    exit 1
fi

echo "Render deployment queued: service=$service_id deploy=$deploy_id commit=$commit_sha"

attempt=0
while [ "$attempt" -lt 90 ]; do
    deploy=$(curl --fail-with-body --retry 3 --retry-all-errors --silent --show-error \
        --header "Authorization: Bearer $RENDER_API_KEY" \
        "https://api.render.com/v1/services/$service_id/deploys/$deploy_id")
    status=$(printf '%s' "$deploy" | jq -r '.status // .deploy.status // "unknown"')
    echo "Render deployment status: service=$service_id deploy=$deploy_id status=$status"

    case "$status" in
        live)
            if [ -n "${GITHUB_STEP_SUMMARY:-}" ]; then
                printf -- '- Render service `%s`: deployment `%s` is `live`\n' "$service_id" "$deploy_id" >> "$GITHUB_STEP_SUMMARY"
            fi
            exit 0
            ;;
        *failed*|canceled|cancelled|deactivated)
            echo "Render deployment failed: service=$service_id deploy=$deploy_id status=$status" >&2
            printf '%s\n' "$deploy" >&2
            exit 1
            ;;
    esac

    attempt=$((attempt + 1))
    sleep 10
done

echo "Timed out waiting for Render deployment: service=$service_id deploy=$deploy_id" >&2
exit 1

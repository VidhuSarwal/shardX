#!/usr/bin/env fish
# Check drive space by calling GET /api/drive/space
# Requires:
#   - BASE_URL env var (e.g. set -x BASE_URL http://localhost:8080)
#   - TOKEN env var (JWT)     (e.g. set -x TOKEN "<jwt>")

if not set -q BASE_URL
  echo "ERROR: BASE_URL is not set. Example: set -x BASE_URL http://localhost:8080"
  exit 1
end

if not set -q TOKEN
  echo "ERROR: TOKEN is not set. Obtain a JWT from /api/login and export it: set -x TOKEN <jwt>"
  exit 1
end

printf "\nTest 4: Checking available drive space\n\n"

set RESPONSE (curl -s -w "\n%{http_code}" -X GET "$BASE_URL/api/drive/space" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: application/json")

# Split body and status
set LINES (string split "\n" -- $RESPONSE)
set status (string trim (printf "%s" $LINES[-1]))
set body (string join "\n" $LINES[1..-2])

if test "$status" != "200"
  echo "HTTP $status"
  if test -n "$body"
    if which jq >/dev/null 2>&1
      echo $body | jq
    else
      echo $body
    end
  end
  exit 1
end

# Pretty print body if possible
if which jq >/dev/null 2>&1
  echo $body | jq
else
  echo $body
end

exit 0

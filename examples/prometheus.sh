#!/bin/sh
# Prometheus 질의의 답을 tidepic 피드로 쓰는 본보기 / example
#   sh prometheus.sh
#   tidetop --scene windows --data feed.json
# 15초마다 PROM_URL 에 QUERY 를 묻고, 시계열 하나를 값 하나로 feed.json 에 쓴다. 1 이면 live, 0 이면 fault 다.
# Grafana 를 거칠 때는 PROM_URL 에 자료원 프록시 주소를, HEADER_FILE 에 토큰 헤더 파일을 준다(README.md).
set -eu

PROM_URL=${PROM_URL:-http://localhost:9090}
PROM_URL=${PROM_URL%/}
QUERY=${QUERY:-up}
HEADER_FILE=${HEADER_FILE:-}
every=15

# 질의를 주소에 넣을 꼴로 미리 바꾼다. -G --data-urlencode 로 보내면 주소가 틀렸을 때 curl 이 까닭을 찍지 않는다
q=$(jq -rn --arg q "$QUERY" '$q | @uri')

# HEADER_FILE 을 줬을 때만 -H 를 넘긴다
set --
[ -z "$HEADER_FILE" ] || set -- -H "@$HEADER_FILE"

# Ctrl+C 로 멈추면 쓰다 만 파일을 지우고 SIGINT 로 끝낸다. 맥의 sh(bash 3.2)는 wait 도중에 멈추면 129 로 끝나서 부른 셸이 멈추지 않는다
trap 'rm -f feed.json.tmp; trap - INT; kill -INT $$' INT

wrote=
while :; do
  # 물음을 시작한 때부터 15초마다 묻는다. 시간 초과가 한 번 났다고 끊김이 되지 않게 한다
  sleep "$every" &
  # 실패하면 feed.json 을 그대로 둔다. ttl 의 세 배보다 오래 그대로면 tidetop 이 끊김으로 그린다
  # 다른 주소로 가라는 답(3xx)은 따라가지 않고 실패로 본다(curl 47). 토큰이 다른 주소로 가지 않는다
  if curl -fsS -m "$every" -L --max-redirs 0 "$@" "$PROM_URL/api/v1/query?query=$q" |
    jq -c --argjson ttl "$every" '
      .data
      | if .resultType != "vector" then error("the answer is not an instant vector; check QUERY") else .result end
      | if length == 0 then error("no series; check QUERY")
        elif length > 64 then error("\(length) series, a feed takes 64 at most; narrow QUERY")
        elif any(.[]; .value[1] != "0" and .value[1] != "1") then error("values must be 0 or 1; check QUERY")
        else . end
      | {spec: 1, meta: {ttl: $ttl}, values: [sort_by(.metric.instance, .metric)[] | .value[1] == "1"
          | {value: (if . then 1 else 0 end), to: 1, state: (if . then "live" else "fault" end)}]}
    ' > feed.json.tmp && [ -s feed.json.tmp ]
  then
    mv feed.json.tmp feed.json
    # 이번 실행에서 처음 쓸 때와, 실패한 뒤 다시 쓸 때만 알린다
    [ -n "$wrote" ] || echo "prometheus.sh: wrote feed.json" >&2
    wrote=1
  else
    rm -f feed.json.tmp
    echo "prometheus.sh: feed.json not updated" >&2
    wrote=
  fi
  wait
done

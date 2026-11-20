#!/bin/sh
# 네트워크 트래픽의 여유를 tidepic 피드로 쓰는 본보기 (리눅스) / example (Linux)
#   sh traffic.sh
#   tidetop --data traffic.json
# EVERY 초마다 장치가 받고 보낸 바이트를 재고, 더 바쁜 쪽이 MAX_MBIT 의 몇 %를 쓰는지 뺀 여유(%)를 OUT 에 쓴다.
# 여유가 10% 아래로 내려가면 tidetop 이 주의 줄을 그린다(from 10).
#   IFACE     잴 장치 (기본: 기본 길의 장치)
#   MAX_MBIT  가득으로 볼 속도, Mbit/s (기본: 장치가 알려 주는 연결 속도, 모르면 100)
#   EVERY     재는 간격, 초 (기본 2)
#   OUT       쓸 파일 (기본 traffic.json)

# 기본 길은 /proc/net/route 에서 목적지가 00000000 인 줄이다. ip 명령이 없는 서버에서도 읽힌다
IFACE=${IFACE:-$(awk '$2 == "00000000" {print $1; exit}' /proc/net/route 2>/dev/null)}
[ -n "$IFACE" ] || {
  echo "traffic.sh: no default route; set IFACE" >&2
  exit 1
}
dir=/sys/class/net/$IFACE
[ -r "$dir/statistics/rx_bytes" ] || {
  echo "traffic.sh: no such interface: $IFACE" >&2
  exit 1
}
# 가상 장치는 연결 속도를 모르거나(-1) 읽을 수 없다
speed=$(cat "$dir/speed" 2>/dev/null)
case $speed in '' | -* | 0 | *[!0-9]*) speed=100 ;; esac
MAX_MBIT=${MAX_MBIT:-$speed}
EVERY=${EVERY:-2}
OUT=${OUT:-traffic.json}

rx0=$(cat "$dir/statistics/rx_bytes")
tx0=$(cat "$dir/statistics/tx_bytes")
wrote=
while sleep "$EVERY"; do
  rx=$(cat "$dir/statistics/rx_bytes")
  tx=$(cat "$dir/statistics/tx_bytes")
  # 장치가 다시 붙어 누적 값이 0 부터 시작하면 차이가 음수가 된다. 그때는 여유 100 으로 적힌다
  awk -v rx=$((rx - rx0)) -v tx=$((tx - tx0)) -v every="$EVERY" -v max="$MAX_MBIT" 'BEGIN {
    used = (rx > tx ? rx : tx) * 8 / every / 1000000
    free = 100 - used / max * 100
    if (free < 0) free = 0
    if (free > 100) free = 100
    printf "{\"spec\":1,\"meta\":{\"ttl\":%s},\"values\":[{\"value\":%.1f,\"from\":10,\"to\":100,\"unit\":\"%%\"}]}\n", every, free
  }' > "$OUT.tmp" && mv "$OUT.tmp" "$OUT"
  [ -n "$wrote" ] || echo "traffic.sh: wrote $OUT ($IFACE, $MAX_MBIT Mbit/s)" >&2
  wrote=1
  rx0=$rx
  tx0=$tx
done

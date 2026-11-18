# Prometheus 와 Grafana

[English](README.md)

`prometheus.sh`는 15초마다 Prometheus 에 질의를 보내고, 그 답을 tidepic 피드 파일 `feed.json`으로 쓴다. tidetop 은 이 파일을 읽어 대상 하나를 창 하나로 그린다. 켜진 창은 Prometheus 가 수치를 긁어 온 대상이고, 검붉은 창은 긁어 오지 못한 대상이다.

tidetop 은 피드를 파일이나 https(주고받는 내용을 암호로 감싸는 방식) 주소에서만 읽는다. 묻는 일은 스크립트가 하고 답을 파일로 남기므로, Prometheus 가 https 가 아니어도 되고 밖에 내놓지 않아도 된다.

필요한 것은 `sh`, `curl`, `jq` 1.5 이상, tidetop 0.1.3 이상이다. 윈도우에서는 tidetop 도 WSL(윈도우에서 리눅스를 돌리는 기능) 안에 리눅스에서처럼 깔고, 스크립트와 tidetop 을 둘 다 그 안에서 돌린다.

## Prometheus 에 바로 묻기

스크립트를 받는다.

```bash
curl -fsSLO https://raw.githubusercontent.com/tidepark/tidetop/main/examples/prometheus.sh
```

Prometheus 가 이 컴퓨터의 9090 포트에 있으면 그대로 돌린다.

```bash
sh prometheus.sh
```

`prometheus.sh: wrote feed.json`(피드 파일을 썼다)이 찍히면, 같은 폴더에서 터미널을 하나 더 열고 tidetop 을 띄운다.

```bash
tidetop --scene windows --data feed.json
```

다른 곳의 Prometheus 에 묻거나 대상을 좁히려면 `PROM_URL`과 `QUERY`를 명령 앞에 붙인다.

```bash
PROM_URL=http://prometheus.example.com:9090 QUERY='up{job="node"}' sh prometheus.sh
```

## Grafana 를 거쳐 묻기

Prometheus 에는 닿지 않고 Grafana 에만 닿는다면, Grafana 의 자료원 프록시(Grafana 가 받은 질의를 Prometheus 에 대신 넘겨 주는 주소)로 같은 질의를 보낸다. 스크립트는 토큰을 헤더 파일(3번)에서 읽는데, 그러려면 curl 7.55 이상이 있어야 한다.

1. Grafana 의 Administration(관리) > Users and access(사용자 및 액세스 권한) > Service accounts(서비스 계정)에서 서비스 계정(사람 대신 프로그램이 쓰는 계정)을 만들고, 역할은 Viewer(보기만 하는 역할)로 고른다.
2. 그 계정에 토큰(비밀번호 대신 쓰는 긴 문자열)을 만들고 만료일을 정한다. 토큰은 만든 직후에 한 번만 보인다.
3. 토큰을 헤더 파일(curl 이 요청에 덧붙여 보낼 줄을 담은 파일)에 둔다. 아래 줄을 치고 먼저 Enter 를 누르면 아무것도 찍지 않고 기다린다. 그때 받은 토큰(`glsa_…`)을 붙여 넣고 Enter 를 누른다. 나만 읽을 수 있는 파일로 생긴다.

   ```bash
   (f="$HOME/.grafana-header"; umask 077; read -r t; rm -f "$f"; printf 'Authorization: Bearer %s\n' "$t" > "$f")
   ```

4. Prometheus 자료원의 uid(Grafana 가 자료원마다 붙인 고유한 이름)를 찾는다. `grafana.example.com`은 내 Grafana 주소로 바꾼다.

   ```bash
   curl -fsS -H @"$HOME/.grafana-header" https://grafana.example.com/api/datasources | jq -r '.[] | select(.type == "prometheus") | .uid + "  " + .name'
   ```

5. 찾은 uid(여기서는 `abc123`)를 주소에 넣어 돌린다.

   ```bash
   PROM_URL=https://grafana.example.com/api/datasources/proxy/uid/abc123 HEADER_FILE="$HOME/.grafana-header" sh prometheus.sh
   ```

403 이 나면 계정의 역할을 본다. 역할이 None(역할 없음)이면 403 이 나고, 자료원마다 권한을 두는 Grafana(Enterprise, Cloud)에서는 그 자료원의 Query(질의) 권한도 있어야 한다. 401 은 Grafana 가 맞는 토큰을 받지 못했다는 뜻이다. 토큰이 틀렸거나 기한이 지났거나, 명령에 `HEADER_FILE`이 빠졌다.

헤더 파일은 저장소에 넣거나 남에게 보내지 않는다. 그만 쓸 때는 Grafana 에서 토큰을 지우고 파일도 지운다.

## 알아 둘 것

- `up`이 0 이면 Prometheus 가 그 대상에서 수치를 긁어 오지 못했다는 뜻이다. 서버가 꺼졌다는 뜻과 같지는 않다. Prometheus 설정에서 빠진 대상은 창도 없어진다.
- 창은 왼쪽 위부터 줄마다 `instance`(대상의 주소) 순으로 놓인다. 글자 순이라서 `node10:9100`이 `node1:9100`보다 앞에 온다.
- 질의의 답은 0 이나 1 이어야 한다. 다른 값이 섞이면 스크립트는 파일을 쓰지 않고 `values must be 0 or 1`(값은 0 이나 1 이어야 한다)을 찍는다. 비교에 `bool`을 붙이면 어떤 수든 0 이나 1 이 된다. 아래 질의는 디스크가 90% 넘게 찬 서버의 창을 검붉게 한다.

  ```bash
  QUERY='node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"} > bool 0.1' sh prometheus.sh
  ```

- 피드에는 값을 64개까지 담는다. 대상이 그보다 많으면 스크립트는 목록을 잘라 내지 않고, 파일을 쓰지 않은 채 `… series, a feed takes 64 at most`(시계열이 …개다, 피드에는 64개까지 담는다)를 찍는다. 잘라 내면 65번째부터는 긁어 오지 못해도 보이지 않기 때문이다. `QUERY`를 좁힌다.
- 질의가 실패하면 스크립트는 `prometheus.sh: feed.json not updated`(피드 파일을 고치지 않았다)를 찍고 파일을 그대로 둔다. 다시 물을 수 있게 되면 `prometheus.sh: wrote feed.json`을 한 번 더 찍는다.
- 까닭은 대개 `feed.json not updated` 바로 앞 줄에 나온다. jq 가 찍은 줄 가운데 `check QUERY`(질의를 확인하라)나 `narrow QUERY`(질의를 좁히라)로 끝나는 줄은 jq 의 고장이 아니라 스크립트가 알리는 까닭이다.
  - `no series`: 질의에 맞는 시계열이 없다.
  - `the answer is not an instant vector`: 질의가 `up[1m]`처럼 범위를 내거나 `scalar(up)`처럼 수 하나를 낸다.
  - `curl: (22) The requested URL returned error: 400`: Prometheus 가 질의를 읽지 못했다. `QUERY`를 확인한다.
  - `curl: (47) Maximum (0) redirects followed`: 그 주소가 로그인 화면이나 Grafana 첫 화면 같은 다른 주소로 가라고 답한다. 스크립트는 따라가지 않는다. `PROM_URL`은 브라우저 주소창에서 복사한 화면 주소가 아니어야 하고, Grafana 를 거칠 때는 5번처럼 `/api/datasources/proxy/uid/` 뒤에 uid 가 붙는다. `http://`로 시작하면 `https://`로 바꿔 본다.
  - `parse error`: 답이 JSON(프로그램끼리 값을 주고받는 글의 형식)이 아니다. 웹 화면 같은 것이 왔으니 `PROM_URL`을 확인한다.
  - `curl: (7)`이나 `curl: (28)`: `PROM_URL`에 닿지 못했거나, 15초 안에 답이 오지 않았다.
- 파일이 45초(15초의 세 배)보다 오래 그대로면 tidetop 이 끊김으로 그린다. 장면이 잿빛이 되고, `i`로 상태 줄을 켜면 몇 초째 그대로인지 나온다. 색 없이 그릴 때는 잿빛이 보이지 않으니 상태 줄로 본다.
- Tab 으로 다른 장면에 가면 첫 값, 곧 첫 대상만 읽는다(tide 는 앞의 둘을 읽는다).
- tidetop 은 곁눈으로 보는 화면이다. 알림을 대신하지 않는다.

# Prometheus and Grafana

[한국어](README.ko.md)

`prometheus.sh` sends a query to Prometheus every 15 seconds and writes the answer as a tidepic feed file, `feed.json`. tidetop reads the file and draws one window for each target. A lit window is a target Prometheus scraped; a dark red window is one it could not scrape.

tidetop reads a feed only from a file or an https address. The script does the asking and leaves the answer in a file, so Prometheus does not need https, and it does not need to be open to the outside.

You need `sh`, `curl`, `jq` 1.5 or later and tidetop 0.1.3 or later. On Windows, install tidetop inside WSL as on Linux, and run both the script and tidetop there.

## Ask Prometheus directly

Get the script.

```bash
curl -fsSLO https://raw.githubusercontent.com/tidepark/tidetop/main/examples/prometheus.sh
```

If Prometheus is on port 9090 of this computer, run it as it is.

```bash
sh prometheus.sh
```

When it prints `prometheus.sh: wrote feed.json`, open another terminal in the same folder and start tidetop.

```bash
tidetop --scene windows --data feed.json
```

To ask a Prometheus elsewhere, or to narrow the targets, set `PROM_URL` and `QUERY` in front of the command.

```bash
PROM_URL=http://prometheus.example.com:9090 QUERY='up{job="node"}' sh prometheus.sh
```

## Ask through Grafana

If you can reach Grafana but not Prometheus, send the same query through Grafana's data source proxy. The script reads the token from a header file (step 3), and reading it needs curl 7.55 or later.

1. In Grafana, go to Administration > Users and access > Service accounts, add a service account and choose the Viewer role.
2. Add a token to the account and give it an expiry date. The token is shown only once, right after you add it.
3. Keep the token in a header file. Type the line below and press Enter first; it then waits without printing anything. Paste the token (`glsa_…`) and press Enter. The file is made readable only by you.

   ```bash
   (f="$HOME/.grafana-header"; umask 077; read -r t; rm -f "$f"; printf 'Authorization: Bearer %s\n' "$t" > "$f")
   ```

4. Find the uid of the Prometheus data source. Replace `grafana.example.com` with the address of your Grafana.

   ```bash
   curl -fsS -H @"$HOME/.grafana-header" https://grafana.example.com/api/datasources | jq -r '.[] | select(.type == "prometheus") | .uid + "  " + .name'
   ```

5. Put the uid (here `abc123`) in the address and run the script.

   ```bash
   PROM_URL=https://grafana.example.com/api/datasources/proxy/uid/abc123 HEADER_FILE="$HOME/.grafana-header" sh prometheus.sh
   ```

On 403, check the account's role. The role None gives 403, and where each data source has its own permissions (Enterprise, Cloud), the account also needs Query on that data source. 401 means Grafana did not get a valid token: the token is wrong or has expired, or `HEADER_FILE` is missing from the command.

Do not put the header file in a repository or send it to anyone. When you stop using it, delete the token in Grafana and delete the file.

## Good to know

- `up` is 0 when Prometheus could not scrape the target. That is not the same as the server being off. A target taken out of the Prometheus configuration loses its window.
- The windows run from the top left, row by row, in `instance` order. The order is by text, so `node10:9100` comes before `node1:9100`.
- The query must answer 0 or 1. If any other value comes back, the script does not write the file and prints `values must be 0 or 1`. Adding `bool` to a comparison turns any number into 0 or 1. This query turns a server's window dark red when its disk is more than 90% full.

  ```bash
  QUERY='node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"} > bool 0.1' sh prometheus.sh
  ```

- A feed holds up to 64 values. With more targets, the script does not cut the list; it writes no file and prints `… series, a feed takes 64 at most`. A cut list would hide every target after the 64th, even one that could not be scraped. Narrow `QUERY`.
- When a query fails, the script prints `prometheus.sh: feed.json not updated` and leaves the file as it is. When asking works again, it prints `prometheus.sh: wrote feed.json` once more.
- The reason is usually on the line just before `feed.json not updated`. A jq line that ends in `check QUERY` or `narrow QUERY` is the script's own reason, not a fault in jq.
  - `no series`: nothing matched the query.
  - `the answer is not an instant vector`: the query gives a range, such as `up[1m]`, or a single number, such as `scalar(up)`.
  - `curl: (22) The requested URL returned error: 400`: Prometheus could not read the query. Check `QUERY`.
  - `curl: (47) Maximum (0) redirects followed`: the address redirects to another one, such as a login page or Grafana's home page. The script does not follow redirects. `PROM_URL` must not be a page address copied from the browser; through Grafana, it ends in `/api/datasources/proxy/uid/` and the uid, as in step 5. If it starts with `http://`, try `https://`.
  - `parse error`: the answer is not JSON, for example a web page. Check `PROM_URL`.
  - `curl: (7)` or `curl: (28)`: the script could not reach `PROM_URL`, or no answer came within 15 seconds.
- If the file stays the same for more than 45 seconds (three times 15), tidetop draws it as stale: the scene turns grey, and with the status line on (`i`) it shows for how many seconds the file has stayed the same. Without colour there is no grey, so read the status line.
- If you press Tab to switch to another scene, tidetop reads only the first value, that is, the first target (tide reads the first two).
- tidetop is for a glance. It does not replace alerts.

# kavenegar-sms

`kavenegar-sms` is a small Alertmanager webhook receiver that sends **Prometheus alerts as SMS via Kavenegar**.

It is designed to run **alongside Prometheus and Alertmanager**, not as a generic SMS gateway:

```text
[ Prometheus ]  --->  [ Alertmanager ]  --->  [ kavenegar-sms ]  --->  [ Kavenegar API ]  --->  [ SMS to phones ]
```

- Prometheus evaluates alerting rules.
- Alertmanager groups and routes alerts.
- `kavenegar-sms` receives Alertmanager webhooks and converts selected alerts into SMS.
- SMS messages are sent through the **official Kavenegar HTTP API** to one or more phone numbers.

This service is **dedicated to Kavenegar** and its only purpose is to connect **Alertmanager → Kavenegar**.

GitHub repository:  
[https://github.com/mmakrami/kavenegar-sms](https://github.com/mmakrami/kavenegar-sms)

---

## Features

- Direct integration with **Alertmanager** as a `webhook_configs` receiver.
- HTTP endpoint: `POST /kavenegar`
- Alert filtering:
  - Only alerts with `severity="critical"`.
  - If `status` is present, only alerts with `status="firing"` are processed.
- Builds a readable SMS body from:
  - `labels.alertname`
  - `labels.instance`
  - `labels.severity`
  - `annotations.description` or `annotations.summary`
- Sends a **single bulk SMS** to multiple receivers using Kavenegar.
- Configuration is done entirely via **environment variables**.
- Single, small Go binary – easy to run next to Alertmanager on bare metal, VM, or container.

---

## Requirements

To use this in a real setup you need:

- A **Kavenegar** account with:
  - A valid **API key**
  - At least one approved **sender line** (for example: `90007060`)
- A running **Prometheus + Alertmanager** stack
- Go 1.21+ (if building from source)

---

## Architecture with Prometheus & Alertmanager

Typical flow:

1. Prometheus evaluates alerting rules (e.g. `InstanceDown`, `HighCPU`, …).
2. When a rule fires, Prometheus sends the alert to Alertmanager.
3. Alertmanager routes alerts based on labels (e.g. `severity="critical"`).
4. For critical alerts, Alertmanager calls the `kavenegar-sms` webhook:

   ```yaml
   receivers:
     - name: 'sms-kavenegar'
       webhook_configs:
         - url: 'http://your-kavenegar-sms-host:8082/kavenegar'
           send_resolved: false
   ```

5. `kavenegar-sms` parses the webhook payload, selects the alerts that should be notified, and sends an SMS via Kavenegar to all configured receivers.

This project is **not** a generic SMS service.  
It is explicitly built for **Kavenegar + Alertmanager**.

---

## Configuration

The service is configured entirely via environment variables.

### Required environment variables

- `KAVENEGAR_API_KEY`  
  Your Kavenegar API key (from the Kavenegar panel).

- `KAVENEGAR_SENDER`  
  The sender line registered and approved in Kavenegar (for example `90007060`).

- `KAVENEGAR_RECEIVERS`  
  Comma-separated list of receiver phone numbers. Example:

  ```text
  09120000000,09350000000,09130000000
  ```

### Optional environment variables

- `LISTEN_ADDR`  
  HTTP listen address for the webhook server.  
  Default: `:8082`

---

### Example configuration (shell)

```bash
export KAVENEGAR_API_KEY="YOUR_REAL_KAVENEGAR_API_KEY"
export KAVENEGAR_SENDER="90007060"
export KAVENEGAR_RECEIVERS="09120000000,09350000000"
export LISTEN_ADDR=":8082"

go build -o kavenegar-sms ./cmd/kavenegar-sms
./kavenegar-sms
```

Expected log:

```text
2025/11/19 15:07:01 kavenegar-sms listening on :8082
```

---

## Building from source

Tested with Go 1.21+.

```bash
git clone https://github.com/mmakrami/kavenegar-sms.git
cd kavenegar-sms

go fmt ./...
go build -o kavenegar-sms ./cmd/kavenegar-sms
```

This produces a single binary named `kavenegar-sms`.

---

## Running next to Alertmanager

Typically you run this service:

- On the same machine as Alertmanager, or
- On a nearby VM/container in the same network.

Example run:

```bash
export KAVENEGAR_API_KEY="YOUR_REAL_KAVENEGAR_API_KEY"
export KAVENEGAR_SENDER="90007060"
export KAVENEGAR_RECEIVERS="09120000000,09350000000"
export LISTEN_ADDR=":8082"

./kavenegar-sms
```

If Alertmanager and `kavenegar-sms` run on the same host:

- Use `http://localhost:8082/kavenegar` in Alertmanager, or
- `http://127.0.0.1:8082/kavenegar`

If they are on different hosts, use the appropriate IP/hostname.

---

## Alertmanager configuration

Minimal Alertmanager configuration snippet to send **critical** alerts to Kavenegar via this service:

```yaml
route:
  group_by: ['alertname']
  receiver: 'sms-kavenegar'

  routes:
    - receiver: 'sms-kavenegar'
      matchers:
        - 'severity="critical"'
      continue: false

receivers:
  - name: 'sms-kavenegar'
    webhook_configs:
      - url: 'http://your-kavenegar-sms-host:8082/kavenegar'
        send_resolved: false
```

Notes:

- `send_resolved: false` is recommended for SMS (less noise).
- You can combine this with other receivers (Telegram, Mattermost, Opsgenie, etc).

Example combined route:

```yaml
route:
  group_by: ['alertname']
  receiver: 'telegram'        # default

  routes:
    - receiver: 'sms-kavenegar'
      matchers:
        - 'severity="critical"'
      continue: true

    - receiver: 'telegram'
      matchers:
        - 'severity="critical"'
      continue: true
```

---

## How alerts are filtered

The alert filtering logic is intentionally simple and explicit:

- Only alerts with label `severity="critical"` are considered.
- If the alert has a `status` field, only `status="firing"` is processed.
- Resolved and non-critical alerts are ignored.

Relevant code (simplified):

```go
func shouldNotify(a Alert) bool {
    severity := a.Labels["severity"]
    if severity != "critical" {
        return false
    }
    if a.Status != "" && a.Status != "firing" {
        return false
    }
    return true
}
```

---

## SMS message format

The SMS body is built from the alert labels and annotations:

```text
ALERT <alertname> [<severity>]
Instance: <instance>
<description or summary>
```

Example:

```text
ALERT InstanceDown [critical]
Instance: example-prometheus:9090
Prometheus example-prometheus:9090 has been unreachable for more than 5 minutes.
```

Details:

- If `annotations.description` is empty, `annotations.summary` is used.
- If both are empty, `"No description"` is used.
- Message length is trimmed to a reasonable size; Kavenegar itself will split long messages into multiple SMS segments.

---

## Local testing without Prometheus/Alertmanager

For quick local testing, a sample JSON payload is provided at `testdata/alertmanager_webhook.json`.

1. Start the service:

   ```bash
   export KAVENEGAR_API_KEY="INVALID_OR_REAL_API_KEY"
   export KAVENEGAR_SENDER="90007060"
   export KAVENEGAR_RECEIVERS="09120000000"
   export LISTEN_ADDR=":8082"

   ./kavenegar-sms
   ```

2. In another terminal, send a test webhook:

   ```bash
   curl -X POST "http://localhost:8082/kavenegar"      -H "Content-Type: application/json"      --data @testdata/alertmanager_webhook.json
   ```

If everything is wired correctly:

- HTTP response:

  ```json
  {"msg":"alerts processed"}
  ```

- In the service logs you will see either:
  - A successful send, or
  - An error from Kavenegar (e.g. 403 if the API key is invalid).

---

## Production considerations

For production use:

- Run the service as a **non-root** user.
- To restrict access:
  - Use `LISTEN_ADDR="127.0.0.1:8082"` and expose it only via a reverse proxy (nginx, HAProxy, etc.), or
  - Restrict access to `/kavenegar` at the network level (firewall, security groups).
- Ensure only Alertmanager can reach the `/kavenegar` endpoint.
- Monitor logs for Kavenegar API errors (invalid key, rate limits, quota issues).
- Do **not** commit your `KAVENEGAR_API_KEY` or real phone numbers to Git or public repos.

---

## Limitations

- This service is **fully dedicated to Kavenegar** – other SMS providers are not supported.
- Filtering logic is intentionally simple (critical + firing only).
- There is no built-in authentication on `/kavenegar`; access control should be handled via network or reverse proxy.

---

## Development

Standard Go workflow:

```bash
go fmt ./...
go test ./...   # when tests are added
go build -o kavenegar-sms ./cmd/kavenegar-sms
```

Issues and pull requests:  
[https://github.com/mmakrami/kavenegar-sms](https://github.com/mmakrami/kavenegar-sms)

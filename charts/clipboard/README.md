# Clipboard Helm Chart

Shared clipboard service with real-time WebSocket sync, from
[ConnorsApps/clipboard](https://github.com/ConnorsApps/clipboard). Plain Helm
templates; no subcharts.

```bash
helm repo add clipboard https://connorsapps.github.io/clipboard
helm upgrade --install clipboard clipboard/clipboard -n clipboard --create-namespace \
  --set config.passwords='my-password'
```

## What it creates

| Resource | Notes |
|---|---|
| Deployment | One replica, `Recreate` strategy. WebSocket state and the in-memory token store live in the process, and the files volume is ReadWriteOnce, so it cannot scale out. |
| Service | `ClusterIP`, port 8080. |
| Secret | `CLIPBOARD_PASSWORDS` and (optionally) `MONGODB_URI`. Skipped when `existingSecret` is set. |
| PersistentVolumeClaim | 10Gi for uploaded files, mounted at `/data`. Skipped with `persistence.existingClaim` or `persistence.enabled: false`. |
| HTTPRoute | Gateway API route. **Disabled by default**; enable with `httpRoute.enabled`. |

## Configuration

Every option is documented in [values.yaml](https://github.com/ConnorsApps/clipboard/blob/main/charts/clipboard/values.yaml).

- `config.passwords` is required unless `existingSecret` is set. Without a password the app would fall back to `1234`, so the chart refuses to render.
- `config.mongodbUri` persists auth tokens across restarts; without it, everyone logs in again after each pod restart. `config.tokenExpiry` (`30d`, `never`, ...) sets how long those tokens last.
- `existingSecret` names a Secret you manage with the key `CLIPBOARD_PASSWORDS` (and optionally `MONGODB_URI`).

### HTTPRoute

```yaml
httpRoute:
  enabled: true
  parentRef:
    name: my-gateway
    namespace: gateway-system  # omit if the Gateway is in the release namespace
    sectionName: https         # optional listener name
  hostnames:
    - clipboard.example.com
```

The app uses long-lived WebSockets (`/ws`) and large uploads (`/api/uploads`),
so make sure the Gateway's request timeouts allow both.

## Upgrading from 1.x

Chart 2.0 dropped [HULL](https://github.com/vidispine/hull). See
[Upgrading from chart 1.x](https://github.com/ConnorsApps/clipboard#upgrading-from-chart-1x)
for the old-to-new values table and how to keep the existing PVC with
`persistence.existingClaim`. A leftover `hull:` block in your values makes the
chart fail instead of being silently ignored.

## Releasing (maintainers)

Bump `version` in `Chart.yaml` and merge to `main`. The
[Release Helm Chart](https://github.com/ConnorsApps/clipboard/blob/main/.github/workflows/release-charts.yml)
workflow tests the chart, then uses
[chart-releaser](https://github.com/helm/chart-releaser-action) to publish a
GitHub Release and update the index on the `gh-pages` branch. Merging without a
version bump releases nothing.

## Development

```bash
make helm-test                                      # lint + render every scenario in tests/values/
helm template test ./charts/clipboard --set config.passwords=x
```

Don't change how object names are generated (`templates/_helpers.tpl`): the PVC
holds user data, and a different name on upgrade orphans it.

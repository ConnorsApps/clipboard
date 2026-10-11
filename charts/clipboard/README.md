# Clipboard Helm Chart

Chart for [ConnorsApps/clipboard](https://github.com/ConnorsApps/clipboard). Plain templates, no subcharts.

```bash
helm repo add clipboard https://connorsapps.github.io/clipboard
helm upgrade --install clipboard clipboard/clipboard -n clipboard --create-namespace \
  --set config.passwords='my-password'
```

Several passwords: use a values file (`config.passwords: "one,two"`); in `--set`, escape the comma (`one\,two`).

## Resources

| Resource | Notes |
|---|---|
| Deployment | 1 replica, `Recreate` (tokens/WebSockets are per-process, PVC is RWO) |
| Service | `ClusterIP`, port 8080 |
| Secret | `CLIPBOARD_PASSWORDS`, optional `MONGODB_URI`; skipped with `existingSecret` |
| PersistentVolumeClaim | 10Gi at `/data`; skipped with `persistence.existingClaim` or `persistence.enabled: false` |
| HTTPRoute | Gateway API; **off by default** (`httpRoute.enabled`) |

All options: [values.yaml](https://github.com/ConnorsApps/clipboard/blob/main/charts/clipboard/values.yaml).
Helm checks them against [values.schema.json](https://github.com/ConnorsApps/clipboard/blob/main/charts/clipboard/values.schema.json), so a mistyped key fails the install.
`config.passwords` (or `existingSecret`) is required; the app would otherwise accept `1234`.

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/ConnorsApps/clipboard/main/charts/clipboard/values.schema.json
httpRoute:
  enabled: true
  parentRef: {name: my-gateway, namespace: gateway-system, sectionName: https}
  hostnames: [clipboard.example.com]
```

## Releasing

Bump `version` in `Chart.yaml` and merge to `main`; the workflow publishes via
[chart-releaser](https://github.com/helm/chart-releaser-action) (GitHub Release + `gh-pages` index).
One-time: Settings → Pages → Source → `gh-pages` (the first run creates the branch).

`make helm-test` lints and renders every scenario in `tests/values/`.
Don't change how names are generated (`templates/_helpers.tpl`): the PVC holds user data.

# Clipboard Helm Chart

This directory contains the Helm chart for the Clipboard application, built using [HULL](https://github.com/vidispine/hull).

## GitHub as Helm Repository

This chart is published directly from GitHub using packaged chart files and an index. Users can add this repository and install the chart without needing a separate chart repository server.

### For Maintainers: Publishing Updates

When you make changes to the chart:

```bash
# 1. Update the version in Chart.yaml
vim Chart.yaml  # Increment version

# 2. Package the chart
cd /path/to/clipboard
./package-chart.sh

# 3. Commit and push
git add helm/*.tgz helm/index.yaml helm/
git commit -m "Release chart version X.Y.Z"
git push
```

The script will:
- Update chart dependencies
- Package the chart into a `.tgz` file
- Update the `index.yaml` with the new version
- Show instructions for committing

### For Users: Installing the Chart

Add the repository:
```bash
helm repo add clipboard https://raw.githubusercontent.com/ConnorsApps/clipboard/main/helm/
helm repo update
```

Install the chart:
```bash
helm install my-clipboard clipboard/clipboard
```

Search available versions:
```bash
helm search repo clipboard
```

## Chart Configuration

This chart uses HULL for object configuration. All configuration is done through the `values.yaml` file.

### Required Configuration

Create a `values.yaml` file with your settings:

```yaml
hull:
  objects:
    secret:
      clipboard:
        data:
          CLIPBOARD_PASSWORDS:
            inline: "your-password-here"
          MONGODB_URI:
            inline: "mongodb://your-mongodb-uri"  # Optional
```

### Chart Resources

The chart creates:
- **Deployment**: Application pods
- **Service**: ClusterIP service (port 8080)
- **Secret**: Environment variables (password, MongoDB URI, etc.)
- **PersistentVolumeClaim**: 10Gi storage for files
- **ServiceAccount, Role, RoleBinding**: RBAC resources

### Storage Configuration

Customize the PVC:

```yaml
hull:
  objects:
    persistentvolumeclaim:
      files:
        storageClassName: "your-storage-class"
        resources:
          requests:
            storage: 50Gi  # Adjust size as needed
```

### Resource Limits

Adjust container resources:

```yaml
hull:
  objects:
    deployment:
      main:
        pod:
          containers:
            main:
              resources:
                limits:
                  cpu: 500m
                  memory: 512Mi
                requests:
                  cpu: 100m
                  memory: 128Mi
```

## Development

Test the chart locally:

```bash
# Render templates to verify
helm template test ./helm

# Install to a test cluster
helm install test ./helm -f my-values.yaml --dry-run --debug
```

## HULL Library

This chart leverages the HULL library for streamlined Kubernetes object configuration:
- Full Kubernetes API access without custom templates
- JSON schema validation
- Automatic metadata management
- Simplified ConfigMap/Secret handling

Learn more: https://github.com/vidispine/hull

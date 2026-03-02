# Clipboard

Shared clipboard service with WebSocket sync for real-time clipboard sharing across devices.

## Features

- 📋 Real-time clipboard synchronization via WebSocket
- 📁 File upload and sharing with resumable uploads (tusd)
- 🔐 Password-based authentication
- 💾 Persistent auth token storage with MongoDB (optional) or in-memory
- 🚀 Easy deployment with Helm
- 💩 Vibe coded AI slop

## Screenshots

<div align="center">
  <img src="assets/screenshot-mobile-clipboard.png" alt="Clipboard View" width="300"/>
  <img src="assets/screenshot-mobile-files.png" alt="Files View" width="300"/>
</div>

## Deployment

#### Add the Helm Repository

```bash
helm repo add clipboard https://raw.githubusercontent.com/ConnorsApps/clipboard/main/helm/
helm repo update
```

#### Install the Chart

```bash
# Create a values file with your configuration
cat > my-values.yaml <<EOF
# yaml-language-server: $schema=https://raw.githubusercontent.com/vidispine/hull/refs/heads/main/hull/values.schema.json

hull:
  objects:
    persistentvolumeclaim:
      files:
        storageClassName: "your-storage-class"
        resources:
          requests:
            storage: 10Gi
    secret:
      clipboard:
        data:
          CLIPBOARD_PASSWORDS:
            inline: "your-secure-password,my-other-users-password"
          MONGODB_URI:
            inline: "mongodb://your-mongo-uri"  # Optional, leave empty for in-memory
EOF

# Install the chart
helm install my-clipboard clipboard/clipboard -f my-values.yaml
```

#### Configuration

The chart uses [HULL](https://github.com/vidispine/hull) for simplified Kubernetes object configuration. All configuration is done via the `values.yaml` file.

Key configuration options:

- `CLIPBOARD_PASSWORDS`: Password(s) for authentication (required), The default is `1234`. Seperate multiple user's passwords with commas. Each user is distinguished by their password.
- `MONGODB_URI`: MongoDB connection string (optional). If not set, the app uses an in-memory token store.
- `FILES_DIR`: Directory for file storage (default: `/data`, mounted from PVC)

#### Authentication and token storage

- **In-memory store** (when `MONGODB_URI` is not set): Auth tokens are kept only in process memory. **All users must log in again after each server restart** (e.g. deploy or pod restart). Use this for single-instance or dev only.
- **MongoDB store**: Set `MONGODB_URI` for persistent tokens so users stay logged in across restarts. Tokens do not expire by default. The optional script `scripts/add-mongo-index.sh` can add a TTL index so MongoDB auto-expires tokens after 30 days; only run it if you want token expiry.
- Transient token-store errors (e.g. MongoDB timeouts) are returned as 503; the frontend retries and does not clear the token, so users are not logged out by brief backend issues.

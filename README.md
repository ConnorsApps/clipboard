# Clipboard

Shared clipboard service with WebSocket sync for real-time clipboard sharing across devices.

## Features

- 📋 Real-time clipboard synchronization via WebSocket
- 📁 File upload and sharing with resumable uploads (tusd)
- 🔐 Password-based authentication
- 💾 Persistent storage with MongoDB (optional) or in-memory
- 🚀 Easy deployment with Helm
- 💩 Vibe coded AI slop

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
hull:
  objects:
    persistentvolumeclaim:
      files:
        storageClassName: "your-storage-class"
    secret:
      clipboard:
        data:
          CLIPBOARD_PASSWORD:
            inline: "your-secure-password"
          MONGODB_URI:
            inline: "mongodb://your-mongo-uri"  # Optional, leave empty for in-memory
EOF

# Install the chart
helm install my-clipboard clipboard/clipboard -f my-values.yaml
```

#### Configuration

The chart uses [HULL](https://github.com/vidispine/hull) for simplified Kubernetes object configuration. All configuration is done via the `values.yaml` file.

Key configuration options:

- `CLIPBOARD_PASSWORD`: Password for authentication (required)
- `MONGODB_URI`: MongoDB connection string (optional, uses in-memory if not set)
- `FILES_DIR`: Directory for file storage (default: `/data`, mounted from PVC)

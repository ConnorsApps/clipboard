#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CHART_DIR="${SCRIPT_DIR}/helm"
PACKAGES_DIR="${SCRIPT_DIR}/helm"

cd "${SCRIPT_DIR}"

echo "📦 Packaging Helm chart..."

# Update dependencies first
echo "Updating chart dependencies..."
helm dependency update "${CHART_DIR}"

# Package the chart into the helm directory
echo "Packaging chart..."
helm package "${CHART_DIR}" -d "${PACKAGES_DIR}"

# Generate/update the index.yaml
echo "Updating repository index..."
helm repo index "${PACKAGES_DIR}" --url https://raw.githubusercontent.com/ConnorsApps/clipboard/main/helm/

echo ""
echo "✅ Chart packaged successfully!"
echo ""
echo "📋 Next steps:"
echo "   1. git add helm/*.tgz helm/index.yaml"
echo "   2. git commit -m 'Update Helm chart package'"
echo "   3. git push"
echo ""
echo "📖 Users can then install with:"
echo "   helm repo add clipboard https://raw.githubusercontent.com/ConnorsApps/clipboard/main/helm/"
echo "   helm repo update"
echo "   helm install my-clipboard clipboard/clipboard"

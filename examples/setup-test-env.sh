#!/bin/bash

# Create test namespaces
kubectl create namespace app1
kubectl create namespace app2
kubectl create namespace monitoring

# Create source ConfigMap
kubectl apply -f source-configmap.yaml

# Create a target ConfigMap with labels for testing selector-based sync
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: target-config
  namespace: app1
  labels:
    app: my-app
    sync-enabled: "true"
data:
  local-config.properties: |
    local.setting=value
EOF

# Apply the ConfigMapSyncer CRs
kubectl apply -f configmapsyncer_sample.yaml
kubectl apply -f configmapsyncer_with_selector.yaml

echo "Test environment setup complete!"
echo "To check the status of the ConfigMapSyncer resources:"
echo "kubectl get configmapsyncers -o wide"
echo ""
echo "To check the synced ConfigMaps:"
echo "kubectl get configmaps -n app1"
echo "kubectl get configmaps -n app2"
echo "kubectl get configmaps -n monitoring" 
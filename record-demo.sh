#!/bin/bash

# This script demonstrates the ConfigMap Sync Controller in action
# It requires asciinema to be installed: https://asciinema.org/docs/installation

# Check if asciinema is installed
if ! command -v asciinema &> /dev/null; then
    echo "asciinema is not installed. Please install it first."
    echo "Visit https://asciinema.org/docs/installation for installation instructions."
    exit 1
fi

# Start recording
echo "Starting demo recording..."
asciinema rec -t "ConfigMap Sync Controller Demo" configmap-sync-controller-demo.cast

# The following commands will be recorded
echo "# ConfigMap Sync Controller Demo"
echo "# This demo shows how the ConfigMap Sync Controller works"
echo ""
sleep 2

echo "# First, let's create the test namespaces"
kubectl create namespace app1
kubectl create namespace app2
kubectl create namespace monitoring
sleep 3

echo "# Now, let's create a source ConfigMap in the default namespace"
kubectl apply -f examples/source-configmap.yaml
echo "---"
kubectl apply -f examples/source-configmap.yaml -n app1
echo "---"
kubectl apply -f examples/source-configmap.yaml -n app2
echo "---"
kubectl apply -f examples/source-configmap.yaml -n monitoring
sleep 2

echo "# Let's check the master source ConfigMap"
kubectl get configmap source-config -o yaml
echo "---"
echo "# Let's check the target ConfigMap in app1 namespace"
kubectl get configmap source-config -n app1 -o yaml
echo "---"
echo "# Let's check the target ConfigMap in app2 namespace"
kubectl get configmap source-config -n app2 -o yaml
echo "---"
echo "# Let's check the target ConfigMap in monitoring namespace"
kubectl get configmap source-config -n monitoring -o yaml

sleep 5

echo "# Now, let's create a ConfigMapSyncer resource to sync the ConfigMap to other namespaces"
kubectl apply -f examples/configmapsyncer_sample.yaml
sleep 2

echo "# Let's check the status of the ConfigMapSyncer resource"
kubectl get configmapsyncers
sleep 2
kubectl describe configmapsyncer sample-configmap-syncer
sleep 5

echo "# Now, let's check if the ConfigMap was synced to the target namespaces"
echo "# Checking app1 namespace:"
echo "=== Before sync state in app1 namespace ==="
kubectl get configmap source-config -n app1 -o yaml
sleep 2

echo "# Checking app2 namespace:"
echo "=== Before sync state in app2 namespace ==="
kubectl get configmap source-config -n app2 -o yaml
sleep 2

echo "# Checking monitoring namespace:"
echo "=== Before sync state in monitoring namespace ==="
kubectl get configmap source-config -n monitoring -o yaml
sleep 5

echo "# Now, let's update the source ConfigMap and see if the changes are propagated"
echo "=== Updating source ConfigMap with new data ==="
kubectl patch configmap source-config --type=merge -p '{"data":{"new-key":"new-value"}}'
sleep 5

echo "# Let's check if the changes were propagated to the target namespaces"
echo "# Verifying sync in app1 namespace:"
echo "=== Checking sync status in app1 namespace ==="
kubectl get configmap source-config -n app1 -o yaml
if kubectl get configmap source-config -n app1 -o jsonpath='{.data.new-key}' | grep -q "new-value"; then
    echo "✅ ConfigMap successfully synced to app1 namespace"
else
    echo "❌ ConfigMap sync failed in app1 namespace"
fi
sleep 2

echo "# Verifying sync in app2 namespace:"
echo "=== Checking sync status in app2 namespace ==="
kubectl get configmap source-config -n app2 -o yaml
if kubectl get configmap source-config -n app2 -o jsonpath='{.data.new-key}' | grep -q "new-value"; then
    echo "✅ ConfigMap successfully synced to app2 namespace"
else
    echo "❌ ConfigMap sync failed in app2 namespace"
fi
sleep 2

echo "# Verifying sync in monitoring namespace:"
echo "=== Checking sync status in monitoring namespace ==="
kubectl get configmap source-config -n monitoring -o yaml
if kubectl get configmap source-config -n monitoring -o jsonpath='{.data.new-key}' | grep -q "new-value"; then
    echo "✅ ConfigMap successfully synced to monitoring namespace"
else
    echo "❌ ConfigMap sync failed in monitoring namespace"
fi
sleep 5

echo "# Let's try the label selector-based sync"
kubectl apply -f examples/configmapsyncer_with_selector.yaml
sleep 2

echo "# Create a ConfigMap with matching labels in app1 namespace"
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
sleep 2

echo "# Let's check if the ConfigMap was synced based on labels"
echo "=== Checking label-based sync status ==="
kubectl get configmap target-config -n app1 -o yaml
if kubectl get configmap target-config -n app1 -o jsonpath='{.metadata.labels.sync-enabled}' | grep -q "true"; then
    echo "✅ Label-based ConfigMap sync verification successful"
else
    echo "❌ Label-based ConfigMap sync verification failed"
fi
sleep 5

echo "# Demo completed!"
sleep 2

kubectl delete configmap source-config
kubectl delete namespace app1
kubectl delete namespace app2
kubectl delete namespace monitoring

# End recording
# The recording will end when you press Ctrl+D

echo "Demo recording completed. The recording is saved as configmap-sync-controller-demo.cast"
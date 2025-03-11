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
kubectl apply -f source-configmap.yaml
sleep 2

echo "# Let's check the source ConfigMap"
kubectl get configmap source-config -o yaml
sleep 5

echo "# Now, let's create a ConfigMapSyncer resource to sync the ConfigMap to other namespaces"
kubectl apply -f configmapsyncer_sample.yaml
sleep 2

echo "# Let's check the status of the ConfigMapSyncer resource"
kubectl get configmapsyncers
sleep 2
kubectl describe configmapsyncer sample-configmap-syncer
sleep 5

echo "# Now, let's check if the ConfigMap was synced to the target namespaces"
echo "# Checking app1 namespace:"
kubectl get configmap -n app1
sleep 2
kubectl get configmap source-config -n app1 -o yaml
sleep 5

echo "# Checking app2 namespace:"
kubectl get configmap -n app2
sleep 2
kubectl get configmap source-config -n app2 -o yaml
sleep 5

echo "# Checking monitoring namespace:"
kubectl get configmap -n monitoring
sleep 2
kubectl get configmap source-config -n monitoring -o yaml
sleep 5

echo "# Now, let's update the source ConfigMap and see if the changes are propagated"
kubectl patch configmap source-config --type=merge -p '{"data":{"new-key":"new-value"}}'
sleep 5

echo "# Let's check if the changes were propagated to the target namespaces"
echo "# Checking app1 namespace:"
kubectl get configmap source-config -n app1 -o yaml
sleep 5

echo "# Let's try the label selector-based sync"
kubectl apply -f configmapsyncer_with_selector.yaml
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
kubectl get configmap target-config -n app1 -o yaml
sleep 5

echo "# Finally, let's clean up"
kubectl delete configmapsyncer --all
kubectl delete configmap source-config
kubectl delete configmap target-config -n app1
kubectl delete namespace app1
kubectl delete namespace app2
kubectl delete namespace monitoring

echo "# Demo completed!"
sleep 2

# End recording
# The recording will end when you press Ctrl+D

echo "Demo recording completed. The recording is saved as configmap-sync-controller-demo.cast"
echo "You can play it with: asciinema play configmap-sync-controller-demo.cast"
echo "Or upload it to asciinema.org with: asciinema upload configmap-sync-controller-demo.cast" 
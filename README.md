# ConfigMap Sync Controller

A Kubernetes controller that synchronizes ConfigMaps across multiple namespaces with configurable sync intervals and merge strategies.

## Architecture

### Components and Their Relationships

1. **Controller Deployment**

   - The controller runs as a deployment in the `configmap-sync-controller-system` namespace
   - It watches for ConfigMapSyncer resources across all namespaces
   - Handles the actual synchronization of ConfigMaps based on ConfigMapSyncer specifications

2. **Custom Resource Definition (CRD)**

   - The ConfigMapSyncer CRD defines the schema for ConfigMapSyncer resources
   - Installed using `make install` command
   - Extends the Kubernetes API to understand ConfigMapSyncer resources

3. **ConfigMapSyncer Resources**
   - These are instances of the ConfigMapSyncer CRD
   - Should be stored in your application's configuration directory (e.g., `config/samples/` or `manifests/`)
   - Can be created in any namespace (the controller watches all namespaces)
   - Define the sync rules for specific ConfigMaps

### Directory Structure

```
configmap-sync-controller/
├── config/                    # Controller configuration
│   ├── crd/                  # Custom Resource Definitions
│   ├── rbac/                 # RBAC permissions
│   └── manager/              # Controller manager configuration
├── manifests/                # Your ConfigMapSyncer resources (recommended)
│   ├── source-config.yaml    # Source ConfigMap
│   └── syncer.yaml          # ConfigMapSyncer configuration
└── ...
```

### Workflow

1. Deploy the controller (`make deploy`)
2. Create your ConfigMap resources
3. Create ConfigMapSyncer resources to define sync rules
4. The controller automatically detects and processes the ConfigMapSyncer resources

## Features

- Synchronize ConfigMaps across multiple namespaces
- Configurable sync interval (default: 3 seconds)
- Support for merge strategies
- Real-time updates when source ConfigMap changes
- Kubernetes operator pattern implementation

## Prerequisites

- Kubernetes cluster (v1.19+)
- kubectl configured to access your cluster
- Docker installed for building images
- Go 1.19+ (for development)

## Installation

### 1. Clone the Repository

```bash
git clone https://github.com/devShahriar/configmap-sync-controller.git
cd configmap-sync-controller
```

### 2. Build and Deploy the Controller

```bash
# Build the controller image
make docker-build IMG=configmap-sync-controller:latest

# Install CRDs
make install

# Deploy the controller
make deploy IMG=configmap-sync-controller:latest
```

To verify the installation:

```bash
# Check if the controller is running
kubectl get pods -n configmap-sync-controller-system

# Check if CRDs are installed
kubectl get crd | grep configmapsyncers
```

## Usage

### 1. Namespace Setup

First, create the required namespaces for your configuration:

```bash
# Create namespaces for your applications
kubectl create namespace app1
kubectl create namespace app2
kubectl create namespace monitoring

# Verify namespaces are created
kubectl get namespaces | grep -E 'app1|app2|monitoring'
```

### 2. Source ConfigMap Management

The source ConfigMap is the master configuration that will be synchronized to other namespaces.

#### Create Source ConfigMap in Default Namespace

```yaml
# source-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: source-config
  namespace: default # Source namespace
data:
  app.properties: |
    log.level=INFO
    max.connections=100
    timeout=30s
  db.properties: |
    db.url=jdbc:mysql://localhost:3306/mydb
    db.user=admin
```

Apply the source ConfigMap:

```bash
kubectl apply -f source-config.yaml
```

Verify the source ConfigMap:

```bash
kubectl get configmap source-config -n default -o yaml
```

### 3. ConfigMap Synchronization

#### Create ConfigMapSyncer Resource

This defines how your ConfigMap should be synchronized across namespaces:

```yaml
# configmap-syncer.yaml
apiVersion: sync.conf-sync.com/v1alpha1
kind: ConfigMapSyncer
metadata:
  name: test-syncer
  namespace: default # The syncer can be in any namespace
spec:
  masterConfigMap:
    name: source-config
    namespace: default # Source ConfigMap namespace
  targetNamespaces: # List of target namespaces
    - app1 # First application namespace
    - app2 # Second application namespace
    - monitoring # Monitoring namespace
  mergeStrategy: Merge # How to handle existing ConfigMaps
  syncInterval: 3 # Sync every 3 seconds
```

Apply the ConfigMapSyncer:

```bash
kubectl apply -f configmap-syncer.yaml
```

#### Verify Synchronization

1. Check ConfigMaps in Each Namespace:

```bash
# Check source ConfigMap
kubectl get configmap source-config -n default -o yaml

# Check synchronized ConfigMaps in target namespaces
kubectl get configmap source-config -n app1 -o yaml
kubectl get configmap source-config -n app2 -o yaml
kubectl get configmap source-config -n monitoring -o yaml
```

2. Verify Data Consistency:

```bash
# Compare ConfigMap data across namespaces
for ns in default app1 app2 monitoring; do
  echo "=== Namespace: $ns ==="
  kubectl get configmap source-config -n $ns -o jsonpath='{.data}' | jq .
done
```

### 4. Testing Updates

#### Update Source ConfigMap

```bash
# Update a single property
kubectl patch configmap source-config -n default --type=merge -p '{"data":{"app.properties":"log.level=DEBUG\nmax.connections=200\ntimeout=60s"}}'

# Add new property
kubectl patch configmap source-config -n default --type=merge -p '{"data":{"new.property":"value1"}}'
```

#### Verify Updates are Synchronized

```bash
# Wait for sync interval (3 seconds)
sleep 4

# Check updates in target namespaces
for ns in app1 app2 monitoring; do
  echo "=== Checking namespace: $ns ==="
  kubectl get configmap source-config -n $ns -o yaml
done
```

### 5. Cleanup

To remove synchronized ConfigMaps:

```bash
# Delete ConfigMapSyncer (stops synchronization)
kubectl delete configmapsyncer test-syncer -n default

# Delete ConfigMaps from all namespaces
for ns in default app1 app2 monitoring; do
  kubectl delete configmap source-config -n $ns
done

# Optionally, delete namespaces
kubectl delete namespace app1
kubectl delete namespace app2
kubectl delete namespace monitoring
```

## Available Make Commands

The following make commands are available for development and deployment:

### Build and Deploy

- `make docker-build IMG=<image-name>` - Build the controller image
- `make deploy IMG=<image-name>` - Deploy the controller to the cluster
- `make undeploy` - Remove the controller from the cluster
- `make install` - Install CRDs into the cluster
- `make uninstall` - Remove CRDs from the cluster

### Development

- `make manifests` - Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects
- `make generate` - Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations
- `make fmt` - Run go fmt against code
- `make vet` - Run go vet against code
- `make test` - Run tests
- `make test-e2e` - Run end-to-end tests

### Debug and Status

- `make debug-controller` - Check controller deployment and logs
- `make check-crd` - Check CRD status and details
- `make list-resources` - List all ConfigMapSyncer resources
- `make check-sync-status` - Check sync status of ConfigMapSyncers
- `make check-configmaps` - Check source and target ConfigMaps
- `make debug-all` - Run all debugging checks

## Running Tests

### Unit Tests

```bash
make test
```

### End-to-End Tests

```bash
# Run e2e tests
make test-e2e
```

## Troubleshooting

### Check Controller Logs

```bash
make debug-controller
```

### Check Sync Status

```bash
make check-sync-status
```

### Check ConfigMaps

```bash
make check-configmaps
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

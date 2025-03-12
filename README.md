# ConfigMap Sync Controller

A Kubernetes controller that synchronizes ConfigMaps across multiple namespaces with configurable sync intervals and merge strategies.

## 🚀 Quick Start (5 minutes)

```bash
# 1. Clone the repository
git clone https://github.com/devShahriar/configmap-sync-controller.git
cd configmap-sync-controller

# 2. Deploy the controller (pre-built image)
make install  # Install CRDs
make deploy IMG=devshahriar/configmap-sync-controller:latest  # Deploy controller

# 3. Create test namespaces
kubectl create namespace app1
kubectl create namespace app2

# 4. Create a sample ConfigMap
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: demo-config
  namespace: default
data:
  app.properties: |
    environment=production
    log.level=INFO
EOF

# 5. Create a ConfigMapSyncer
cat <<EOF | kubectl apply -f -
apiVersion: sync.conf-sync.com/v1alpha1
kind: ConfigMapSyncer
metadata:
  name: demo-syncer
  namespace: default
spec:
  masterConfigMap:
    name: demo-config
    namespace: default
  targetNamespaces:
    - app1
    - app2
  syncInterval: 3
EOF

# 6. Verify synchronization
kubectl get configmap demo-config -n app1 -o yaml
kubectl get configmap demo-config -n app2 -o yaml
```

## 🎯 Project Highlights

- **Problem Solved**: Eliminates manual ConfigMap duplication across namespaces
- **Key Features**:
  - Real-time ConfigMap synchronization
  - Flexible merge strategies
  - Namespace-specific configurations
  - Kubernetes-native implementation
- **Technical Stack**:
  - Go 1.19+
  - Kubernetes Operator Pattern
  - Controller Runtime
  - Custom Resource Definitions (CRDs)

## 🛠️ Technical Implementation

### Key Components

1. **Controller**: Manages ConfigMap synchronization across namespaces
2. **Custom Resource**: ConfigMapSyncer CRD for defining sync rules
3. **RBAC**: Fine-grained permission control
4. **Merge Strategies**: Configurable approaches for handling conflicts

### Architecture Diagram

```
┌──────────────┐      ┌─────────────────┐
│              │      │                 │
│  Source      │──┐   │ ConfigMapSyncer │
│  ConfigMap   │  │   │     (CRD)       │
│              │  │   │                 │
└──────────────┘  │   └─────────────────┘
                  │           │
                  │           │
        ┌─────────▼───────────▼────────┐
        │                              │
        │    Controller (Operator)     │
        │                              │
        └───────────┬──────────┬───────┘
                    │          │
         ┌──────────▼─┐    ┌──▼──────────┐
         │            │    │             │
         │  Target    │    │   Target    │
         │ ConfigMap  │    │  ConfigMap  │
         │ (app1)     │    │  (app2)     │
         └────────────┘    └─────────────┘
```

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
- Support for different target ConfigMap names
- Label selector support for target ConfigMaps

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
# Build the controller image locally
make docker-build IMG=configmap-sync-controller:latest

# Image is already uploaded at devshahriar/configmap-sync-controller:latest you can use this one too

# Install CRDs
make install

# Deploy the controller
make deploy IMG=devshahriar/configmap-sync-controller:latest
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

Apply the target ConfigMap

kubectl apply -f source-config.yaml -n app1
kubectl apply -f source-config.yaml -n app2
kubectl apply -f source-config.yaml -n monitoring

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
kubectl apply -f examples/configmapsyncer_sample.yaml
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

# RBAC Permissions

## Default Behavior

By default, RBAC creation is **enabled** in the Helm chart. This means:

1. A ServiceAccount is created automatically
2. Required RBAC permissions are set up during installation
3. The controller can immediately start watching and managing ConfigMapSyncer resources

The following permissions are granted by default:

- List, watch, and modify ConfigMapSyncer resources
- List, watch, and modify ConfigMaps in all namespaces
- List and watch Namespaces
- Leader election permissions

## Customizing RBAC

If you want to disable RBAC (not recommended), you can do so during installation:

```bash
helm install configmap-sync-controller . --set rbac.create=false --set serviceAccount.create=false
```

Without these permissions, you'll see errors like:

```
E0312 11:42:24.094197       1 reflector.go:166] "Unhandled Error" err="failed to list *v1alpha1.ConfigMapSyncer: configmapsyncers.sync.conf-sync.com is forbidden: User \"system:serviceaccount:default:configmap-sync-controller\" cannot list resource \"configmapsyncers\""
```

## Alternative Ways to Add RBAC Permissions

If you can't use the Helm chart's RBAC creation (due to cluster policies or other restrictions), you can manually create the required RBAC resources:

1. Create a ServiceAccount:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: configmap-sync-controller
  namespace: default # or your desired namespace
```

2. Create the ClusterRole:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: configmap-sync-controller-role
rules:
  - apiGroups:
      - sync.conf-sync.com
    resources:
      - configmapsyncers
    verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
  - apiGroups:
      - sync.conf-sync.com
    resources:
      - configmapsyncers/status
      - configmapsyncers/finalizers
    verbs:
      - get
      - patch
      - update
  - apiGroups:
      - ""
    resources:
      - configmaps
    verbs:
      - create
      - delete
      - get
      - list
      - patch
      - update
      - watch
  - apiGroups:
      - ""
    resources:
      - namespaces
    verbs:
      - get
      - list
      - watch
```

3. Create the ClusterRoleBinding:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: configmap-sync-controller-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: configmap-sync-controller-role
subjects:
  - kind: ServiceAccount
    name: configmap-sync-controller
    namespace: default # same as ServiceAccount namespace
```

4. Apply these resources:

```bash
# Save the above YAML in rbac-manual.yaml and apply:
kubectl apply -f rbac-manual.yaml
```

5. Update your deployment to use this ServiceAccount:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: configmap-sync-controller
spec:
  template:
    spec:
      serviceAccountName: configmap-sync-controller # reference the ServiceAccount
```

## Configuration Options

### ConfigMapSyncer Fields

| Field                             | Type     | Required | Default        | Description                                                                                                                                                             |
| --------------------------------- | -------- | -------- | -------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `masterConfigMap`                 | Object   | Yes      | -              | Specifies the source ConfigMap to sync                                                                                                                                  |
| `masterConfigMap.name`            | String   | Yes      | -              | Name of the source ConfigMap                                                                                                                                            |
| `masterConfigMap.namespace`       | String   | Yes      | -              | Namespace where the source ConfigMap is located                                                                                                                         |
| `targetConfigMapName`             | String   | No       | Same as source | Name to use for ConfigMaps in target namespaces. If not specified, uses the source ConfigMap's name                                                                     |
| `targetNamespaces`                | []String | Yes      | -              | List of namespaces where the ConfigMap should be synchronized to                                                                                                        |
| `mergeStrategy`                   | String   | No       | "Replace"      | How to handle existing ConfigMaps in target namespaces:<br>- `Replace`: Overwrites existing ConfigMaps<br>- `Merge`: Merges with existing data, source takes precedence |
| `syncInterval`                    | Integer  | No       | 3              | How often to check for changes and sync (in seconds)                                                                                                                    |
| `targetSelector`                  | Object   | No       | -              | Label selector to identify specific ConfigMaps to sync                                                                                                                  |
| `targetSelector.matchLabels`      | Map      | No       | -              | Key-value pairs that ConfigMaps must match                                                                                                                              |
| `targetSelector.matchExpressions` | []Object | No       | -              | Advanced label selection rules                                                                                                                                          |

### Example Configuration

```yaml
apiVersion: sync.conf-sync.com/v1alpha1
kind: ConfigMapSyncer
metadata:
  name: example-syncer
  namespace: default
spec:
  # Required: Source ConfigMap details
  masterConfigMap:
    name: source-config
    namespace: default

  # Optional: Custom name for target ConfigMaps
  targetConfigMapName: synced-config

  # Required: Target namespaces to sync to
  targetNamespaces:
    - app1
    - app2
    - monitoring

  # Optional: How to handle existing ConfigMaps
  mergeStrategy: Merge

  # Optional: Sync interval in seconds
  syncInterval: 5

  # Optional: Select specific ConfigMaps by labels
  targetSelector:
    matchLabels:
      environment: production
    matchExpressions:
      - key: tier
        operator: In
        values:
          - frontend
          - backend
```

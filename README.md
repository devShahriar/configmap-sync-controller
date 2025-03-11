# ConfigMap Sync Controller

A Kubernetes controller that watches a "master" ConfigMap in one namespace and automatically propagates its changes (merging) to ConfigMaps in target namespaces.

## Overview

The ConfigMap Sync Controller allows you to define a "master" ConfigMap in one namespace and automatically sync its contents to ConfigMaps in other namespaces. This is useful for maintaining consistent configuration across multiple namespaces, such as:

- Sharing common configuration across multiple applications
- Propagating global settings to different environments
- Centralizing configuration management

The controller introduces a Custom Resource Definition (CRD) called `ConfigMapSyncer` which allows you to specify:

- The source ConfigMap to sync from
- Target namespaces to sync to
- Optional label selectors to target specific ConfigMaps
- Merge strategy (Replace or Merge)

## Features

- **Namespace-based targeting**: Sync to specific namespaces or all namespaces
- **Label-based targeting**: Target ConfigMaps with specific labels
- **Flexible merge strategies**: Choose between replacing or merging ConfigMap data
- **Status tracking**: Monitor sync status through the ConfigMapSyncer status
- **Automatic reconciliation**: Periodically checks for changes and syncs as needed

## Installation

### Prerequisites

- Kubernetes cluster v1.19+
- kubectl v1.19+
- Go v1.19+ (for development)

### Installing the Controller

1. Clone the repository:

```sh
git clone https://github.com/shahriar-siemens/configmap-sync-controller.git
cd configmap-sync-controller
```

2. Install the CRDs:

```sh
make install
```

3. Run the controller (development mode):

```sh
make run
```

### Deploying to a Cluster

1. Build and push the Docker image:

```sh
make docker-build docker-push IMG=<your-registry>/configmap-sync-controller:latest
```

2. Deploy the controller:

```sh
make deploy IMG=<your-registry>/configmap-sync-controller:latest
```

## Usage

### Creating a ConfigMapSyncer Resource

1. Create a source ConfigMap:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: source-config
  namespace: default
data:
  app.properties: |
    app.name=MyApp
    app.version=1.0.0
    app.environment=production
```

2. Create a ConfigMapSyncer resource:

```yaml
apiVersion: sync.siemens.com/v1alpha1
kind: ConfigMapSyncer
metadata:
  name: sample-configmap-syncer
  namespace: default
spec:
  masterConfigMap:
    name: source-config
    namespace: default
  targetNamespaces:
    - app1
    - app2
    - monitoring
  mergeStrategy: Merge
```

### Using Label Selectors

You can use label selectors to target specific ConfigMaps:

```yaml
apiVersion: sync.siemens.com/v1alpha1
kind: ConfigMapSyncer
metadata:
  name: selector-configmap-syncer
  namespace: default
spec:
  masterConfigMap:
    name: source-config
    namespace: default
  targetNamespaces:
    - app1
    - app2
  targetSelector:
    matchLabels:
      app: my-app
      sync-enabled: "true"
  mergeStrategy: Replace
```

### Checking Sync Status

You can check the status of the ConfigMapSyncer resource:

```sh
kubectl get configmapsyncers -o wide
kubectl describe configmapsyncer sample-configmap-syncer
```

## API Reference

### ConfigMapSyncer

| Field                   | Type               | Description                                                              |
| ----------------------- | ------------------ | ------------------------------------------------------------------------ |
| `spec.masterConfigMap`  | ConfigMapReference | Reference to the source ConfigMap                                        |
| `spec.targetNamespaces` | []string           | List of namespaces to sync to (optional)                                 |
| `spec.targetSelector`   | LabelSelector      | Label selector to target specific ConfigMaps (optional)                  |
| `spec.mergeStrategy`    | string             | Strategy for merging ConfigMaps: "Replace" or "Merge" (default: "Merge") |

### ConfigMapReference

| Field       | Type   | Description                |
| ----------- | ------ | -------------------------- |
| `name`      | string | Name of the ConfigMap      |
| `namespace` | string | Namespace of the ConfigMap |

## Examples

Check the [examples](./examples) directory for sample ConfigMapSyncer resources and test scripts.

## Development

### Prerequisites

- Go v1.19+
- Kubebuilder v3.0.0+
- Docker

### Building and Testing

```sh
# Run tests
make test

# Build the controller
make build

# Run the controller locally
make run
```

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

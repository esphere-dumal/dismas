# dismas
Dismas is a simple Operator dispatching job into each nodes in a k8s cluster.

## Description
Dismas is built by Kubebuilder, and is able to be deployed by helmx

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use [KIND](https://sigs.k8s.io/kind) to get a local cluster for testing, or run against a remote cluster.
**Note:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Running on the cluster

> Ensure there is kustomize and helm executable file in bin/

- Generate CRD files by controller-gen
```sh
make manifests
```

- Build the image by docker:
``` sh
make docker-build docker-push IMG=esphe/dismas:latest
```
- Generate config file for helm by kustomize
``` sh
make helm
```
- Install in cluster
> Later I should upload the charts into repos
```sh
helm package deploy
helm install dismas dismas-deploy-0.1.0
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy charts
UnDeploy the charts from the cluster:

```sh
helm uninstall dismas
```

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/),
which provide a reconcile function responsible for synchronizing resources until the desired state is reached on the cluster.

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

## License

Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.


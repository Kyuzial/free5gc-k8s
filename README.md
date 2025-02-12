# Free5GC Kubernetes Operator

This operator automates the deployment and management of Free5GC components in a Kubernetes cluster.

## Features

- Automated deployment of Free5GC components (AMF, SMF, UPF, etc.)
- MongoDB database management
- Network interface configuration for N2, N3, N4, and N6 interfaces
- Support for UPF in ULCL mode
- Automatic status tracking for all components

## Prerequisites

- Kubernetes cluster 1.19+
- kubectl configured to communicate with your cluster
- Multus CNI plugin installed for network interface management

## Installation

1. Install the CRDs:
```bash
make install
```

2. Build and deploy the operator:
```bash
# Build the operator image
make docker-build IMG=<your-registry>/free5gc-operator:v0.1.0

# Push the image to your registry
make docker-push IMG=<your-registry>/free5gc-operator:v0.1.0

# Deploy the operator to the cluster
make deploy IMG=<your-registry>/free5gc-operator:v0.1.0
```

## Usage

1. Create the necessary NetworkAttachmentDefinitions for Free5GC interfaces:

```yaml
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: n2-net
spec:
  config: '{
    "cniVersion": "0.3.1",
    "type": "macvlan",
    "master": "eth1",
    "mode": "bridge",
    "ipam": {
      "type": "host-local",
      "subnet": "10.100.50.0/24"
    }
  }'
```

2. Deploy Free5GC using the operator:

```yaml
apiVersion: core.free5gc.org/v1alpha1
kind: Free5GC
metadata:
  name: free5gc-sample
spec:
  mongodb:
    image: mongo:4.4
    storage:
      size: 1Gi
      storageClassName: standard

  network:
    n2Network:
      name: n2-net
      interface: n2
    n3Network:
      name: n3-net
      interface: n3
    n4Network:
      name: n4-net
      interface: n4
    n6Network:
      name: n6-net
      interface: n6

  nrf:
    image: free5gc/nrf:v3.2.0
    replicas: 1
    resources:
      requests:
        cpu: 100m
        memory: 128Mi
      limits:
        cpu: 250m
        memory: 256Mi

  # Other components...
```

## Component Configuration

### MongoDB

MongoDB can be deployed either internally or externally:

```yaml
spec:
  mongodb:
    # Use external MongoDB
    external: true
    uri: "mongodb://external-mongodb:27017"
    
    # Or deploy MongoDB internally
    external: false
    image: mongo:4.4
    storage:
      size: 1Gi
      storageClassName: standard
```

### UPF

UPF can be deployed in standard mode or ULCL mode:

```yaml
spec:
  # Standard UPF
  upf:
    image: free5gc/upf:v3.2.0
    replicas: 1
    config:
      pfcp:
        addr: upf.free5gc.org
        nodeID: upf.free5gc.org
        retransTimeout: 1s
        maxRetrans: 3
      gtpu:
        forwarder: gtp5g
        ifname: upfgtp

  # Or ULCL-enabled UPF
  upf:
    image: free5gc/upf:v3.2.0
    ulcl:
      enabled: true
      instances:
        - name: upf1
          image: free5gc/upf:v3.2.0
        - name: upf2
          image: free5gc/upf:v3.2.0
```

## Status

The operator reports status for all components:

```bash
kubectl get free5gc free5gc-sample -o yaml
```

```yaml
status:
  components:
    amf:
      phase: Running
      readyReplicas: 1
      replicas: 1
    smf:
      phase: Running
      readyReplicas: 1
      replicas: 1
    # ...
  mongodb:
    phase: Running
    readyReplicas: 1
    replicas: 1
```

## Development

1. Clone the repository:
```bash
git clone https://github.com/your-org/free5gc-operator.git
cd free5gc-operator
```

2. Install the dependencies:
```bash
go mod download
```

3. Run the operator locally:
```bash
make run
```

4. Run tests:
```bash
# Unit tests
make test

# E2E tests
make test-e2e
```

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details.

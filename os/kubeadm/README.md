# Join a worker to ManagedControlPlane using kubeadm

This README shows how to join a Docker container as a Kubernetes worker node to an existing **ManagedControlPlane (MCP)** using `kubeadm`.

## Prerequisites

- Docker installed
- kubectl installed
- A working MCP kubeconfig:

## 0. Setup

Run below commands:

```bash
task kind:create
task dev:install
kubectl apply -f ./os/kubeadm/mcp.yaml
```

## 1. Create Docker Volume

```bash
docker volume create mcp-worker1-var
```

## 2. Start Worker Container

```bash
docker run -d --name mcp-worker1   --hostname mcp-worker1   --privileged   --network kind   --cgroupns=private   -v /lib/modules:/lib/modules:ro   -v mcp-worker1-var:/var   --tmpfs /run   --tmpfs /tmp   --security-opt seccomp=unconfined   --security-opt apparmor=unconfined   --security-opt label=disable   kindest/node:v1.35.0
```

## 3. Deploy controlplane-operator

```bash
./bin/controlplane-operator -kubeconfig <kubeconfig>
```

## 4. Get MCP kubeconfig

```bash
./bin/tesseractl mcp kubeconfig my-kubernetes > ~/.kube/mcp
```

## 5. Get Bootstrap Token

```bash
SECRET_NAME=$(KUBECONFIG=~/.kube/mcp kubectl -n kube-system get secret --sort-by=.metadata.creationTimestamp | grep bootstrap-token | tail -n 1 | cut -d " " -f1)

TOKEN_SECRET=$(KUBECONFIG=~/.kube/mcp kubectl -n kube-system get secret $SECRET_NAME   -o jsonpath='{.data.token-secret}' | base64 -d)

TOKEN_ID=$(KUBECONFIG=~/.kube/mcp kubectl -n kube-system get secret $SECRET_NAME   -o jsonpath='{.data.token-id}' | base64 -d)

TOKEN=${TOKEN_ID}.${TOKEN_SECRET}
echo "$TOKEN"
```

## 6. Compute CA Cert Hash

```bash
KUBECONFIG=~/.kube/mcp kubectl -n kube-public get cm cluster-info   -o jsonpath='{.data.kubeconfig}'   | awk '/certificate-authority-data:/ {print $2}'   | base64 -d > /tmp/cluster-info-ca.crt

HASH=$(openssl x509 -pubkey -in /tmp/cluster-info-ca.crt   | openssl pkey -pubin -outform der 2>/dev/null   | openssl dgst -sha256 -hex   | awk '{print $2}')

echo "sha256:$HASH"
```

## 7. Get API Server Address

```bash
API=$(kubectl get svc -n default my-kubernetes-apiserver   -o=jsonpath="{.status.loadBalancer.ingress[0].ip}"):6443
```

## 8. Join the Cluster

```bash
docker exec -it mcp-worker1 bash -lc   "kubeadm join $API    --token $TOKEN    --discovery-token-ca-cert-hash sha256:$HASH    --ignore-preflight-errors=all    -v=10"
```

## Done 🎉

Verify the node:

```bash
$ KUBECONFIG=~/.kube/mcp k get node
NAME          STATUS   ROLES    AGE     VERSION
mcp-worker1   Ready    <none>   8m14s   v1.34.0
$ KUBECONFIG=~/.kube/mcp k get pods -A
NAMESPACE            NAME                                      READY   STATUS              RESTARTS   AGE
kube-flannel         kube-flannel-ds-85tpm                     1/1     Running             0          8m18s
kube-system          coredns-7445cdc8fd-plbqh                  0/1     ContainerCreating   0          18m
kube-system          coredns-7445cdc8fd-xb5ql                  0/1     ContainerCreating   0          18m
kube-system          konnectivity-agent-smk9s                  1/1     Running             0          8m5s
kube-system          kube-proxy-p57j6                          1/1     Running             0          8m18s
local-path-storage   local-path-provisioner-866d54d4c8-stjnq   0/1     ContainerCreating   0          18m
```

You should see `mcp-worker1` in `Ready` state.

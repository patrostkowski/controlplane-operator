on test-kubectl:

```
kubectl apply -f - <<EOF
---
apiVersion: v1
kind: Secret
metadata:
  name: bootstrap-token-abc123
  namespace: kube-system
type: bootstrap.kubernetes.io/token
stringData:
  description: "Bootstrap token for kubelet workers"
  token-id: abc123
  token-secret: abcdefg123456789
  auth-extra-groups: "system:bootstrappers:kubelet-bootstrap"
  usage-bootstrap-authentication: "true"
  usage-bootstrap-signing: "true"
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubelet-bootstrap-auto-approve-node-client-certs
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:certificates.k8s.io:certificatesigningrequests:nodeclient
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: system:bootstrappers:kubelet-bootstrap
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubelet-auto-approve-node-client-cert-rotation
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:certificates.k8s.io:certificatesigningrequests:selfnodeclient
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: Group
  name: system:nodes
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kubelet-bootstrap
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: system:node-bootstrapper
subjects:
- kind: Group
  name: system:bootstrappers:kubelet-bootstrap
  apiGroup: rbac.authorization.k8s.io
---
EOF
```

on management cluster:

```
kubectl get -n mcp secret managed-ca -o=jsonpath="{.data.ca\.crt}" | base64 -d > ca.crt
```

run worker:

./run.sh worker-0
./run.sh worker-1
./run.sh worker-2

enjoy hosts showing up in mcp!

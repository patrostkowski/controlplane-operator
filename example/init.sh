#!/bin/bash

set -xeu

helm install cert-manager "oci://quay.io/jetstack/charts/cert-manager" \
    --namespace cert-manager \
    --create-namespace \
    --set crds.enabled=true

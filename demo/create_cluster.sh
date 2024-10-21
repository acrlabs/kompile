#!/bin/bash

MOUNT_PATH=$HOME/tmp/kind-node-data
echo "Using $MOUNT_PATH for kind node mounts"
mkdir -p $MOUNT_PATH

cat <<EOF > kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
    labels:
      type: kind-worker
    extraMounts:
      - hostPath: $MOUNT_PATH
        containerPath: /data
  - role: worker
    labels:
      type: kind-worker
    extraMounts:
      - hostPath: $MOUNT_PATH
        containerPath: /data
EOF

kind create cluster --config kind-config.yaml
rm kind-config.yaml


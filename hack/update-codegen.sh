#!/usr/bin/env bash

if [ "$IS_CONTAINER" != "" ]; then
  set -xe

  SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
  bash ${SCRIPT_ROOT}/vendor/k8s.io/code-generator/generate-groups.sh all \
    github.com/openshift-splat-team/vsphere-capacity-manager/pkg/generated \
    github.com/openshift-splat-team/vsphere-capacity-manager/pkg/apis \
    "vsphere-capacity-manager.splat-team.io:v1" \
    --go-header-file ${SCRIPT_ROOT}/hack/custom-boilerplate.go.txt
  set +ex
  # git diff --exit-code
else
  podman run --rm \
    --env IS_CONTAINER=TRUE \
    --volume "${PWD}:/go/src/github.com/openshift-splat-team/vsphere-capacity-manager:z" \
    --workdir /go/src/github.com/openshift-splat-team/vsphere-capacity-manager \
    docker.io/golang:1.18 \
    ./hack/update-codegen.sh "${@}"
fi

#!/bin/bash

export E2E_TYPE=${E2E_TYPE:-"ALL"}

# Run e2e test

go install -mod=mod github.com/onsi/ginkgo/v2/ginkgo

case ${E2E_TYPE} in
"ALL")
    echo "Running e2e..."
    KUBECONFIG=${KUBECONFIG} ginkgo -r --slow-spec-threshold='30s' --progress ./test/e2e/jobtemplate-controller/
    KUBECONFIG=${KUBECONFIG} ginkgo -r --slow-spec-threshold='30s' --progress ./test/e2e/jobflow-controller/
    ;;
"JOBTEMPLATECONTROLLER")
    echo "Running jobtemplate controller e2e suite..."
    KUBECONFIG=${KUBECONFIG} ginkgo -r --slow-spec-threshold='30s' --progress ./test/e2e/jobtemplate-controller/
    ;;
"JOBFLOWCONTROLLER")
    echo "Running jobflow controller e2e suite..."
    KUBECONFIG=${KUBECONFIG} ginkgo -r --slow-spec-threshold='30s' --progress ./test/e2e/jobflow-controller/
    ;;
esac


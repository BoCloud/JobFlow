#!/bin/bash

export E2E_TYPE=${E2E_TYPE:-"ALL"}

# Run e2e test

pwd

case ${E2E_TYPE} in
"ALL")
    echo "Running e2e..."
    KUBECONFIG=${KUBECONFIG} ginkgo -r --slowSpecThreshold=30 --progress ./test/e2e/jobtemplate-controller/
    KUBECONFIG=${KUBECONFIG} ginkgo -r --slowSpecThreshold=30 --progress ./test/e2e/jobflow-controller/
    ;;
"JOBTEMPLATECONTROLLER")
    echo "Running sequence job e2e suite..."
    KUBECONFIG=${KUBECONFIG} ginkgo -r --slowSpecThreshold=30 --progress ./test/e2e/jobtemplate-controller/
    ;;
"JOBFLOWCONTROLLER")
    echo "Running scheduling base e2e suite..."
    KUBECONFIG=${KUBECONFIG} ginkgo -r --slowSpecThreshold=30 --progress ./test/e2e/jobflow-controller/
    ;;
esac


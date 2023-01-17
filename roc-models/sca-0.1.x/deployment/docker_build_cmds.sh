#!/usr/bin/env bash
# SPDX-FileCopyrightText: (C) 2022 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

docker build --platform linux/x86_64 -f roc-models/sca-0.1.x/Dockerfile -t default-route-openshift-image-registry.apps.nex.one-edge.intel.com/springboard-dev-common/roc-model-sca-0.1.x:0.1.2-dev roc-models/sca-0.1.x&
#docker push default-route-openshift-image-registry.apps.nex.one-edge.intel.com/springboard-dev-common/roc-model-sca-0.1.x:0.1.2-dev

docker build --platform linux/x86_64 -f roc-models/sca-0.1.x/out/roc-api-server-sca-0.1.x/build/Dockerfile -t default-route-openshift-image-registry.apps.nex.one-edge.intel.com/springboard-dev-common/roc-api-server-sca-0.1.x:0.1.2-dev roc-models/sca-0.1.x/out/roc-api-server-sca-0.1.x&
#docker push default-route-openshift-image-registry.apps.nex.one-edge.intel.com/springboard-dev-common/roc-api-server-sca-0.1.x:0.1.2-dev

docker build --platform linux/x86_64 -f roc-models/sca-0.1.x/out/roc-adapter-impl-sca-0.1.x/Dockerfile -t default-route-openshift-image-registry.apps.nex.one-edge.intel.com/springboard-dev-common/roc-adapter-sca-0.1.x:0.1.2-dev roc-models/sca-0.1.x/out/roc-adapter-impl-sca-0.1.x&
#docker push default-route-openshift-image-registry.apps.nex.one-edge.intel.com/springboard-dev-common/roc-adapter-sca-0.1.x:0.1.2-dev

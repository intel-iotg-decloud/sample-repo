#!/usr/bin/env bash
# SPDX-FileCopyrightText: (C) 2022 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

docker build --platform linux/x86_64 -f roc-models/sra-0.2.x/Dockerfile -t default-route-openshift-image-registry.apps.nex.one-edge.intel.com/springboard-dev-common/roc-sra-0.2.x:0.2.1-dev roc-models/sra-0.2.x&

docker build --platform linux/x86_64 -f roc-models/sra-0.2.x/out/roc-api-server-sra-0.2.x/build/Dockerfile -t default-route-openshift-image-registry.apps.nex.one-edge.intel.com/springboard-dev-common/roc-api-server-sra-0.2.x:0.2.1-dev roc-models/sra-0.2.x/out/roc-api-server-sra-0.2.x&

docker build --platform linux/x86_64 -f roc-models/sra-0.2.x/out/roc-adapter-impl-sra-0.2.x/Dockerfile -t default-route-openshift-image-registry.apps.nex.one-edge.intel.com/springboard-dev-common/roc-adapter-sra-0.2.x:0.2.1-dev roc-models/sra-0.2.x/out/roc-adapter-impl-sra-0.2.x&

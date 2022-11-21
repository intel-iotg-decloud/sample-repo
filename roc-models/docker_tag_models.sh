#!/usr/bin/env bash
# SPDX-FileCopyrightText: (C) 2022 Intel Corporation
# SPDX-License-Identifier: LicenseRef-Intel

#set -x
set -e
set -o pipefail

echo "tagging all repos for alternate registry"

export SOURCE_REGISTRY="${SOURCE_REGISTRY:-amr-registry.caas.intel.com}"
export SOURCE_REPOSITORY_BASE="${SOURCE_REPOSITORY_BASE:-one-intel-edge/roc/}"

TARGET_REGISTRY=
TARGET_REPOSITORY=

while getopts "g:r:h" opt; do
  case "$opt" in
    g)
      TARGET_REGISTRY=${OPTARG}
      ;;
    r)
      TARGET_REPOSITORY=${OPTARG}
      ;;
    :)
      echo "Usage: $(basename $0) -g registry -r repository"
      exit 1
      ;;
    ?|h)
      echo "Usage: $(basename $0) -g registry -r repository"
      exit 1
      ;;
  esac
done
shift "$(($OPTIND -1))"

if [ -z "$TARGET_REGISTRY" ] || [ -z "$TARGET_REPOSITORY" ]; then
        echo "Missing -g or -r ${TARGET_REGISTRY}${TARGET_REPOSITORY}" >&2
        exit 1
fi

for FILE in roc-models/*/deployment/docker_build_cmds.sh; do
  for line in `grep "docker build" $FILE | awk '{print $8}'`; do
    REL_REPO=`echo $line | awk -F '/' '{print $3}'`
    echo "docker tag $line ${TARGET_REGISTRY}${TARGET_REPOSITORY}${REL_REPO}"
    echo "docker push ${TARGET_REGISTRY}${TARGET_REPOSITORY}${REL_REPO}"
  done

done;
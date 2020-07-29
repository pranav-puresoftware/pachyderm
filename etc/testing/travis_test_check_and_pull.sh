#!/bin/bash

set -ex

# See travis_build_check_and_stash.sh for context.
#
# This script takes the docker image which that script pushed, and unpacks the
# git source tree from it, as it is more reliably going to be the correct
# source tree for this build than what Travis and GitHub give us (in the PR
# build case, at least).

if [ "${TRAVIS_PULL_REQUEST}" == "false" ]; then
    # These shenannigans not needed for release and branch builds, hopefully.
    exit 0
fi

if [[ ! "$TRAVIS_SECURE_ENV_VARS" == "true" ]]; then
    echo "Not trying to fetch pachyderm.tar as we are running for an external"
    echo "contributor"
else
    cd /home/travis/gopath/src/github.com/pranav-puresoftware/
    mv pachyderm pachyderm.old

    docker login -u pachydermbuildbot -p "${DOCKER_PWD}"
    docker run -v "$(pwd)":/unpack pachyderm/ci_code_bundle:"${TRAVIS_BUILD_NUMBER}" \
        tar xf /pachyderm.tar -C /unpack/

    ls -alh pachyderm
    sudo chown -R "${USER}:${USER}" pachyderm
    cd pachyderm
fi

# Check that the state we got matches the commit we're supposed to be testing.
parents=$(git rev-list --parents -n 1 HEAD)
if [[ ! "$parents" == *"$TRAVIS_PULL_REQUEST_SHA"* ]]; then
    set +x
    echo "===================================================================================="
    echo "GitHub didn't give us the commit we're meant to be testing ($TRAVIS_PULL_REQUEST_SHA)"
    echo "as one of the parents of the HEAD merge preview commit ($parents). Giving up!"
    echo "===================================================================================="
    set -x
    exit 1
fi

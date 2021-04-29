#!/bin/bash

set -e

VERSION=$1
UIVERSION=${2:-$1}

if [[ -z $VERSION ]]
then
cat <<EOF

<------------------------------------------------>
    Usage:
        ./gen_image.sh {version}
        ./gen_image.sh {clxone-controller version} {clxone-web version}
<------------------------------------------------>

EOF
    exit 1
fi

cat <<EOF

<------------------------------------------------>
    Building ...
<------------------------------------------------>

EOF

cat <<'EOF' | docker build -f - -t linkingthing/clxone-controller:${VERSION}-${UIVERSION} --build-arg version=${VERSION} --build-arg uiversion=${UIVERSION} .
ARG version
ARG uiversion

FROM linkingthing/clxone-controller:$version as go
FROM linkingthing/clxone-web:$uiversion as js

FROM alpine:3.12

COPY --from=go /clxone-controller /
COPY --from=js /opt/website /opt/website

ENTRYPOINT ["/clxone-controller"]
EOF

if [[ $? -eq 0 ]]
then
docker image prune -f
cat <<EOF

<------------------------------------------------>
  Image build complete.
  Build: zdnscloud/clxone-controller:${VERSION}-${UIVERSION}
<------------------------------------------------>

EOF
else
cat <<EOF

<------------------------------------------------>
  Image build failure.
<------------------------------------------------>

EOF
fi

# This file is part of CardinalHQ, Inc.
#
# CardinalHQ, Inc. proprietary and confidential.
# Unauthorized copying, distribution, or modification of this file,
# via any medium, is strictly prohibited without prior written consent.
#
# Copyright 2025 CardinalHQ, Inc. All rights reserved.


FROM alpine:latest AS certs
RUN apk --update add ca-certificates

# pull in geoip database
FROM scratch

ARG USER_UID=2000
USER ${USER_UID}:${USER_UID}

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY flutter /app/bin/flutter
EXPOSE 8080
CMD ["/app/bin/flutter"]

FROM scratch
# Copy our static executable.
COPY bin/gropsgenie /go/bin/gropsgenie
# Run gropsgenie.
ENTRYPOINT ["/go/bin/gropsgenie"]
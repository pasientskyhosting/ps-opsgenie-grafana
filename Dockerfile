FROM scratch
# Copy our static executable.
COPY bin/ps-opsgenie-grafana64 /go/bin/ps-opsgenie-grafana64
# Run gropsgenie.
ENTRYPOINT ["/go/bin/ps-opsgenie-grafana64"]
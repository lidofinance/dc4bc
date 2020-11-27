FROM golang:1.15.5-buster

WORKDIR ${GOPATH}/src/${APP_DIR}
RUN PATH=${PATH}:${GOPATH}
COPY . .

ENV DATA_DIR ""
ENV USERNAME ""
ENV PASSWORD_EXPIRATION ""
ENV AIRGAPPED_STATE_DIR ""

# RUN make test-short
RUN make build-airgapped-machine

ENTRYPOINT ["./airgapped-machine-entrypoint.sh"]

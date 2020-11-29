FROM golang:1.15.5-buster

WORKDIR ${GOPATH}/src/${APP_DIR}
RUN PATH=${PATH}:${GOPATH}
COPY . .

ENV DATA_DIR ""
ENV USERNAME ""
ENV STORAGE_DBDSN ""
ENV STORAGE_TOPIC ""

RUN make test-short
RUN make build-node

ENTRYPOINT ["./node-entrypoint.sh"]

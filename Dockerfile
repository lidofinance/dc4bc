FROM golang:1.19.4-alpine3.17 as builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download && go mod verify

COPY build.sh .
COPY airgapped/ ./airgapped
COPY client/ ./client
COPY cmd/ ./cmd
COPY dkg/ ./dkg
COPY fsm/ ./fsm
COPY pkg/ ./pkg
COPY storage/ ./storage

ARG platform

RUN chmod +x ./build.sh && ./build.sh $platform

FROM scratch

WORKDIR /

COPY --from=builder /app/build/ ./
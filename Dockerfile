FROM golang:1.22 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download

COPY main.go main.go

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o pcie-device-plugin main.go

FROM alpine:3.20.2
COPY --from=builder /workspace/pcie-device-plugin /usr/local/bin/pcie-device-plugin
ENTRYPOINT ["/usr/local/bin/pcie-device-plugin"]

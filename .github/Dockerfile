# Build the manager binary
FROM golang:1.15-alpine as builder
WORKDIR /build

COPY . .

# Disable Cgo so that the binary doesn't rely on glibc and works with the
# scratch, alpine or distroless image.
RUN CGO_ENABLED=0 go build .

FROM scratch
COPY --from=builder /build/clockidup /clockidup
ENTRYPOINT ["/clockidup"]

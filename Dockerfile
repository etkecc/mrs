FROM ghcr.io/etkecc/base/build AS builder

WORKDIR /app
COPY . .
RUN just build

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /app/mrs /bin/mrs

USER app

ENTRYPOINT ["/bin/mrs"]

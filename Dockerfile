FROM registry.gitlab.com/etke.cc/base/build AS builder
WORKDIR /mrs
COPY . .
RUN just build

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /mrs/api /bin/mrs
USER app
ENTRYPOINT ["/bin/mrs"]

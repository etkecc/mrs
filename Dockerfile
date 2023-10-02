FROM registry.gitlab.com/etke.cc/base/build AS builder
WORKDIR /mrs
COPY . .
RUN just build

FROM registry.gitlab.com/etke.cc/base/app
COPY --from=builder /mrs/mrs /bin/mrs
RUN apk --no-cache add vips libheif
USER app
ENTRYPOINT ["/bin/mrs"]

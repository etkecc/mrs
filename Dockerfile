FROM registry.gitlab.com/etke.cc/base/build AS builder
RUN apk --no-cache add vips-dev libheif-dev pkgconfig
WORKDIR /mrs
COPY . .
RUN just build

FROM registry.gitlab.com/etke.cc/base/app
COPY --from=builder /mrs/mrs /bin/mrs
RUN apk --no-cache add vips libheif
USER app
ENTRYPOINT ["/bin/mrs"]

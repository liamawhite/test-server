FROM alpine:latest as certs
RUN apk --update add ca-certificates

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ADD test-server /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/test-server"]

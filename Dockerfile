ARG GO_VERSION
ARG VERSION

FROM golang:${GO_VERSION}-bullseye as gobuild
RUN adduser \
    --disabled-password \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid 65532 \
    sparrow

WORKDIR /app
ADD . .
RUN go mod download
RUN go mod verify
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION}" -o sparrow .

FROM scratch
COPY --from=gobuild /etc/passwd /etc/passwd
COPY --from=gobuild /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=gobuild /app/sparrow /sparrow

USER sparrow
ENTRYPOINT ["/sparrow", "run"]
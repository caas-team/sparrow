FROM golang:1.21-bullseye as gobuild
ARG VERSION
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
RUN CGO_ENABLED=0 go build -ldflags '-s -w -extldflags "-static" -X main.version=${VERSION}' -o sparrow .

FROM scratch
COPY --from=gobuild /etc/passwd /etc/passwd
COPY --from=gobuild /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=gobuild /app/sparrow /sparrow

USER sparrow
ENTRYPOINT ["/sparrow", "run"]
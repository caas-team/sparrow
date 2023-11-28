FROM cgr.dev/chainguard/go:latest as gobuild
ARG VERSION
WORKDIR /app
ADD . .
RUN go mod download
RUN CGO_ENABLED=0 go build -ldflags "-s -w -X main.version=${VERSION}" -o sparrow .

FROM scratch
COPY --from=gobuild /app/sparrow /sparrow
COPY --from=gobuild /etc/passwd /etc/passwd
USER 65532
ENTRYPOINT ["/sparrow", "run"]
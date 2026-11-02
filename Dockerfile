# Pinned Go toolchain patch. Pin base-image digests before any production deployment.
FROM golang:1.27.3 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/webhook ./cmd/webhook

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/webhook /webhook
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/webhook"]

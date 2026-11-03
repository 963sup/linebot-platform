# golang:1.27.3, pinned to the reviewed multi-platform manifest.
FROM golang@sha256:2920bb124f1a2e12a1d39f0cab756ae0068441ee1da7655dec7f5e374944095b AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/webhook ./cmd/webhook

# static-debian13:nonroot, pinned to the reviewed multi-platform manifest.
FROM gcr.io/distroless/static-debian13@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329
COPY --from=build /out/webhook /webhook
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/webhook"]

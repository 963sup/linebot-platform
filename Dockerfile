# Packages a prebuilt binary produced in Cloud Build's shared /workspace.
FROM gcr.io/distroless/static-debian13@sha256:25ac8f0c3b2334a9eb05452266bd4697864a8c8ef0ef71e5e6480d7f92868329
ARG BINARY
COPY .release/${BINARY} /service
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/service"]

# Dependency Rules

```text
inbound adapters -> application-owned inbound contracts / use cases
application -> domain
application -> application-owned outbound ports
outbound adapters -> application/domain ports
outbound adapters -> infrastructure -> external SDKs
cmd/bootstrap -> inbound adapters, use cases, outbound adapters, infrastructure
```

箭頭代表 compile-time import／composition dependency，不代表 runtime call direction。Domain never imports application, adapters, infrastructure or SDKs. Application never imports concrete adapters, infrastructure or SDKs. Infrastructure provides technical clients and lifecycle primitives; it does not import adapters or business layers. Every external I/O operation propagates context and an explicit deadline. SDK types stop at adapter/infrastructure boundaries.


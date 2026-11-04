# Dependency Rules

```text
inbound adapters -> inbound ports -> application -> domain
                                      application -> outbound ports
outbound adapters -> outbound ports
infrastructure -> adapters and external SDKs
cmd/bootstrap -> concrete implementations
```

Domain never imports application, adapters, infrastructure or SDKs. Application never imports concrete adapters or SDKs. Every external I/O operation propagates context and an explicit deadline. SDK types stop at adapter/infrastructure boundaries.


# Architecture Boundaries

The primary session coordinates work. Domain is independent of transport, cloud SDKs, persistence and infrastructure. Application owns use-case orchestration and consumer-oriented ports. Inbound and outbound adapters isolate protocols and SDKs. Bootstrap and command packages compose concrete implementations.

Bounded contexts own their language, aggregates and invariants. Cross-context communication uses explicit ports, DTOs or domain events; mutable domain models are not shared.


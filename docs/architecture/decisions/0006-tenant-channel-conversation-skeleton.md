# ADR 0006: Tenant channel and inbound conversation skeleton

Status: Superseded by [ADR 0008](0008-shared-bot-durable-skeleton.md).

The earlier design bound each Messaging destination to a Tenant/ChannelConnection and supported direct, group and room sources. The product instead shares one Bot and isolates conversations by user/group. The business models, per-request resolver and registration CLI have been removed.

The historical schema v3 stored tenant/connection fields and used them in event keys. Read-only compatibility metadata remains solely to preserve those keys during cutover; it is not an active tenant registry. See ADR 0008 for the current contract and retirement conditions.

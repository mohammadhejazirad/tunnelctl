# ADR 3: No host network mutations in proxy mode

Accepted for Phase 1. The application owns its loopback listener, child process,
control socket and private runtime metadata. It owns no route, DNS setting,
interface, firewall rule, global proxy setting or shell startup file.

Phase 3 must choose one route owner per backend, persist a write-ahead ownership
journal, recover only still-owned resources, protect SSH with external confirmation
and timed rollback, and define DNS/IPv6 and fail-open versus blocking policies.
No TUN option is exposed now. All privileged tests must run in disposable isolated
VMs/namespaces, never against a developer or production host's real default route.

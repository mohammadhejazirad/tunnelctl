# ADR 2: Unprivileged application and future narrow helper

Accepted for Phase 1. CLI, parsers, storage and proxy supervision require no root.
No capabilities, privilege escalation, service installation or persistent helper
are requested by the application. A future TUN helper must authenticate clients,
accept only a narrow validated plan, and journal owned mutations before applying
them. Giving a user-writable core permanent network administration privileges is
not an acceptable substitute. Native Windows ACL/job and macOS lifecycle adapters
must precede runtime support on those systems.

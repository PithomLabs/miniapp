# SSRF Test Design (Phase 0; probes run at the G2 entry gate, F03)

Custom REST destinations are operator-approved origins enforced by a
dedicated egress layer (`capability/egress`, implemented Phase 2). No
custom-REST connector ships before this suite passes.

## Controls under test

HTTPS only; exact host/origin allowlist; DNS A/AAAA resolution with
connection-time validation of every resolved IP; blocking of loopback,
private, link-local, carrier-grade NAT, multicast, reserved, cloud
metadata, Fly private-network/`.internal`, and database-internal
destinations; connect to the validated IP while preserving hostname
for Host/TLS SNI/certificate checks; no second unchecked resolution;
redirects disabled by default (explicitly permitted hops revalidated,
cross-host rejected); same policy on retries and fallbacks; response
size/duration/content-type/header/decompression limits.

## Probe suite (adversarial, CI)

Loopback/private/link-local/metadata/Fly-internal/Neon destinations;
DNS rebinding (answer flip between resolve and connect); IPv4/IPv6
alternate answers; cross-host redirect; redirect loops; DNS change
during retries; oversized bodies; unsupported content types; slow
responses. Each probe asserts rejection or containment, and the
transport test confirms connections land on the validated IP with
intact TLS hostname validation.

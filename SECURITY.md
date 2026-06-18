# Security Policy

## Reporting a vulnerability

Please report security issues privately, not through public issues. Open a
private advisory with the "Report a vulnerability" button under this
repository's Security tab.

Include a description, reproduction steps, and the affected version
(`hop version`). Expect an acknowledgement within 72 hours and a remediation
plan once the report is triaged.

## Supported versions

Only the latest released version receives security fixes.

## Supply chain posture

- CI runs with a read-only default token; all GitHub Actions are pinned to full
  commit SHAs.
- Dependencies are scanned with govulncheck and Trivy on every push and weekly,
  and kept current by Dependabot.
- Planned for the first public release: signed release checksums (cosign) and
  build provenance (SLSA), plus CodeQL and OpenSSF Scorecard.

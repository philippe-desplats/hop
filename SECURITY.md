# Security Policy

## Reporting a vulnerability

Please report security issues privately, not through public issues. Open a private advisory at https://github.com/philippe-desplats/hop/security/advisories/new (the "Report a vulnerability" button under this repository's Security tab).

Include a description, reproduction steps, and the affected version (`hop version`). Expect an acknowledgement within 72 hours and a remediation plan once the report is triaged.

## Supported versions

Only the latest released version receives security fixes.

## Supply chain posture

- CI runs with a read-only default token; all GitHub Actions are pinned to full commit SHAs and hardened with step-security/harden-runner.
- Dependencies are scanned with govulncheck and Trivy on every push and weekly, static analysis with gosec and CodeQL, and the project is tracked by OpenSSF Scorecard. Dependencies are kept current by Dependabot.
- Releases ship signed checksums (cosign keyless), an SBOM (syft), and SLSA build provenance attestation.

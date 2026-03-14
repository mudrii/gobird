# Security Policy

## Supported versions

Security fixes are applied to the latest release line.

| Version | Supported |
|---|---|
| `26.03.x` | yes |

## Reporting a vulnerability

Please do not open a public issue for a suspected security vulnerability.

Instead, report it privately with:
- affected version
- reproduction details
- impact summary
- any suggested mitigation

If no private channel is yet configured, open a minimal GitHub issue asking for a security contact path without disclosing exploit details.

## Scope notes

`gobird` depends on unofficial X/Twitter web APIs and browser cookie extraction. Reports involving credential handling, request signing/headers, token leakage, or unsafe local file access are in scope.

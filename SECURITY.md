# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.1.x   | :white_check_mark: |
| < 0.1   | :x:                |

## Reporting a Vulnerability

We take security seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

**DO NOT** open a public GitHub issue for security vulnerabilities.

Instead, please report them via email to: **security@gitant.io**

### What to Include

Please include the following in your report:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Acknowledgment**: Within 24 hours
- **Initial Assessment**: Within 72 hours
- **Fix Development**: Within 7 days for critical, 30 days for others
- **Public Disclosure**: After fix is released

### Bug Bounty

We offer bug bounties for critical vulnerabilities:

| Severity | Bounty |
|----------|--------|
| Critical (RCE, Auth Bypass) | $500-$2000 |
| High (Data Leak, CSRF) | $200-$500 |
| Medium (XSS, Info Disclosure) | $50-$200 |
| Low (Minor Issues) | $25-$50 |

### Scope

The following are in scope:

- gitant-daemon
- gitant-cli
- gitant-mcp
- gitant-web
- Smart contracts (Solidity)

### Out of Scope

- Social engineering
- Physical attacks
- Denial of service
- Third-party dependencies (report to them directly)

## Security Features

### Authentication
- UCAN capability tokens
- HTTP Signatures (RFC 9421)
- OAuth2 integration
- API key authentication
- LDAP integration
- TOTP 2FA

### Authorization
- Role-based access control (RBAC)
- Scoped UCAN capabilities
- Repository-level permissions
- Branch protection rules

### Data Protection
- Encrypted secrets at rest
- SHA-256 hashed API keys
- Bcrypt password hashing
- TLS encryption in transit

### Network Security
- CORS validation
- CSRF protection
- Rate limiting
- Input validation
- SSRF protection
- WebSocket origin validation

### Audit
- Request logging
- Activity tracking
- Webhook notifications
- Anomaly detection

## Security Contacts

- **Email**: security@gitant.io
- **PGP Key**: [Available on request]

## Acknowledgments

We thank the following security researchers:

- (Be the first to report a vulnerability!)

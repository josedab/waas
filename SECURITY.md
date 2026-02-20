# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |

## Reporting a Vulnerability

We take the security of WaaS seriously. If you discover a security vulnerability, please report it responsibly.

### How to Report

1. **Do NOT open a public GitHub issue for security vulnerabilities.**
2. Email your findings to **security@waas-project.dev**.
3. If you prefer encrypted communication, you can use [GitHub's private vulnerability reporting](https://github.com/waas-project/waas/security/advisories/new) to securely share details with the maintainers.
4. Include as much detail as possible:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

### What to Expect

- **Acknowledgment**: We will acknowledge receipt of your report within **48 hours**.
- **Assessment**: We will provide an initial assessment within **5 business days**.
- **Resolution**: We aim to release a fix within **30 days** of confirming the vulnerability, depending on complexity.
- **Disclosure**: We will coordinate public disclosure with you after the fix is released.

### Encrypted Reports

If you need to send sensitive information, you can use [GitHub's private vulnerability reporting](https://github.com/waas-project/waas/security/advisories/new) to securely share details with the maintainers.

## Scope

The following are in scope for security reports:

- Authentication and authorization bypasses
- SQL injection, XSS, CSRF, or other injection attacks
- Sensitive data exposure (credentials, tokens, PII)
- Webhook signature verification bypasses
- Privilege escalation
- Denial of service vulnerabilities in the API or delivery pipeline

## Out of Scope

- Vulnerabilities in third-party dependencies (report these upstream, but let us know)
- Social engineering attacks
- Physical security
- Issues in environments running unsupported versions

## Safe Harbor

We support safe harbor for security researchers who:

- Act in good faith to avoid privacy violations, data destruction, or service disruption
- Only interact with accounts you own or with explicit permission
- Report vulnerabilities promptly and do not exploit them beyond what is necessary to demonstrate the issue
- Do not publicly disclose the vulnerability before we have addressed it

We will not pursue legal action against researchers who follow this policy.

## Recognition

We appreciate the security research community. With your permission, we will acknowledge your contribution in our release notes and security advisories.

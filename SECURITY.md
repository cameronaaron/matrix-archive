# Security Policy

## Supported Versions

We provide security updates for the following versions:

| Version | Supported          |
| ------- | ------------------ |
| Latest  | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in Matrix Archive, please report it responsibly:

### Private Disclosure

1. **Do NOT** open a public issue for security vulnerabilities
2. Email security reports to the maintainers
3. Provide detailed information about the vulnerability
4. Allow reasonable time for the issue to be addressed before public disclosure

### What to Include

When reporting a security vulnerability, please include:

- Description of the vulnerability
- Steps to reproduce the issue
- Potential impact and attack scenarios
- Any suggested fixes or mitigations
- Your contact information for follow-up

## Security Considerations

### Data Privacy

Matrix Archive handles sensitive data including:
- Matrix room messages and content
- User credentials and authentication tokens
- Encryption keys and cryptographic material

### Best Practices for Users

- **Keep credentials secure**: Never share or commit authentication files
- **Review exports**: Ensure exported data doesn't contain sensitive information before sharing
- **Use secure channels**: Transfer credentials and recovery keys through secure channels only
- **Regular updates**: Keep the software updated to receive security patches
- **Environment isolation**: Use appropriate file permissions for database and credential files

### Security Features

- Credentials are stored locally with appropriate file permissions
- Database files are created with restricted access
- Sensitive data is excluded from logs and error messages
- Export files can be configured to use local paths instead of Matrix URLs

## Responsible Disclosure

We appreciate security researchers who help keep Matrix Archive and its users safe. We are committed to working with the security community to verify and respond to legitimate security issues.

### Timeline

- **Initial Response**: Within 48 hours of receiving a report
- **Assessment**: Within 1 week for initial assessment
- **Resolution**: Timeline varies based on complexity and severity
- **Disclosure**: Public disclosure after fix is available and users have had time to update

Thank you for helping keep Matrix Archive secure!
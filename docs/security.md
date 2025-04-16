# TailPost Security Features

This document describes the security features available in TailPost, including TLS support, authentication mechanisms, and data encryption.

## TLS Configuration

TailPost supports TLS for securing all communications between the agent and the server. TLS ensures that data is encrypted during transit and provides authentication of the server.

### TLS Configuration Options

In your configuration file, you can specify the following TLS options:

```yaml
security:
  tls:
    enabled: true                     # Enable TLS
    cert_file: /path/to/cert.crt      # Client certificate for mutual TLS
    key_file: /path/to/key.key        # Client key for mutual TLS
    ca_file: /path/to/ca.crt          # CA certificate for server validation
    insecure_skip_verify: false       # Skip server certificate verification (not recommended for production)
    server_name: logs.example.com     # Server name to verify against certificate
    min_version: tls12                # Minimum TLS version (tls10, tls11, tls12, tls13)
    max_version: tls13                # Maximum TLS version (optional)
    prefer_server_cipher_suites: true # Prefer server's cipher suites over client's
```

### TLS Best Practices

1. Always use TLS in production environments
2. Set a minimum version of TLS 1.2 (`tls12`) or higher
3. Use a valid certificate from a trusted Certificate Authority
4. For high-security environments, use mutual TLS by providing both client and server certificates
5. Never set `insecure_skip_verify` to `true` in production

## Authentication

TailPost supports multiple authentication methods to secure access to your log data.

### Available Authentication Methods

1. **Basic Authentication**: Username and password authentication
2. **Token Authentication**: Bearer token-based authentication
3. **OAuth2 Authentication**: Client credentials OAuth2 flow
4. **Custom Header Authentication**: Custom HTTP headers for authentication

### Authentication Configuration

Configure authentication in your configuration file:

```yaml
security:
  auth:
    # Basic auth
    type: basic
    username: myuser
    password: mypassword  # Or use ${PASSWORD_ENV_VAR} for environment variables

    # Token auth
    type: token
    token_file: /path/to/token.txt  # File containing the token

    # OAuth2 auth
    type: oauth2
    client_id: my-client
    client_secret: my-secret  # Or use ${SECRET_ENV_VAR}
    token_url: https://auth.example.com/oauth/token
    scopes:
      - logs:write

    # Custom header auth
    type: header
    headers:
      X-API-Key: your-api-key
      X-Tenant-ID: your-tenant-id
```

### Secure Password Handling

For Basic Authentication and OAuth2, it's recommended to:

1. Use environment variables for secrets instead of hardcoding in config
2. Use a secret manager when available
3. Ensure proper file permissions for config files containing credentials

## Data Encryption

TailPost can encrypt log data before sending it to the server, providing an additional layer of security for sensitive information.

### Encryption Configuration

Configure encryption in your configuration file:

```yaml
security:
  encryption:
    enabled: true
    type: aes                      # Encryption algorithm (aes or chacha20poly1305)
    key_file: /path/to/key.bin     # File containing the encryption key
    key_env: ENCRYPTION_KEY_VAR    # Alternative: environment variable containing the key
    key_id: key-2023               # Identifier for the key (useful for key rotation)
    rotation_days: 90              # How often to rotate keys
```

### Key Management

For secure key management:

1. Use a 32-byte (256-bit) key for both AES and ChaCha20-Poly1305
2. Store keys securely, preferably in a key management system
3. Implement key rotation regularly
4. Limit access to encryption keys to only necessary services

### Generating Secure Encryption Keys

You can generate a secure random key using the following commands:

```bash
# Generate a 32-byte key and encode as hex
openssl rand -hex 32 > /path/to/key.bin

# For environment variable usage
export ENCRYPTION_KEY_VAR=$(openssl rand -hex 32)
```

## Security Best Practices

1. **Defense in Depth**: Use all security features together for maximum protection
2. **Principle of Least Privilege**: Limit access to only what's necessary
3. **Regular Updates**: Keep TailPost and its dependencies up to date
4. **Audit Logging**: Enable telemetry and monitor for security events
5. **Network Security**: Use firewalls and network policies to restrict access
6. **Environmental Security**: Secure the systems where TailPost agents run

## Security Features Roadmap

Future security enhancements planned for TailPost:

1. Key rotation automation
2. Integration with external key management systems (HashiCorp Vault, AWS KMS, etc.)
3. Certificate auto-renewal
4. Intrusion detection features
5. Enhanced audit logging
6. Compliance reporting for standards like PCI-DSS, HIPAA, etc.

## Reporting Security Issues

If you discover a security vulnerability in TailPost, please follow the responsible disclosure process by emailing security@tailpost.example.com. Please do not disclose security vulnerabilities publicly until we've had a chance to address them. 
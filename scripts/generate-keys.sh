#!/bin/bash
set -e

echo "🔑 Tiga Security Keys Generator"
echo "================================"
echo ""

# Generate JWT Secret (64 bytes, base64 encoded = 88 characters)
echo "Generating JWT_SECRET..."
JWT_SECRET=$(openssl rand -base64 48)
echo "✅ JWT_SECRET generated"

# Generate Encryption Key (32 bytes, base64 encoded = 44 characters)
echo "Generating ENCRYPTION_KEY..."
ENCRYPTION_KEY=$(openssl rand -base64 32)
echo "✅ ENCRYPTION_KEY generated"

# Generate Credential Key (32 bytes, base64 encoded = 44 characters)
echo "Generating CREDENTIAL_KEY..."
CREDENTIAL_KEY=$(openssl rand -base64 32)
echo "✅ CREDENTIAL_KEY generated"

echo ""
echo "================================"
echo "Generated Security Keys:"
echo "================================"
echo ""
echo "# Add these to your .env file or config.yaml"
echo ""
echo "JWT_SECRET=\"$JWT_SECRET\""
echo "ENCRYPTION_KEY=\"$ENCRYPTION_KEY\""
echo "CREDENTIAL_KEY=\"$CREDENTIAL_KEY\""
echo ""
echo "================================"
echo "config.yaml format:"
echo "================================"
echo ""
echo "security:"
echo "  jwt_secret: \"$JWT_SECRET\""
echo "  encryption_key: \"$ENCRYPTION_KEY\""
echo ""
echo "database_management:"
echo "  credential_key: \"$CREDENTIAL_KEY\""
echo ""
echo "================================"
echo ""
echo "⚠️  IMPORTANT:"
echo "  1. Save these keys securely"
echo "  2. Never commit them to version control"
echo "  3. Use different keys for different environments"
echo ""

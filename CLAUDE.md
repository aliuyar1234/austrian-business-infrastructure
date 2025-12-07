# austrian-business-infrastructure Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-12-07

## Active Technologies
- Go 1.23+ + Cobra (CLI), errgroup (parallelism), argon2 (KDF), encoding/xml (SOAP/XML) (002-vision-completion-roadmap)
- AES-256-GCM encrypted file-based credential store (existing from Spec 001) (002-vision-completion-roadmap)
- Go 1.23+ + Cobra v1.8.1 (CLI), golang.org/x/crypto (encryption), golang.org/x/sync (parallelism), mark3labs/mcp-go v0.43.2 (MCP server) (004-full-vision-completion)
- Encrypted file-based credential store (`accounts.enc`), no external database (004-full-vision-completion)

- Go 1.23+ + Cobra (CLI framework), encoding/xml (SOAP), crypto/aes + crypto/cipher (encryption) (001-finanzonline-cli)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.23+

## Code Style

Go 1.23+: Follow standard conventions

## Recent Changes
- 004-full-vision-completion: Added Go 1.23+ + Cobra v1.8.1 (CLI), golang.org/x/crypto (encryption), golang.org/x/sync (parallelism), mark3labs/mcp-go v0.43.2 (MCP server)
- 002-vision-completion-roadmap: Added Go 1.23+ + Cobra (CLI), errgroup (parallelism), argon2 (KDF), encoding/xml (SOAP/XML)

- 001-finanzonline-cli: Added Go 1.23+ + Cobra (CLI framework), encoding/xml (SOAP), crypto/aes + crypto/cipher (encryption)

<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->

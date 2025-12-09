# Documentation

Welcome to the Austrian Business Infrastructure documentation.

## Quick Links

- [Setup & Installation](setup.md) - Get started with the platform
- [Configuration](configuration.md) - Environment variables and settings
- [API Reference](api-reference.md) - REST API endpoints

## Module Documentation

- [FinanzOnline](modules/finanzonline.md) - Austrian tax authority integration
- [ELDA](modules/elda.md) - Social insurance registration
- [Firmenbuch](modules/firmenbuch.md) - Company register lookups
- [E-Rechnung](modules/e-rechnung.md) - Electronic invoicing (XRechnung/ZUGFeRD)
- [SEPA](modules/sepa.md) - Payment processing
- [MCP Server](modules/mcp.md) - AI integration via Model Context Protocol

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Frontend (SvelteKit)                    │
├─────────────────────────────────────────────────────────────┤
│                        REST API (Go)                         │
├──────────┬──────────┬──────────┬──────────┬─────────────────┤
│ FinanzOn │  ELDA    │Firmenbuch│E-Rechnung│      SEPA       │
│  line    │          │          │          │                 │
├──────────┴──────────┴──────────┴──────────┴─────────────────┤
│                    Background Jobs                           │
├─────────────────────────────────────────────────────────────┤
│              PostgreSQL  │  Redis  │  S3 Storage            │
└─────────────────────────────────────────────────────────────┘
```

## Getting Help

- Check the [Configuration](configuration.md) guide for environment setup
- See module-specific docs for integration details
- Open an issue on GitHub for bugs or feature requests

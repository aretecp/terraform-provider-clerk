# Terraform Provider for Clerk

A Terraform provider for managing [Clerk](https://clerk.com) resources as infrastructure-as-code.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (to build the provider)

## Using the Provider

```hcl
terraform {
  required_providers {
    clerk = {
      source = "aretecp/clerk"
    }
  }
}

provider "clerk" {
  # Set the CLERK_API_KEY environment variable
}
```

See the [documentation](https://registry.terraform.io/providers/aretecp/clerk/latest/docs) for full resource reference.

## Supported Resources

- `clerk_jwt_template` — Manage JWT templates

## Authentication

Set the `CLERK_API_KEY` environment variable to your Clerk secret key, or configure it directly in the provider block:

```hcl
provider "clerk" {
  api_key = "sk_test_..."
}
```

## Development

### Build

```bash
make build
```

### Run Acceptance Tests

Acceptance tests create real resources against the Clerk API.

```bash
export CLERK_API_KEY="sk_test_..."
make testacc
```

### Local Installation

Add a dev override to `~/.terraformrc`:

```hcl
provider_installation {
  dev_overrides {
    "aretecp/clerk" = "/Users/<you>/go/bin"
  }
  direct {}
}
```

Then:

```bash
make install
```

### Generate Documentation

```bash
go generate ./...
```

## License

[Mozilla Public License v2.0](./LICENSE)

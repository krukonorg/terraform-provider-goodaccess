# Terraform Provider: GoodAccess

![Terraform](https://img.shields.io/badge/Terraform-Provider-purple?logo=terraform)
![Status](https://img.shields.io/badge/status-experimental-orange)

## Overview

The `krukonorg/goodaccess` Terraform provider allows you to manage **GoodAccess** resources using infrastructure-as-code, including:

- Systems
- Access Cards
- Access Card ↔ System Relations

This provider is in **active development** and is considered **experimental**. Breaking changes may occur in future releases.

---

## ⚙️ Supported Resources

- `goodaccess_system`
- `goodaccess_access_card`
- `goodaccess_relation_ac_s`

---

## 🚀 Getting Started

### Install via Terraform Registry

```hcl
terraform {
  required_providers {
    goodaccess = {
      source  = "krukonorg/goodaccess"
      version = "0.1.0"
    }
  }
}

provider "goodaccess" {
  token = "your_real_goodaccess_api_token"
}
```



📦 Example Usage
```
resource "goodaccess_system" "example" {
name     = "My System"
host     = "https://example.com"
uri      = "https://example.com"
port     = "8080"
protocol = "UDP"
}

resource "goodaccess_access_card" "example" {
name        = "My Access Card"
description = "Managed by Terraform"
}

resource "goodaccess_relation_ac_s" "example" {
access_card_id = goodaccess_access_card.example.id
system_id      = goodaccess_system.example.id
}
```
🚧 Development Status

This provider is currently under active development. Use at your own risk in production environments. Contributions and issue reports are welcome!
🧩 Contributing

    Fork the repository

    Create a new feature branch

    Submit a pull request

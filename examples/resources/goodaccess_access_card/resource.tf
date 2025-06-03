# Copyright (c) HashiCorp, Inc.

resource "goodaccess_access_card" "example" {
  name        = "Access Card from TF"
  description = "Managed by Terraform"
}

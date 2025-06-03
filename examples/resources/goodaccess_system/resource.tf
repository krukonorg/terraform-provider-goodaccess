# Copyright (c) HashiCorp, Inc.

resource "goodaccess_system" "example" {
  name     = "GoodAccess from tf"
  host     = "https://goodaccess22.com"
  uri      = "https://goodaccess22.com"
  port     = "8081"
  protocol = "UDP"
}
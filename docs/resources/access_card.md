---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "goodaccess_access_card Resource - goodaccess"
subcategory: ""
description: |-
  
---

# goodaccess_access_card (Resource)



## Example Usage

```terraform
resource "goodaccess_access_card" "example" {
  name        = "Access Card from TF"
  description = "Managed by Terraform"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String)

### Optional

- `description` (String)

### Read-Only

- `id` (String) The ID of this resource.

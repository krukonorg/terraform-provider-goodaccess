
resource "goodaccess_relation_ac_s" "example" {
  access_card_id = goodaccess_access_card.example.id
  system_id      = goodaccess_system.example.id
}
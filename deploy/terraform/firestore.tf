# deploy/firestore/indexes.json is the sole index and TTL contract.
locals {
  firestore_contract = jsondecode(file("${path.module}/../firestore/indexes.json"))
  firestore_indexes = {
    for index in local.firestore_contract.indexes :
    "${index.collectionGroup}_${index.fields[1].fieldPath}" => index
  }
  firestore_ttl_overrides = {
    for override in local.firestore_contract.fieldOverrides :
    "${override.collectionGroup}_${override.fieldPath}" => override
    if try(override.ttl, false)
  }
}

resource "google_firestore_index" "delivery" {
  for_each    = local.firestore_indexes
  project     = var.project_id
  database    = "(default)"
  collection  = each.value.collectionGroup
  query_scope = each.value.queryScope
  dynamic "fields" {
    for_each = each.value.fields
    content {
      field_path = fields.value.fieldPath
      order      = fields.value.order
    }
  }
}

resource "google_firestore_field" "expires_at_ttl" {
  for_each   = local.firestore_ttl_overrides
  project    = var.project_id
  database   = "(default)"
  collection = each.value.collectionGroup
  field      = each.value.fieldPath
  ttl_config {}
}

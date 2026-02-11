resource "clerk_organization" "example" {
  name                    = "My Organization"
  slug                    = "my-org"
  max_allowed_memberships = 100
}

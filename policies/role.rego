package authz

import data.authz

# Only ADMIN can update user roles
allow {
    input.action == "update_user_role"
    authz.is_admin
}

deny[reason] {
    input.action == "update_user_role"
    not authz.is_admin
    reason := sprintf("User %v (role: %v) cannot update user roles", [input.user.email, authz.user_role])
}
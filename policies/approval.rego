package authz

import data.authz

# ─── APPROVE REQUEST ──────────────────────────────────────────────────────
allow {
    input.action == "approve_request"
    authz.is_admin
}

allow {
    input.action == "approve_request"
    authz.user_role == "ENGINEERING"
    input.resource == "engineering-system"
}

allow {
    input.action == "approve_request"
    authz.user_role == "SUPPORT"
    input.resource == "support-system"
}

deny[reason] {
    input.action == "approve_request"
    input.user.role == "EMPLOYEE"
    reason := "EMPLOYEE cannot approve requests"
}

deny[reason] {
    input.action == "approve_request"
    not authz.is_admin
    not (authz.user_role == "ENGINEERING" and input.resource == "engineering-system")
    not (authz.user_role == "SUPPORT" and input.resource == "support-system")
    reason := sprintf("User role %v cannot approve resource %v", [authz.user_role, input.resource])
}

# ─── REJECT REQUEST ──────────────────────────────────────────────────────
allow {
    input.action == "reject_request"
    authz.is_admin
}

allow {
    input.action == "reject_request"
    authz.user_role == "ENGINEERING"
    input.resource == "engineering-system"
}

allow {
    input.action == "reject_request"
    authz.user_role == "SUPPORT"
    input.resource == "support-system"
}

deny[reason] {
    input.action == "reject_request"
    input.user.role == "EMPLOYEE"
    reason := "EMPLOYEE cannot reject requests"
}

# ─── MARK UNDER REVIEW ──────────────────────────────────────────────────
allow {
    input.action == "mark_under_review"
    authz.is_admin
}

allow {
    input.action == "mark_under_review"
    authz.user_role == "ENGINEERING"
    input.resource == "engineering-system"
}

allow {
    input.action == "mark_under_review"
    authz.user_role == "SUPPORT"
    input.resource == "support-system"
}

deny[reason] {
    input.action == "mark_under_review"
    input.user.role == "EMPLOYEE"
    reason := "EMPLOYEE cannot mark requests under review"
}

# ─── UPDATE USER ROLE (admin only) ──────────────────────────────────────
allow {
    input.action == "update_user_role"
    authz.is_admin
}

deny[reason] {
    input.action == "update_user_role"
    not authz.is_admin
    reason := sprintf("User %v (role: %v) cannot update user roles", [input.user.email, authz.user_role])
}
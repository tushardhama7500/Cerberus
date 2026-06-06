package authz

default permitted = false

known_actions := {
    "create_request",
    "upload_screenshot",
    "view_own_requests",
    "view_all_requests",
    "update_user_role",
    "approve_request",
    "reject_request",
    "mark_under_review",
}

allow := {
    "allow": permitted,
    "reason": decision_reason,
    "details": {
        "action": input.action,
        "role": user_role,
        "effective_role": effective_role,
        "user_department": user_department,
        "request_department": request_department,
    },
}

decision_reason := "Access granted" {
    permitted
}

decision_reason := reason {
    not permitted
    reason := deny[_]
}

decision_reason := "Access denied" {
    not permitted
    count(deny) == 0
}

# SUPER_ADMIN override.
permitted {
    is_super_admin
}

# ADMIN capabilities.
permitted {
    input.action == "update_user_role"
    is_admin
}

permitted {
    input.action == "view_all_requests"
    is_admin
}

# EMPLOYEE workflow.
# These preserve the existing request creation and screenshot upload flow.
permitted {
    input.action == "create_request"
    is_employee
    input.user.is_active == true
}

permitted {
    input.action == "upload_screenshot"
    is_employee
    input.user.is_active == true
    owns_request
}

permitted {
    input.action == "view_own_requests"
    is_employee
    owns_request
}

# Optional compatibility:
# Existing ADMIN users can still perform normal employee-style actions.
permitted {
    input.action == "create_request"
    is_admin
}

permitted {
    input.action == "upload_screenshot"
    is_admin
}

permitted {
    input.action == "view_own_requests"
    is_admin
}

deny[reason] {
    input.action == "update_user_role"
    not is_super_admin
    not is_admin
    reason := sprintf(
        "User %v with role %v cannot update user roles. Required role: SUPER_ADMIN or ADMIN.",
        [input.user.email, effective_role],
    )
}

deny[reason] {
    input.action == "view_all_requests"
    not is_super_admin
    not is_admin
    reason := sprintf(
        "User %v with role %v cannot view all requests. Required role: SUPER_ADMIN or ADMIN.",
        [input.user.email, effective_role],
    )
}

deny[reason] {
    input.action == "create_request"
    not is_super_admin
    not is_admin
    not is_employee
    reason := sprintf(
        "User %v with role %v cannot create access requests.",
        [input.user.email, effective_role],
    )
}

deny[reason] {
    input.action == "upload_screenshot"
    not is_super_admin
    not is_admin
    not is_employee
    reason := sprintf(
        "User %v with role %v cannot upload screenshots.",
        [input.user.email, effective_role],
    )
}

deny[reason] {
    input.action == "view_own_requests"
    not is_super_admin
    not is_admin
    not is_employee
    reason := sprintf(
        "User %v with role %v cannot view employee request history.",
        [input.user.email, effective_role],
    )
}

deny[reason] {
    not known_actions[input.action]
    reason := sprintf("Unsupported authorization action: %v.", [input.action])
}
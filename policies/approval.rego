package authz

review_actions := {
    "approve_request",
    "reject_request",
    "mark_under_review",
}

permitted {
    review_actions[input.action]
    can_review
}

deny[reason] {
    review_actions[input.action]
    is_employee
    reason := sprintf(
        "User %v with role EMPLOYEE cannot %v. Employees can create requests, upload screenshots, and view their own requests only.",
        [input.user.email, input.action],
    )
}

deny[reason] {
    review_actions[input.action]
    is_manager
    not same_department
    reason := sprintf(
        "Manager %v belongs to department %v and cannot %v a request for department %v.",
        [input.user.email, user_department, input.action, request_department],
    )
}

deny[reason] {
    review_actions[input.action]
    is_approver
    not same_department
    reason := sprintf(
        "Approver %v belongs to department %v and cannot %v a request for department %v.",
        [input.user.email, user_department, input.action, request_department],
    )
}

deny[reason] {
    review_actions[input.action]
    not is_super_admin
    not is_admin
    not is_manager
    not is_approver
    not is_employee
    reason := sprintf(
        "User %v has unsupported role %v and cannot %v.",
        [input.user.email, user_role, input.action],
    )
}
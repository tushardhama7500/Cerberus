package authz

# --------------------------------------------------------------------
# ENTERPRISE IAM ROLES
# --------------------------------------------------------------------

enterprise_roles := {
    "SUPER_ADMIN",
    "ADMIN",
    "MANAGER",
    "APPROVER",
    "EMPLOYEE",
}

# --------------------------------------------------------------------
# DEPARTMENTS
# --------------------------------------------------------------------

departments := {
    "ENGINEERING",
    "SUPPORT",
    "FINANCE",
    "HR",
    "SALES",
}

# --------------------------------------------------------------------
# RESOURCE → DEPARTMENT MAPPING
# --------------------------------------------------------------------

resource_to_department := {
    "engineering-system": "ENGINEERING",
    "support-system": "SUPPORT",
    "finance-system": "FINANCE",
    "hr-system": "HR",
    "sales-system": "SALES",
}

# --------------------------------------------------------------------
# SAFE DEFAULTS
# --------------------------------------------------------------------

default user_role := ""
default effective_role := ""
default user_department := ""
default request_department := ""
default request_owner_email := ""

# --------------------------------------------------------------------
# USER ROLE
# --------------------------------------------------------------------

user_role := input.user.role

effective_role := role {
    enterprise_roles[input.user.role]
    role := input.user.role
}

# --------------------------------------------------------------------
# USER DEPARTMENT
# --------------------------------------------------------------------

user_department := dept {
    dept := input.user.department
    departments[dept]
}

# --------------------------------------------------------------------
# REQUEST DEPARTMENT
# --------------------------------------------------------------------

request_department := dept {
    input.data.request.department
    dept := input.data.request.department
    departments[dept]
}

request_department := dept {
    input.department
    dept := input.department
    departments[dept]
}

request_department := dept {
    dept := resource_to_department[input.resource]
}

# --------------------------------------------------------------------
# REQUEST OWNER
# --------------------------------------------------------------------

request_owner_email := email {
    email := input.data.request.requester_email
}

request_owner_email := email {
    email := input.data.request.owner_email
}

# --------------------------------------------------------------------
# ROLE HELPERS
# --------------------------------------------------------------------

is_super_admin {
    effective_role == "SUPER_ADMIN"
}

is_admin {
    effective_role == "ADMIN"
}

is_manager {
    effective_role == "MANAGER"
}

is_approver {
    effective_role == "APPROVER"
}

is_employee {
    effective_role == "EMPLOYEE"
}

# --------------------------------------------------------------------
# DEPARTMENT HELPERS
# --------------------------------------------------------------------

same_department {
    user_department == request_department
}

owns_request {
    input.user.email == request_owner_email
}

# --------------------------------------------------------------------
# REVIEW HELPERS
# --------------------------------------------------------------------

can_review {
    is_super_admin
}

can_review {
    is_admin
}

can_review {
    is_manager
    same_department
}

can_review {
    is_approver
    same_department
}
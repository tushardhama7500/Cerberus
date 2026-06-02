package authz

# Define roles and their permissions
roles := {
    "ADMIN": {"all": true},
    "ENGINEERING": {"approve": true, "reject": true, "review": true},
    "SUPPORT": {"approve": true, "reject": true, "review": true},
    "EMPLOYEE": {"request": true},
}

# Helper: check if user has a specific action permission
user_can(action) {
    user_role := input.user.role
    roles[user_role][action] == true
}

# Helper: check if user is admin
is_admin {
    input.user.role == "ADMIN"
}

# Helper: get user's role
user_role := input.user.role
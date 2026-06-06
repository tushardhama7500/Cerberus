# Cerberus

## Why the Name "Cerberus"?

Cerberus is inspired by Greek mythology, where Cerberus was the multi-headed guardian dog that protected the gates of the underworld and controlled access. The name fits this project because the system is focused on access management, authorization, security, and controlling who can access protected resources.

## Overview

Cerberus is a Go-based Access Request Management System built with GraphQL. It enables employees to request access to protected organizational resources, upload supporting screenshots, and track request status through a controlled approval workflow.

The platform uses JWT authentication, Open Policy Agent (OPA) for centralized authorization, Ent ORM for database management, MySQL for persistence, AWS S3 for screenshot storage, and GraphQL for API communication.

Cerberus follows modern Identity and Access Management (IAM) principles by separating roles from departments and enforcing approval decisions through OPA policies instead of hardcoded application rules.

---

## Features

* User registration and login
* JWT-based authentication
* Open Policy Agent (OPA) authorization
* Role-Based Access Control (RBAC)
* Department-Based Access Control (DBAC)
* Department-aware approval workflow
* Access request creation and tracking
* Screenshot upload to AWS S3 using base64 GraphQL input
* Request approval, rejection, and under-review workflow
* Audit logging for security-sensitive request lifecycle actions
* User role management
* GraphQL Playground in development
* Query complexity limiting
* Production introspection disabling
* Graceful server shutdown
* Automatic Ent schema migration on startup
* Secure password hashing using bcrypt
* Fail-closed authorization behavior when OPA checks fail

---

## Tech Stack

### Backend

* Go 1.23
* Gin
* GraphQL (gqlgen)

### Database

* MySQL
* Ent ORM

### Authentication & Authorization

* JWT Authentication
* Open Policy Agent (OPA)
* OPA policy bundle mounted from `policies/`

### Cloud Services

* AWS S3

### Infrastructure

* Docker
* Docker Compose for OPA

---

## Project Structure

```text
cerberus/
|-- cmd/
|   `-- server/
|       `-- main.go              # Application entrypoint and dependency wiring
|-- config/
|   `-- config.go                # Environment loading and typed config
|-- ent/
|   |-- schema/                  # Ent entity schemas
|   |-- migrate/                 # Generated migration helpers
|   `-- ...                      # Generated Ent client, queries, mutations
|-- graph/
|   |-- schema/
|   |   `-- schema.graphqls      # GraphQL schema
|   |-- resolver/                # gqlgen resolvers and mapping helpers
|   |-- model/                   # GraphQL models
|   `-- generated.go             # Generated gqlgen execution code
|-- internal/
|   |-- auth/                    # JWT generation, validation, context claims
|   |-- authz/                   # OPA HTTP client
|   |-- middleware/              # Gin JWT and recovery middleware
|   |-- repository/              # Ent-backed persistence repositories
|   `-- service/                 # Business logic and authorization checks
|-- policies/
|   |-- approval.rego            # Department-based review policy
|   |-- role.rego                # Action and role authorization policy
|   `-- user.rego                # Role, department, ownership helpers
|-- pkg/
|   |-- errors/                  # Application error types
|   `-- s3/                      # AWS S3 upload client
|-- docker-compose.opa.yml       # Local OPA server
|-- Dockerfile                   # Production container image
|-- gqlgen.yml
|-- go.mod
|-- go.sum
|-- .env.example
|-- Architecture.md
`-- README.md
```

---

## Requirements

Before running the project, make sure you have:

* Go 1.23.4 
* MySQL running
* Docker installed for local OPA
* AWS S3 bucket configured
* Valid AWS credentials
* Git installed

---

## Environment Variables

Create a `.env` file in the project root.

Example:

```env
# Server
SERVER_PORT=8080
ENV=development

# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=cerberus

# JWT
JWT_SECRET=your_jwt_secret
JWT_EXPIRY_HOURS=24

# AWS S3
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_REGION=ap-south-1
AWS_S3_BUCKET=your_bucket_name
```

Required variables:

* `DB_USER`
* `DB_PASSWORD`
* `JWT_SECRET`
* `AWS_ACCESS_KEY_ID`
* `AWS_SECRET_ACCESS_KEY`
* `AWS_S3_BUCKET`

The current OPA client is wired to `http://localhost:8181` in `cmd/server/main.go`.

---

## Installation

Clone the repository and install Go dependencies:

```bash
git clone <repository-url>
cd cerberus
go mod download
```

Create a local environment file:

```bash
cp .env.example .env
```

Update `.env` with your MySQL, JWT, and AWS S3 settings.

---

## Database Setup

Create the MySQL database before starting the API:

```sql
CREATE DATABASE cerberus CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

The server runs Ent schema creation on startup:

```text
cmd/server/main.go
    |
    v
entClient.Schema.Create(...)
```

For local development this creates or updates the required tables:

* `users`
* `access_requests`
* `audit_logs`

For production, use versioned migrations instead of relying on auto-migration for destructive schema changes.

---

## Running the Application

Start OPA first:

```bash
docker compose -f docker-compose.opa.yml up -d
```

Run the API:

```bash
go run ./cmd/server
```

GraphQL endpoint:

```text
http://localhost:8080/query
```

GraphQL Playground is available in development:

```text
http://localhost:8080/playground
```

Health check:

```text
http://localhost:8080/health
```

---

## OPA Setup

Cerberus uses Open Policy Agent to centralize authorization decisions.

Policy files:

* `policies/user.rego`
* `policies/role.rego`
* `policies/approval.rego`

Local OPA is started with:

```bash
docker compose -f docker-compose.opa.yml up -d
```

The Docker Compose file mounts the `policies/` directory as a bundle and exposes OPA on:

```text
http://localhost:8181
```

Authorization decisions are evaluated through:

```text
POST /v1/data/authz/allow
```

The application follows a fail-closed security model. If an OPA authorization request fails, the service returns an authorization error instead of allowing the action.

---

## Authentication

Cerberus uses JWT authentication.

Public GraphQL mutations:

* `register`
* `login`

Authenticated requests include a token:

```text
Authorization: Bearer <token>
```

The JWT middleware validates the token and attaches claims to the request context. Resolvers then read those claims before calling services.

JWT claims include:

* `user_id`
* `email`
* `role`
* standard registered claims such as issuer, issued time, and expiry

Passwords are hashed with bcrypt and stored as `password_hash`. Plain text passwords are never persisted.

---

## Authorization Model

Cerberus uses OPA as the centralized authorization engine for request workflow and role-management decisions.

OPA receives authorization input in this shape:

```json
{
  "input": {
    "action": "approve_request",
    "resource": "engineering-system",
    "user": {
      "email": "approver@example.com",
      "role": "APPROVER",
      "department": "ENGINEERING",
      "is_active": true
    },
    "data": {
      "request": {
        "department": "ENGINEERING",
        "requester_email": "employee@example.com"
      }
    }
  }
}
```

OPA returns a decision:

```json
{
  "result": {
    "allow": true,
    "reason": "Access granted"
  }
}
```

Authorization is enforced in the service layer for:

* Creating access requests
* Uploading screenshots
* Marking requests under review
* Approving requests
* Rejecting requests
* Updating user roles

The `users` query is currently guarded in the resolver and only allows `ADMIN`.

---

## Roles and Departments

### Roles

* `EMPLOYEE`
* `APPROVER`
* `MANAGER`
* `ADMIN`
* `SUPER_ADMIN`

### Departments

* `ENGINEERING`
* `SUPPORT`
* `FINANCE`
* `HR`
* `SALES`

### Access Rules

#### EMPLOYEE

Can:

* Register and log in
* Create access requests when active
* Upload screenshots to their own requests when active
* View authenticated user details

Cannot:

* Approve requests
* Reject requests
* Mark requests under review
* Update user roles
* View all users

#### APPROVER

Can:

* Approve requests in their own department
* Reject requests in their own department
* Mark requests under review in their own department

Cannot:

* Review requests from other departments
* Update user roles

#### MANAGER

Can:

* Approve requests in their own department
* Reject requests in their own department
* Mark requests under review in their own department

Cannot:

* Review requests from other departments
* Update user roles

#### ADMIN

Can:

* Update user roles
* List users through the `users` query
* View all requests through OPA policy
* Create requests
* Upload screenshots
* Approve, reject, and mark requests under review

#### SUPER_ADMIN

Can:

* Perform all OPA-protected actions across all departments

Department-based approval rules are enforced through `approval.rego` and helper rules in `user.rego`.

---

## GraphQL Examples

### Register

```graphql
mutation Register {
  register(
    input: {
      email: "user@example.com"
      name: "Example User"
      password: "password123"
      department: "ENGINEERING"
    }
  ) {
    token
    user {
      id
      email
      name
      role
      department
      isActive
    }
  }
}
```

### Login

```graphql
mutation Login {
  login(
    input: {
      email: "user@example.com"
      password: "password123"
    }
  ) {
    token
    user {
      id
      email
      name
      role
      department
    }
  }
}
```

### Current User

```graphql
query Me {
  me {
    id
    email
    name
    role
    department
    isActive
  }
}
```

### Create Access Request

```graphql
mutation CreateAccessRequest {
  createAccessRequest(
    input: {
      resource: "engineering-system"
      reason: "Need production log access for incident investigation"
      managerEmail: "manager@example.com"
    }
  ) {
    id
    resource
    reason
    status
    managerEmail
    requester {
      email
      department
    }
  }
}
```

### Upload Screenshot

```graphql
mutation UploadScreenshot {
  uploadScreenshot(
    requestId: "1"
    fileName: "proof.png"
    fileBase64: "<base64-image-data>"
  ) {
    id
    screenshotUrl
    status
  }
}
```

### Mark Under Review

```graphql
mutation MarkUnderReview {
  markUnderReview(
    input: {
      requestId: "1"
      comment: "Validating business justification"
    }
  ) {
    id
    status
    reviewComment
    reviewer {
      email
      role
    }
  }
}
```

### Approve Request

```graphql
mutation ApproveRequest {
  approveRequest(
    input: {
      requestId: "1"
      comment: "Approved for 7 days"
    }
  ) {
    id
    status
    resolvedAt
  }
}
```

### Reject Request

```graphql
mutation RejectRequest {
  rejectRequest(
    input: {
      requestId: "1"
      comment: "Insufficient business justification"
    }
  ) {
    id
    status
    resolvedAt
    reviewComment
  }
}
```

### List Access Requests

```graphql
query AccessRequests {
  accessRequests(
    filter: {
      status: PENDING
      resource: "engineering"
    }
    pagination: {
      page: 1
      pageSize: 20
    }
  ) {
    nodes {
      id
      resource
      status
      requester {
        email
        department
      }
    }
    pageInfo {
      total
      hasNextPage
      hasPreviousPage
    }
  }
}
```

### Audit Logs

```graphql
query AuditLogs {
  auditLogs(requestId: "1") {
    id
    action
    actorEmail
    actorRole
    metadata
    createdAt
  }
}
```

### Update User Role

```graphql
mutation UpdateUserRole {
  updateUserRole(
    input: {
      userId: "1"
      role: APPROVER
    }
  ) {
    id
    email
    role
    department
  }
}
```

---

## Audit Logging

Cerberus records security-sensitive request lifecycle operations in the `audit_logs` table.

Tracked audit actions include:

* `REQUEST_CREATED`
* `SCREENSHOT_UPLOADED`
* `REQUEST_UNDER_REVIEW`
* `REQUEST_APPROVED`
* `REQUEST_REJECTED`
* `COMMENT_ADDED`
* `USER_CREATED`
* `USER_ROLE_CHANGED`

The current service implementation writes audit entries for:

* Access request creation
* Screenshot uploads
* Request approval
* Request rejection
* Request under-review transitions

Each audit entry stores:

* Access request ID
* Action
* Actor email
* Actor role
* Optional metadata JSON
* Creation timestamp

---

## GraphQL Code Generation

After changing GraphQL schema files:

```bash
go run github.com/99designs/gqlgen@v0.17.70 generate
```

---

## Ent Code Generation

After changing Ent schemas:

```bash
go generate ./ent
```

or:

```bash
go run entgo.io/ent/cmd/ent generate ./ent/schema
```

---

## Security Features

Cerberus includes several security-focused features:

* Password hashing using bcrypt
* JWT signature validation with HMAC signing-method checks
* JWT expiry enforcement
* OPA-based centralized authorization
* Department-aware approval restrictions
* Fail-closed authorization behavior
* Audit logging for request lifecycle actions
* Query complexity limiting
* Production introspection disabling
* Recovery middleware for safe panic handling
* CORS configuration for frontend origins
* Principle of Least Privilege enforcement through roles and departments
* AWS S3 file type validation for screenshots
* AWS S3 screenshot size limit of 5 MB
* Private S3 object upload by default

---

## Production Notes

For production deployments:

* Set `ENV=production`
* Use a strong `JWT_SECRET`
* Store secrets in environment variables or a secrets manager
* Configure CORS to allow only trusted frontend domains
* Use versioned database migrations
* Do not rely on auto-migration for destructive schema changes
* Keep AWS credentials secure
* Prefer IAM roles for AWS access where possible
* Disable GraphQL Playground
* Disable GraphQL introspection
* Use HTTPS behind a reverse proxy or load balancer
* Deploy OPA separately and securely
* Monitor OPA health and policy bundle loading
* Review audit logging coverage before handling production security workflows
* Avoid returning public S3 URLs if screenshots must remain private; use presigned URLs or protected download endpoints

---

## Development

Run OPA:

```bash
docker compose -f docker-compose.opa.yml up -d
```

```bash
docker compose -f docker-compose.opa.yml down
```

Run the server:

```bash
go run ./cmd/server
```

Open GraphQL Playground:

```text
http://localhost:8080/playground
```

Regenerate GraphQL code after schema changes:

```bash
go run github.com/99designs/gqlgen@v0.17.70 generate
```

Regenerate Ent code after entity schema changes:

```bash
go generate ./ent
```

---

## License

This project is currently private/internal. Add a license before publishing or distributing it.

# Cerberus


Why the Name "Cerberus"?

Cerberus is inspired by Greek mythology, where Cerberus was the multi-headed guardian dog that protected the gates of the underworld and controlled access. The name fits this project because the system is focused on access management, authorization, security, and controlling who can access protected resources.

Cerberus is a Go-based access request management backend built with GraphQL. It allows users to register, log in, create access requests, upload proof screenshots, approve or reject requests, update request status, and track audit logs.

The project uses Gin for HTTP routing, gqlgen for GraphQL, Ent for database modeling and queries, MySQL for persistence, JWT for authentication, and AWS S3 for screenshot storage.

## Features

* User registration and login
* JWT-based authentication
* Role-based access control
* Access request creation
* Screenshot upload to AWS S3
* Request approval, rejection, and review workflow
* Audit logging for request activity
* Admin user management
* GraphQL Playground in development
* Query complexity limit for GraphQL
* Production introspection disabling
* Graceful server shutdown
* Automatic Ent schema migration on startup

## Tech Stack

* Go
* Gin
* gqlgen
* Ent
* MySQL
* JWT
* AWS S3

## Project Structure

```text
cerberus/
├── cmd/
│   └── server/
│       └── main.go
├── config/
│   └── config.go
├── ent/
│   └── schema/
├── graph/
│   ├── schema/
│   ├── resolver/
│   ├── model/
│   └── generated.go
├── internal/
│   ├── auth/
│   ├── middleware/
│   ├── repository/
│   └── service/
├── pkg/
│   ├── errors/
│   └── s3/
├── gqlgen.yml
├── go.mod
├── go.sum
└── tools.go
```

## Requirements

Before running the project, make sure you have:

* Go installed
* MySQL running
* AWS S3 bucket configured
* Valid AWS credentials
* Git installed

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
DB_PASSWORD=your_password
DB_NAME=cerberus

# JWT
JWT_SECRET=change_this_to_a_very_long_random_secret_in_production
JWT_EXPIRY_HOURS=24

# AWS S3
AWS_ACCESS_KEY_ID=your_key
AWS_SECRET_ACCESS_KEY=your_secret
AWS_REGION=ap-south-1
AWS_S3_BUCKET=cerberus-screenshots
```

## Installation

Clone the repository:

```bash
git clone <repository-url>
cd cerberus
```

Install dependencies:

```bash
go mod tidy
```

## Database Setup

Create a MySQL database:

```sql
CREATE DATABASE cerberus;
```

The application runs Ent auto-migration when the server starts, so tables are created automatically.

For production, use versioned migrations instead of auto-migration.

## Run the Server

Start the server:

```bash
go run .\cmd\server\main.go
```

On Linux or macOS:

```bash
go run ./cmd/server/main.go
```

By default, the server runs on:

```text
http://localhost:8080
```

## API Endpoints

### GraphQL Endpoint

```text
POST /query
```

### GraphQL Playground

Available only when ENV is not production.

```text
GET /playground
```

Example:

```text
http://localhost:8080/playground
```

### Health Check

```text
GET /health
```

Response:

```json
{
  "service": "cerberus",
  "status": "ok"
}
```

## Authentication

Most GraphQL operations require a JWT token.

Send the token in the Authorization header:

```text
Authorization: Bearer <token>
```

You receive the token after registering or logging in.

## Roles

Cerberus supports the following roles:

* EMPLOYEE
* SUPPORT
* ENGINEERING
* ADMIN

Default registered users are created as EMPLOYEE.

Admins can update user roles and list all users.

## GraphQL Examples

### Register

```graphql
mutation Register {
  register(input: {
    email: "user@example.com"
    name: "Example User"
    password: "password123"
  }) {
    token
    user {
      id
      email
      name
      role
    }
  }
}
```

### Login

```graphql
mutation Login {
  login(input: {
    email: "user@example.com"
    password: "password123"
  }) {
    token
    user {
      id
      email
      name
      role
    }
  }
}
```

### Get Current User

```graphql
query Me {
  me {
    id
    email
    name
    role
    isActive
    createdAt
  }
}
```

### Create Access Request

```graphql
mutation CreateAccessRequest {
  createAccessRequest(input: {
    resource: "Production Database"
    reason: "Need access for debugging an incident"
    managerEmail: "manager@example.com"
  }) {
    id
    resource
    reason
    status
    managerEmail
    createdAt
  }
}
```

### List Access Requests

```graphql
query AccessRequests {
  accessRequests(
    filter: {
      status: PENDING
    }
    pagination: {
      page: 1
      pageSize: 10
    }
  ) {
    nodes {
      id
      resource
      reason
      status
      requester {
        id
        email
        name
      }
      createdAt
    }
    pageInfo {
      total
      hasNextPage
      hasPreviousPage
    }
  }
}
```

### Approve Request

```graphql
mutation ApproveRequest {
  approveRequest(input: {
    requestId: "1"
    comment: "Approved for temporary access"
  }) {
    id
    status
    reviewComment
    resolvedAt
  }
}
```

### Reject Request

```graphql
mutation RejectRequest {
  rejectRequest(input: {
    requestId: "1"
    comment: "Insufficient business justification"
  }) {
    id
    status
    reviewComment
    resolvedAt
  }
}
```

### Mark Request Under Review

```graphql
mutation MarkUnderReview {
  markUnderReview(input: {
    requestId: "1"
    comment: "Need more details before approval"
  }) {
    id
    status
    reviewComment
  }
}
```

### Upload Screenshot

```graphql
mutation UploadScreenshot {
  uploadScreenshot(
    requestId: "1"
    fileBase64: "<base64-file-data>"
    fileName: "proof.png"
  ) {
    id
    screenshotUrl
  }
}
```

### Get Audit Logs

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
  updateUserRole(input: {
    userId: "1"
    role: ADMIN
  }) {
    id
    email
    name
    role
  }
}
```

## GraphQL Code Generation

This project uses gqlgen.

After changing GraphQL schema files, regenerate GraphQL code:

```bash
go run github.com/99designs/gqlgen generate
```

Then tidy dependencies:

```bash
go mod tidy
```

## Ent Code Generation

If Ent schemas are changed, regenerate Ent code:

```bash
go generate ./ent
```

or:

```bash
go run entgo.io/ent/cmd/ent generate ./ent/schema
```

## Production Notes

For production deployments:

* Set `ENV=production`
* Use a strong `JWT_SECRET`
* Configure proper CORS origins
* Use versioned database migrations
* Do not rely on auto-migration for destructive schema changes
* Keep AWS credentials secure
* Disable GraphQL Playground
* Disable GraphQL introspection
* Use HTTPS behind a reverse proxy or load balancer

## Security

Cerberus includes several security-focused features:

* Passwords are hashed using bcrypt
* JWT is used for stateless authentication
* Password hashes are marked sensitive in Ent
* GraphQL query complexity is limited
* GraphQL introspection is disabled in production
* Request actions are recorded in audit logs
* Role checks protect admin-only operations

## Development

Run the server in development mode:

```env
ENV=development
```

Start the app:

```bash
go run .\cmd\server\main.go
```

Open GraphQL Playground:

```text
http://localhost:8080/playground
```

## License

This project is currently private/internal. Add a license before publishing or distributing it.

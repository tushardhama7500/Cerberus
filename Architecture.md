## Architecture

Cerberus follows a layered backend architecture. The frontend communicates with a single GraphQL endpoint, while the backend separates HTTP handling, authentication, GraphQL resolution, business logic, persistence, audit logging, and external file storage.

```text
Client / Blaze Frontend
        |
        | HTTP POST /query
        | Authorization: Bearer <JWT>
        v
+------------------------------------------------------------------+
|                         Gin HTTP Server                          |
|                                                                  |
|  +----------------+   +----------------+   +------------------+  |
|  |   CORS MW      |   |  Recovery MW   |   |   JWT Auth MW    |  |
|  |                |   |                |   |                  |  |
|  | Allows frontend|   | Handles panics |   | Reads token and  |  |
|  | origins        |   | safely         |   | attaches claims  |  |
|  +----------------+   +----------------+   +------------------+  |
|                                                                  |
|                         Route Layer                              |
|                                                                  |
|  POST /query      -> GraphQL API                                 |
|  GET /playground  -> GraphQL Playground, development only         |
|  GET /health      -> Health check                                |
|                                                                  |
|                              |                                   |
|                              v                                   |
|                       gqlgen Handler                             |
|                                                                  |
|  +------------------------------------------------------------+  |
|  | GraphQL Schema                                             |  |
|  |                                                            |  |
|  | Queries:                                                   |  |
|  | - me                                                       |  |
|  | - users                                                    |  |
|  | - accessRequest                                            |  |
|  | - accessRequests                                           |  |
|  | - auditLogs                                                |  |
|  |                                                            |  |
|  | Mutations:                                                 |  |
|  | - register                                                 |  |
|  | - login                                                    |  |
|  | - createAccessRequest                                      |  |
|  | - uploadScreenshot                                         |  |
|  | - approveRequest                                           |  |
|  | - rejectRequest                                            |  |
|  | - markUnderReview                                          |  |
|  | - updateUserRole                                           |  |
|  +------------------------------------------------------------+  |
|                              |                                   |
|             +----------------+----------------+                  |
|             |                                 |                  |
|             v                                 v                  |
|       Query Resolvers                  Mutation Resolvers         |
|             |                                 |                  |
|             +----------------+----------------+                  |
|                              |                                   |
|                              v                                   |
|                       Service Layer                              |
|                                                                  |
|  +-------------------+       +------------------------------+    |
|  | AuthService       |       | AccessRequestService         |    |
|  |                   |       |                              |    |
|  | - Register user   |       | - Create access request      |    |
|  | - Login user      |       | - Upload screenshots         |    |
|  | - Hash password   |       | - Approve requests           |    |
|  | - Generate JWT    |       | - Reject requests            |    |
|  |                   |       | - Mark under review          |    |
|  |                   |       | - Update user roles          |    |
|  |                   |       | - Write audit logs           |    |
|  +-------------------+       +------------------------------+    |
|                              |                                   |
|                              v                                   |
|                      Repository Layer                            |
|                                                                  |
|  +-------------------+   +-------------------+   +------------+ |
|  | UserRepository    |   | AccessRequestRepo |   | AuditRepo  | |
|  |                   |   |                   |   |            | |
|  | - Find user       |   | - Create request  |   | - Add log  | |
|  | - Create user     |   | - Find request    |   | - List logs| |
|  | - Update role     |   | - List requests   |   |            | |
|  | - List users      |   | - Update status   |   |            | |
|  +-------------------+   +-------------------+   +------------+ |
|                              |                                   |
|                              v                                   |
|                         Ent ORM                                  |
|                              |                                   |
|                              v                                   |
|                            MySQL                                 |
+------------------------------------------------------------------+

                              |
                              |
          +-------------------+-------------------+
          |                                       |
          v                                       v
      AWS S3                              Audit Log Records
 Screenshot storage                    Request lifecycle history
```

## Request Flow

### 1. Client Request

The Blaze frontend sends all GraphQL operations to:

```text
POST /query
```

Authenticated requests include a JWT token:

```text
Authorization: Bearer <token>
```

The frontend does not call repository or service code directly. It only communicates through GraphQL.

### 2. Gin Middleware

Every request passes through Gin middleware before reaching GraphQL.

```text
Client Request
    |
    v
CORS Middleware
    |
    v
Recovery Middleware
    |
    v
JWT Auth Middleware
    |
    v
GraphQL Handler
```

The middleware responsibilities are:

* CORS Middleware: allows approved frontend origins to call the API.
* Recovery Middleware: catches unexpected panics and returns safe error responses.
* JWT Auth Middleware: validates the JWT token and attaches user claims to the request context.

The attached claims include details such as:

* userId
* email
* role

Resolvers and services use these claims for authorization decisions.

### 3. GraphQL Handler

The gqlgen handler receives the request and routes it to the correct resolver based on the GraphQL operation.

Examples:

```text
query me                     -> Query Resolver
mutation createAccessRequest -> Mutation Resolver
mutation approveRequest      -> Mutation Resolver
query auditLogs              -> Query Resolver
```

The handler also applies:

* GraphQL schema validation
* Query complexity limit
* Introspection disabling in production

### 4. Resolver Layer

Resolvers are the bridge between GraphQL and backend business logic.

Resolvers should stay thin. Their main responsibilities are:

* Read GraphQL input
* Read authenticated user claims from context
* Convert GraphQL IDs to internal IDs
* Call the correct service method
* Convert Ent models into GraphQL response models

Example flow:

```text
createAccessRequest mutation
        |
        v
Mutation Resolver
        |
        v
AccessRequestService.Create(...)
```

### 5. Service Layer

The service layer contains the core business rules of Cerberus.

This is where the system decides things like:

* Whether a user is allowed to create a request
* Whether a user can approve or reject a request
* Whether only admins can update user roles
* How request status should change
* When audit logs should be created
* When screenshots should be uploaded to S3
* What errors should be returned for invalid operations

Main services:

* AuthService
* AccessRequestService

### 6. Repository Layer

The repository layer handles database access using Ent.

Repositories keep database queries separate from business logic.

Main repositories:

* UserRepository
* AccessRequestRepository
* AuditRepository

The service layer calls repositories, and repositories call Ent.

```text
Service Layer
    |
    v
Repository Layer
    |
    v
Ent ORM
    |
    v
MySQL
```

### 7. Database Layer

Cerberus stores core data in MySQL.

Main entities:

* users
* access_requests
* audit_logs

The database stores:

* User accounts
* Password hashes
* User roles
* Access request details
* Request status
* Manager email
* Reviewer information
* Screenshot URL
* Review comments
* Audit history

### 8. AWS S3 Integration

Screenshots are uploaded to AWS S3.

The database does not store the image file itself. It stores only the uploaded file URL.

```text
uploadScreenshot mutation
        |
        v
Mutation Resolver
        |
        v
AccessRequestService
        |
        v
S3 Client
        |
        v
AWS S3 Bucket
        |
        v
Screenshot URL saved in MySQL
```

This keeps the database lightweight and lets S3 handle binary file storage.

## Authentication Architecture

```text
Register / Login
        |
        v
AuthService
        |
        v
bcrypt password hashing / verification
        |
        v
JWT generation
        |
        v
Token returned to frontend
        |
        v
Frontend sends token in Authorization header
        |
        v
JWT middleware validates token on future requests
```

Passwords are never stored as plain text. Cerberus stores only bcrypt password hashes.

## Authorization Model

Cerberus uses role-based access control.

* EMPLOYEE
* SUPPORT
* ENGINEERING
* ADMIN

General access pattern:

### EMPLOYEE

* Can register and log in
* Can create access requests
* Can view authenticated user details

### SUPPORT / ENGINEERING

* Can review operational access requests
* Can mark requests under review
* Can approve or reject requests depending on service rules

### ADMIN

* Can manage users
* Can update user roles
* Can list users
* Can perform administrative request workflows

Authorization is enforced in the resolver and service layers using JWT claims.

## Audit Logging Architecture

Audit logs capture important lifecycle events for each access request.

Examples:

* REQUEST_CREATED
* SCREENSHOT_UPLOADED
* REQUEST_UNDER_REVIEW
* REQUEST_APPROVED
* REQUEST_REJECTED
* USER_ROLE_CHANGED

Audit logging flow:

```text
User action
    |
    v
Resolver
    |
    v
Service Layer
    |
    v
Business operation completed
    |
    v
AuditRepository creates audit log
    |
    v
Audit log stored in MySQL
```

Audit logs help the company track:

* Who performed an action
* What action was performed
* Which request was affected
* When the action happened
* Any additional metadata or comments

## Company Use Case Flow

A typical company access approval flow looks like this:

```text
Employee logs in
        |
        v
Employee creates access request
        |
        v
Request is stored as PENDING
        |
        v
Audit log records REQUEST_CREATED
        |
        v
Employee uploads screenshot or proof
        |
        v
Screenshot is uploaded to AWS S3
        |
        v
S3 URL is saved on the request
        |
        v
Audit log records SCREENSHOT_UPLOADED
        |
        v
Support / Engineering / Admin reviews request
        |
        v
Reviewer marks request UNDER_REVIEW, APPROVED, or REJECTED
        |
        v
Request status is updated in MySQL
        |
        v
Audit log records final decision
        |
        v
Frontend displays updated request status
```

## Deployment View

```text
Blaze Frontend
      |
      | HTTPS
      v
Load Balancer / Reverse Proxy
      |
      v
Cerberus Go API
      |
      +------------------> MySQL Database
      |
      +------------------> AWS S3 Bucket
```

In production:

* The frontend should call the backend over HTTPS.
* The backend should run with `ENV=production`.
* GraphQL Playground should be disabled.
* GraphQL introspection should be disabled.
* CORS should allow only trusted frontend domains.
* JWT secrets should be strong and stored securely.
* Database credentials should come from environment variables or a secrets manager.
* AWS credentials should use IAM roles where possible.

## Why This Architecture

This architecture keeps responsibilities clean:

### Gin

* HTTP routing and middleware

### gqlgen

* GraphQL schema and request execution

### Resolvers

* GraphQL input/output mapping

### Services

* Business rules and authorization decisions

### Repositories

* Database access

### Ent

* ORM and schema modeling

### MySQL

* Persistent relational storage

### AWS S3

* Screenshot file storage

The result is a backend that is easier to maintain, test, extend, and deploy for company access management workflows.

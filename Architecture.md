## Architecture

Cerberus follows a layered backend architecture. The frontend communicates with a single GraphQL endpoint, while the backend separates HTTP handling, authentication, GraphQL resolution, business logic, OPA authorization, persistence, audit logging, and external file storage.

```text
Client / Blaze Frontend
        |
        | HTTP POST /query
        | Authorization: Bearer <JWT>
        v
+-------------------------------------------------------------------------+
|                           Gin HTTP Server                               |
|                                                                         |
|  +----------------+   +----------------+   +--------------------------+ |
|  |   CORS MW      |   |  Recovery MW   |   |       JWT Auth MW        | |
|  |                |   |                |   |                          | |
|  | Allows frontend|   | Handles panics |   | Validates Bearer token   | |
|  | origins        |   | safely         |   | and attaches JWT claims  | |
|  +----------------+   +----------------+   +--------------------------+ |
|                                                                         |
|                            Route Layer                                  |
|                                                                         |
|  POST /query      -> GraphQL API                                        |
|  GET /playground  -> GraphQL Playground, development only               |
|  GET /health      -> Health check                                       |
|                                                                         |
|                                 |                                       |
|                                 v                                       |
|                          gqlgen Handler                                 |
|                                                                         |
|  +--------------------------------------------------------------------+ |
|  | GraphQL Schema                                                     | |
|  |                                                                    | |
|  | Queries:                                                           | |
|  | - me                                                               | |
|  | - users                                                            | |
|  | - accessRequest                                                    | |
|  | - accessRequests                                                   | |
|  | - auditLogs                                                        | |
|  |                                                                    | |
|  | Mutations:                                                         | |
|  | - register                                                         | |
|  | - login                                                            | |
|  | - createAccessRequest                                              | |
|  | - uploadScreenshot                                                 | |
|  | - approveRequest                                                   | |
|  | - rejectRequest                                                    | |
|  | - markUnderReview                                                  | |
|  | - updateUserRole                                                   | |
|  +--------------------------------------------------------------------+ |
|                                 |                                       |
|                +----------------+----------------+                      |
|                |                                 |                      |
|                v                                 v                      |
|          Query Resolvers                  Mutation Resolvers            |
|                |                                 |                      |
|                +----------------+----------------+                      |
|                                 |                                       |
|                                 v                                       |
|                          Service Layer                                  |
|                                                                         |
|  +-------------------+       +----------------------------------------+ |
|  | AuthService       |       | AccessRequestService                   | |
|  |                   |       |                                        | |
|  | - Register user   |       | - Create access request                | |
|  | - Login user      |       | - Upload screenshots                   | |
|  | - Hash password   |       | - Approve requests                     | |
|  | - Generate JWT    |       | - Reject requests                      | |
|  |                   |       | - Mark under review                    | |
|  |                   |       | - Update user roles                    | |
|  |                   |       | - Call OPA authorization               | |
|  |                   |       | - Write audit logs                     | |
|  +-------------------+       +----------------------------------------+ |
|                                 |                                       |
|        +------------------------+---------------------------+           |
|        |                        |                           |           |
|        v                        v                           v           |
|  Repository Layer          OPA Client                  S3 Client        |
|        |                        |                           |           |
|        v                        v                           v           |
|  +-------------------+   POST /v1/data/authz/allow     AWS S3 Bucket    |
|  | UserRepository    |                                                  |
|  | AccessRequestRepo |                                                  |
|  | AuditRepository   |                                                  |
|  +-------------------+                                                  |
|        |                                                                |
|        v                                                                |
|      Ent ORM                                                            |
|        |                                                                |
|        v                                                                |
|      MySQL                                                              |
+-------------------------------------------------------------------------+

                 +--------------------+          +----------------------+
                 | OPA Policy Bundle  |          | Audit Log Records    |
                 |                    |          |                      |
                 | - user.rego        |          | Request lifecycle    |
                 | - role.rego        |          | security history     |
                 | - approval.rego    |          |                      |
                 +--------------------+          +----------------------+
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
Gin Logger
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
* Gin Logger: records HTTP request activity.
* JWT Auth Middleware: validates a provided JWT token and attaches user claims to the request context.

The JWT middleware allows requests without a token so public GraphQL operations such as `register` and `login` can run. If a token is present and invalid, the middleware rejects the request immediately.

The attached claims include:

* userId
* email
* role

Resolvers and services use these claims for authentication context and authorization decisions.

### 3. GraphQL Handler

The gqlgen handler receives the request and routes it to the correct resolver based on the GraphQL operation.

Examples:

```text
query me                     -> Query Resolver
query users                  -> Query Resolver
query accessRequests         -> Query Resolver
mutation createAccessRequest -> Mutation Resolver
mutation uploadScreenshot    -> Mutation Resolver
mutation approveRequest      -> Mutation Resolver
mutation updateUserRole      -> Mutation Resolver
```

The handler also applies:

* GraphQL schema validation
* Query complexity limit of 300
* Introspection disabling in production

### 4. Resolver Layer

Resolvers are the bridge between GraphQL and backend business logic.

Resolvers should stay thin. Their main responsibilities are:

* Read GraphQL input
* Read authenticated user claims from context
* Convert GraphQL IDs to internal integer IDs
* Call the correct service method
* Convert Ent models into GraphQL response models

Example flow:

```text
createAccessRequest mutation
        |
        v
Mutation Resolver
        |
        | requireAuth(ctx)
        | parse requester ID
        v
AccessRequestService.Create(...)
```

Public resolver operations:

* register
* login

Protected resolver operations:

* me
* users
* accessRequest
* accessRequests
* auditLogs
* createAccessRequest
* uploadScreenshot
* approveRequest
* rejectRequest
* markUnderReview
* updateUserRole

The `users` query currently performs a resolver-level role check and only allows `ADMIN`.

### 5. Service Layer

The service layer contains the core business rules of Cerberus.

This is where the system decides things like:

* Whether a user can create a request
* Whether a manager email exists before request creation
* Whether a user can upload a screenshot
* Whether a reviewer can approve, reject, or mark a request under review
* Whether only OPA-authorized administrators can update user roles
* How request status should change
* When terminal states should set `resolved_at`
* When audit logs should be created
* When screenshots should be uploaded to S3
* What errors should be returned for invalid operations

Main services:

* AuthService
* AccessRequestService
* AuditService

### 6. OPA Authorization Layer

Cerberus uses Open Policy Agent as the centralized authorization engine.

The service layer sends authorization decisions to:

```text
POST http://localhost:8181/v1/data/authz/allow
```

OPA evaluates policies from:

* user.rego
* role.rego
* approval.rego

Authorization flow:

```text
Resolver
    |
    v
Service Layer
    |
    | Build OPA input:
    | - action
    | - resource
    | - user role
    | - user department
    | - user active state
    | - request department
    | - requester email
    v
OPA Client
    |
    | POST /v1/data/authz/allow
    v
OPA Policy Bundle
    |
    v
Decision: allow true/false
    |
    v
Service continues or returns FORBIDDEN
```

OPA-protected actions include:

* create_request
* upload_screenshot
* approve_request
* reject_request
* mark_under_review
* update_user_role
* view_own_requests
* view_all_requests

Department-based approval rules are enforced through OPA:

```text
APPROVER / MANAGER
        |
        | can review only when:
        | user.department == request.department
        v
Approve / Reject / Mark Under Review
```

`ADMIN` and `SUPER_ADMIN` have broader review permissions. `SUPER_ADMIN` has an OPA override for all OPA-protected actions.

If OPA is unavailable or returns an error, the service fails closed and does not allow the protected operation.

### 7. Repository Layer

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

Repository responsibilities:

* UserRepository
  * Create users with default `EMPLOYEE` role
  * Find users by email
  * Find users by ID
  * List users
  * Update user roles
* AccessRequestRepository
  * Create access requests
  * Find requests with requester, reviewer, and audit log edges
  * List requests with filters and pagination
  * Update request status
  * Save screenshot URLs
* AuditRepository
  * Create audit entries
  * List audit entries by request ID

### 8. Ent ORM Layer

Ent defines the database model and generates type-safe Go code.

Entity schemas:

* User
* AccessRequest
* AuditLog

Generated Ent code is used for:

* Creating records
* Updating records
* Querying records
* Loading edges
* Enforcing enum values
* Running schema creation on startup

Code generation:

```text
ent/schema/*.go
        |
        v
go generate ./ent
        |
        v
Generated Ent client and query code
```

### 9. MySQL Layer

Cerberus stores core data in MySQL.

Main tables:

* users
* access_requests
* audit_logs

The database stores:

* User accounts
* Password hashes
* User roles
* User departments
* User active state
* Access request details
* Request status
* Manager email
* Reviewer information
* Screenshot URL
* Review comments
* Resolution timestamps
* Audit history

Important user fields:

* email
* name
* password_hash
* role
* department
* is_active
* created_at
* updated_at

Important access request fields:

* requester_id
* manager_id
* reviewer_id
* resource
* reason
* status
* manager_email
* screenshot_url
* review_comment
* created_at
* updated_at
* resolved_at

Important audit log fields:

* access_request_id
* action
* actor_email
* actor_role
* metadata
* created_at

### 10. AWS S3 Integration

Screenshots are uploaded to AWS S3.

The database does not store the image file itself. It stores only the uploaded file URL.

```text
uploadScreenshot mutation
        |
        v
Mutation Resolver
        |
        | requireAuth(ctx)
        v
AccessRequestService.UploadScreenshot
        |
        | Fetch request
        | Authorize upload_screenshot through OPA
        v
S3 Client
        |
        | Decode base64
        | Validate extension
        | Enforce 5 MB limit
        | PutObject to S3
        v
AWS S3 Bucket
        |
        v
S3 URL saved in MySQL
        |
        v
Audit log records SCREENSHOT_UPLOADED
```

Allowed screenshot extensions:

* .jpg
* .jpeg
* .png
* .gif
* .webp

The S3 key format is:

```text
screenshots/{requestID}/{timestamp}_{fileName}
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
        |
        v
Claims attached to request context
        |
        v
Resolvers read claims with requireAuth(ctx)
```

Passwords are never stored as plain text. Cerberus stores only bcrypt password hashes.

JWTs are signed with HMAC and include:

* user_id
* email
* role
* issuer
* issued at
* expiry

## Authorization Model

Cerberus uses role-based and department-based access control enforced primarily through Open Policy Agent.

Roles:

* EMPLOYEE
* APPROVER
* MANAGER
* ADMIN
* SUPER_ADMIN

Departments:

* ENGINEERING
* SUPPORT
* FINANCE
* HR
* SALES

General access pattern:

### EMPLOYEE

* Can register and log in
* Can create access requests when active
* Can upload screenshots to their own requests when active
* Can view authenticated user details

### APPROVER

* Can review access requests in their own department
* Can mark requests under review in their own department
* Can approve or reject requests in their own department

### MANAGER

* Can review access requests in their own department
* Can mark requests under review in their own department
* Can approve or reject requests in their own department

### ADMIN

* Can update user roles
* Can list users through the current resolver guard
* Can create requests
* Can upload screenshots
* Can approve, reject, or mark requests under review
* Can view all requests through OPA policy

### SUPER_ADMIN

* Can perform all OPA-protected actions across all departments

OPA policy files:

```text
policies/
|-- user.rego
|-- role.rego
`-- approval.rego
```

Authorization is enforced in the service layer using JWT claims, database context, and OPA decisions.

## GraphQL Layer

GraphQL is implemented with gqlgen.

Schema file:

```text
graph/schema/schema.graphqls
```

Supported queries:

* me
* users
* accessRequest
* accessRequests
* auditLogs

Supported mutations:

* register
* login
* createAccessRequest
* uploadScreenshot
* approveRequest
* rejectRequest
* markUnderReview
* updateUserRole

The `accessRequests` query supports:

* requester email filtering
* status filtering
* resource filtering
* pagination

```text
accessRequests(filter, pagination)
        |
        v
AccessRequestConnection
        |
        +-- nodes
        `-- pageInfo
            +-- total
            +-- hasNextPage
            `-- hasPreviousPage
```

## Audit Logging Architecture

Audit logs capture important lifecycle events for each access request.

Audit actions include:

* REQUEST_CREATED
* SCREENSHOT_UPLOADED
* REQUEST_UNDER_REVIEW
* REQUEST_APPROVED
* REQUEST_REJECTED
* COMMENT_ADDED
* USER_CREATED
* USER_ROLE_CHANGED

Current service-created audit logs include:

* REQUEST_CREATED
* SCREENSHOT_UPLOADED
* REQUEST_UNDER_REVIEW
* REQUEST_APPROVED
* REQUEST_REJECTED

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
Authorization and business operation completed
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
* The actor role at the time of action
* Any additional metadata or comments

## Company Use Case Flow

A typical company access approval flow looks like this:

```text
Employee logs in
        |
        v
Employee creates access request
        |
        | OPA action: create_request
        v
Request is stored as PENDING
        |
        v
Audit log records REQUEST_CREATED
        |
        v
Employee uploads screenshot or proof
        |
        | OPA action: upload_screenshot
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
Approver / Manager / Admin reviews request
        |
        | OPA action:
        | - mark_under_review
        | - approve_request
        | - reject_request
        v
OPA enforces role and department rules
        |
        v
Reviewer marks request UNDER_REVIEW, APPROVED, or REJECTED
        |
        v
Request status is updated in MySQL
        |
        v
Terminal decisions set resolved_at
        |
        v
Audit log records review action
        |
        v
Frontend displays updated request status
```

Workflow states:

```text
PENDING
   |
   +--> UNDER_REVIEW
   |        |
   |        +--> APPROVED
   |        |
   |        `--> REJECTED
   |
   +--> APPROVED
   |
   `--> REJECTED
```

Approved and rejected requests are terminal states and cannot be changed by the current service implementation.

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
      +------------------> OPA Server
      |                    |
      |                    `--> Policy Bundle
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
* OPA should be deployed as a separate secured service.
* OPA policy bundle loading should be monitored.
* Audit logging coverage should be reviewed before handling production security workflows.

## Why This Architecture

This architecture keeps responsibilities clean:

### Gin

* HTTP routing and middleware
* CORS handling
* Recovery handling
* Health check routing

### gqlgen

* GraphQL schema and request execution
* Query complexity limiting
* Production introspection control

### Resolvers

* GraphQL input/output mapping
* Authentication claim extraction
* GraphQL ID conversion
* Service method dispatch

### Services

* Business rules
* OPA authorization calls
* Request workflow transitions
* Audit log creation
* S3 upload orchestration

### OPA

* Centralized authorization decisions
* Role-based access control
* Department-based access control
* Approval policy enforcement
* Fail-closed security boundary

### Repositories

* Database access
* Query filtering and pagination
* Entity updates
* Audit log persistence

### Ent

* ORM and schema modeling
* Type-safe database operations
* Generated query and mutation APIs

### MySQL

* Persistent relational storage
* User, request, and audit log records

### AWS S3

* Screenshot file storage
* Binary object handling outside the database

The result is a backend that is easier to maintain, test, extend, and deploy for company access management workflows.

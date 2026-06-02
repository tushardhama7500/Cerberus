package model

type User struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      Role   `json:"role"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
}

type AccessRequest struct {
	ID            string        `json:"id"`
	Resource      string        `json:"resource"`
	Reason        string        `json:"reason"`
	Status        RequestStatus `json:"status"`
	ManagerEmail  string        `json:"managerEmail"`
	ScreenshotURL *string       `json:"screenshotUrl"`
	ReviewComment *string       `json:"reviewComment"`
	Requester     *User         `json:"requester"`
	Reviewer      *User         `json:"reviewer"`
	CreatedAt     string        `json:"createdAt"`
	UpdatedAt     string        `json:"updatedAt"`
	ResolvedAt    *string       `json:"resolvedAt"`
	AuditLogs     []*AuditLog   `json:"auditLogs"`
}

type AuditLog struct {
	ID         string      `json:"id"`
	Action     AuditAction `json:"action"`
	ActorEmail string      `json:"actorEmail"`
	ActorRole  string      `json:"actorRole"`
	Metadata   *string     `json:"metadata"`
	CreatedAt  string      `json:"createdAt"`
}

type AuthPayload struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

type PageInfo struct {
	HasNextPage     bool `json:"hasNextPage"`
	HasPreviousPage bool `json:"hasPreviousPage"`
	Total           int  `json:"total"`
}

type AccessRequestConnection struct {
	Nodes    []*AccessRequest `json:"nodes"`
	PageInfo *PageInfo        `json:"pageInfo"`
}

type RegisterInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateAccessRequestInput struct {
	Resource     string `json:"resource"`
	Reason       string `json:"reason"`
	ManagerEmail string `json:"managerEmail"`
}

type ReviewRequestInput struct {
	RequestID string  `json:"requestId"`
	Comment   *string `json:"comment"`
}

type AccessRequestFilter struct {
	RequesterEmail *string        `json:"requesterEmail"`
	Status         *RequestStatus `json:"status"`
	Resource       *string        `json:"resource"`
}

type PaginationInput struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
}

type UpdateUserRoleInput struct {
	UserID string `json:"userId"`
	Role   Role   `json:"role"`
}

type Role string

const (
	RoleEmployee    Role = "EMPLOYEE"
	RoleSupport     Role = "SUPPORT"
	RoleEngineering Role = "ENGINEERING"
	RoleAdmin       Role = "ADMIN"
)

type RequestStatus string

const (
	RequestStatusPending     RequestStatus = "PENDING"
	RequestStatusApproved    RequestStatus = "APPROVED"
	RequestStatusRejected    RequestStatus = "REJECTED"
	RequestStatusUnderReview RequestStatus = "UNDER_REVIEW"
)

func (r RequestStatus) String() string { return string(r) }

type AuditAction string

const (
	AuditActionRequestCreated     AuditAction = "REQUEST_CREATED"
	AuditActionRequestApproved    AuditAction = "REQUEST_APPROVED"
	AuditActionRequestRejected    AuditAction = "REQUEST_REJECTED"
	AuditActionRequestUnderReview AuditAction = "REQUEST_UNDER_REVIEW"
	AuditActionCommentAdded       AuditAction = "COMMENT_ADDED"
	AuditActionScreenshotUploaded AuditAction = "SCREENSHOT_UPLOADED"
	AuditActionUserCreated        AuditAction = "USER_CREATED"
	AuditActionUserRoleChanged    AuditAction = "USER_ROLE_CHANGED"
)

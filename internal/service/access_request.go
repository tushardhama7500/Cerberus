package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cerberus/ent"
	"cerberus/ent/accessrequest"
	"cerberus/ent/auditlog"
	"cerberus/ent/user"
	"cerberus/internal/authz"
	"cerberus/internal/repository"
	apperrors "cerberus/pkg/errors"
	"cerberus/pkg/s3"
)

type AccessRequestService struct {
	requestRepo *repository.AccessRequestRepository
	auditRepo   *repository.AuditRepository
	userRepo    *repository.UserRepository
	s3Client    *s3.Client
	opaClient   *authz.Client
}

func NewAccessRequestService(
	requestRepo *repository.AccessRequestRepository,
	auditRepo *repository.AuditRepository,
	userRepo *repository.UserRepository,
	s3Client *s3.Client,
	opaClient *authz.Client,
) *AccessRequestService {
	return &AccessRequestService{
		requestRepo: requestRepo,
		auditRepo:   auditRepo,
		userRepo:    userRepo,
		s3Client:    s3Client,
		opaClient:   opaClient,
	}
}

func (s *AccessRequestService) Create(
	ctx context.Context,
	resource, reason, managerEmail, actorEmail, actorRole string,
	requesterID int,
) (*ent.AccessRequest, error) {
	// Validate manager email exists in system
	manager, err := s.userRepo.FindByEmail(ctx, managerEmail)
	fmt.Printf("\n\n 10. Manager lookup result: %v, error: %v", manager, err)
	user, err := s.userRepo.FindByID(ctx, requesterID)
	if err != nil {
		return nil, err
	}

	opaInput := map[string]interface{}{
		"action": "create_request",
		"user": map[string]interface{}{
			"email":      user.Email,
			"role":       user.Role,
			"is_active":  user.IsActive,
			"department": user.Department,
		},
	}
	allowed, err := s.opaClient.Allow(ctx, opaInput)
	if err != nil {
		return nil, apperrors.Internal("manager lookup failed", err)
	}
	if !allowed {
		fmt.Printf("\n\n OPA policy denied access for user %s with role %s to create request", user.Email, user.Role)
		return nil, apperrors.Forbidden("access denied")
	}

	if manager == nil {
		return nil, apperrors.ValidationError(fmt.Sprintf("manager with email %s not found in system", managerEmail))
	}

	req, err := s.requestRepo.Create(ctx, resource, reason, managerEmail, requesterID)
	req.Edges.Requester = user
	fmt.Printf("\n\n 11. AccessRequest creation result: %v, error: %v", req, err)
	if err != nil {
		return nil, apperrors.Internal("failed to create request", err)
	}

	// Audit — fire and forget style in service layer
	// In production, you might want async audit via a channel
	if _, auditErr := s.auditRepo.Create(
		ctx,
		auditlog.ActionREQUEST_CREATED,
		actorEmail,
		actorRole,
		req.ID,
		nil,
	); auditErr != nil {
		// Log but don't fail the request — audit failure shouldn't block business operations
		fmt.Printf("WARN: audit log creation failed for request %d: %v\n", req.ID, auditErr)
	}

	return req, nil
}

func (s *AccessRequestService) UploadScreenshot(
	ctx context.Context,
	requestID int,
	fileName, fileBase64 string,
	actorEmail, actorRole string,
) (*ent.AccessRequest, error) {
	req, err := s.requestRepo.FindByID(ctx, requestID)
	if err != nil || req == nil {
		return nil, apperrors.NotFound("access request")
	}
	opaInput := map[string]interface{}{
		"action":   "upload_screenshot",
		"resource": req.Resource,
		"user": map[string]interface{}{
			"email":      actorEmail,
			"role":       actorRole,
			"is_active":  req.Edges.Requester.IsActive,
			"department": req.Edges.Requester.Department,
		},
		"data": map[string]interface{}{
			"request": map[string]interface{}{
				"requester_email": req.Edges.Requester.Email,
			},
		},
	}

	allowed, err := s.opaClient.Allow(ctx, opaInput)
	fmt.Printf("\n\n OPA authorization result for uploading screenshot: allowed=%v, error=%v", allowed, err)
	if err != nil {
		return nil, apperrors.Internal("OPA authorization failed", err)
	}

	if !allowed {
		return nil, apperrors.Forbidden("not authorized to upload screenshot")
	}

	// Only the requester can upload screenshot
	// if req.Edges.Requester.Email != actorEmail {
	// 	return nil, apperrors.Forbidden("only the requester can upload screenshots")
	// }

	url, err := s.s3Client.UploadBase64(ctx, fmt.Sprintf("%d", requestID), fileName, fileBase64)
	fmt.Printf("\n\n 12. S3 upload result: %s, error: %v", url, err)
	if err != nil {
		fmt.Printf("\n\n S3 upload failed for request %d: %v", requestID, err)
		return nil, apperrors.Internal("screenshot upload failed", err)
	}

	updated, err := s.requestRepo.SetScreenshotURL(ctx, requestID, url)
	if err != nil {
		return nil, apperrors.Internal("failed to save screenshot url", err)
	}

	meta := fmt.Sprintf(`{"file":"%s","url":"%s"}`, fileName, url)
	s.auditRepo.Create(ctx, auditlog.ActionSCREENSHOT_UPLOADED, actorEmail, actorRole, requestID, &meta) //nolint

	return updated, nil
}

// reviewAction is a shared internal helper for approve/reject/under-review.
// This avoids duplicating the access-check + status-update + audit logic three times.
//
// Flow:
// 1. Fetch request from DB
// 2. Check request is not already in terminal state (approved/rejected)
// 3. Check user role is allowed (SUPPORT/ENGINEERING/ADMIN)
// 4. NEW: Check OPA policy for action + resource + role combo
// 5. Update request status in DB
// 6. Create audit log entry
func (s *AccessRequestService) reviewAction(
	ctx context.Context,
	requestID int,
	status accessrequest.Status,
	auditAction auditlog.Action,
	comment *string,
	actorID int,
	actorEmail, actorRole string,
	opaAction string,
) (*ent.AccessRequest, error) {
	req, err := s.requestRepo.FindByID(ctx, requestID)
	if err != nil || req == nil {
		return nil, apperrors.NotFound("access request")
	}

	reviewer, err := s.userRepo.FindByID(ctx, actorID)
	if err != nil || reviewer == nil {
		return nil, apperrors.NotFound("reviewer")
	}

	if req.Status == accessrequest.StatusAPPROVED || req.Status == accessrequest.StatusREJECTED {
		return nil, apperrors.ValidationError(
			fmt.Sprintf("request is already %s and cannot be changed", req.Status),
		)
	}

	allowed, err := s.opaClient.Allow(ctx, map[string]any{
		"action":   opaAction,
		"resource": req.Resource,

		"user": map[string]any{
			"email":      reviewer.Email,
			"role":       reviewer.Role,
			"department": reviewer.Department,
			"is_active":  reviewer.IsActive,
		},

		"data": map[string]any{
			"request": map[string]any{
				"department":      req.Edges.Requester.Department,
				"requester_email": req.Edges.Requester.Email,
			},
		},
	})

	if err != nil {
		return nil, apperrors.Internal("authorization check failed", err)
	}

	if !allowed {
		return nil, apperrors.Forbidden(
			fmt.Sprintf("user %s with role %s cannot perform %s on resource %s",
				actorEmail, actorRole, opaAction, req.Resource),
		)
	}

	updated, err := s.requestRepo.UpdateStatus(ctx, requestID, status, actorID, comment)
	if err != nil {
		return nil, apperrors.Internal("failed to update request", err)
	}

	meta := fmt.Sprintf(`{"previous_status":"%s","new_status":"%s"}`, req.Status, status)
	s.auditRepo.Create(ctx, auditAction, actorEmail, actorRole, requestID, &meta)

	return updated, nil
}

func (s *AccessRequestService) Approve(
	ctx context.Context, requestID, actorID int,
	comment *string, actorEmail, actorRole string,
) (*ent.AccessRequest, error) {
	return s.reviewAction(
		ctx, requestID,
		accessrequest.StatusAPPROVED,
		auditlog.ActionREQUEST_APPROVED,
		comment, actorID, actorEmail, actorRole,
		"approve_request", // OPA action name
	)
}

func (s *AccessRequestService) Reject(
	ctx context.Context, requestID, actorID int,
	comment *string, actorEmail, actorRole string,
) (*ent.AccessRequest, error) {
	return s.reviewAction(
		ctx, requestID,
		accessrequest.StatusREJECTED,
		auditlog.ActionREQUEST_REJECTED,
		comment, actorID, actorEmail, actorRole,
		"reject_request", // OPA action name
	)
}

// MarkUnderReview marks a request under review with OPA authorization check
func (s *AccessRequestService) MarkUnderReview(
	ctx context.Context, requestID, actorID int,
	comment *string, actorEmail, actorRole string,
) (*ent.AccessRequest, error) {
	return s.reviewAction(
		ctx, requestID,
		accessrequest.StatusUNDER_REVIEW,
		auditlog.ActionREQUEST_UNDER_REVIEW,
		comment, actorID, actorEmail, actorRole,
		"mark_under_review", // OPA action name
	)
}

func (s *AccessRequestService) GetByID(ctx context.Context, id int) (*ent.AccessRequest, error) {
	req, err := s.requestRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to fetch request", err)
	}
	if req == nil {
		return nil, apperrors.NotFound("access request")
	}
	return req, nil
}

func (s *AccessRequestService) List(
	ctx context.Context,
	requesterEmail, resource *string,
	status *accessrequest.Status,
	page, pageSize int,
) ([]*ent.AccessRequest, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filter := repository.ListFilter{
		RequesterEmail: requesterEmail,
		Status:         status,
		Resource:       resource,
		Page:           page,
		PageSize:       pageSize,
	}

	return s.requestRepo.List(ctx, filter)
}

func (s *AccessRequestService) GetUser(ctx context.Context, id int) (*ent.User, error) {
	u, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, apperrors.Internal("failed to fetch user", err)
	}
	if u == nil {
		return nil, apperrors.NotFound("user")
	}
	return u, nil
}

func (s *AccessRequestService) GetAuditLogs(ctx context.Context, requestID int) ([]*ent.AuditLog, error) {
	logs, err := s.auditRepo.ListByRequestID(ctx, requestID)
	if err != nil {
		return nil, apperrors.Internal("failed to fetch audit logs", err)
	}
	return logs, nil
}

func (s *AccessRequestService) ListUsers(ctx context.Context) ([]*ent.User, error) {
	users, err := s.userRepo.ListAll(ctx)
	if err != nil {
		return nil, apperrors.Internal("failed to list users", err)
	}
	return users, nil
}

// UpdateUserRole updates a user's role with OPA authorization check (admin only)
func (s *AccessRequestService) UpdateUserRole(
	ctx context.Context,
	userID int,
	role string,
	actorEmail, actorRole string,
) (*ent.User, error) {

	allowed, err := s.opaClient.Allow(ctx, map[string]any{
		"action":   "update_user_role",
		"resource": "user-management",
		"user": map[string]any{
			"email": actorEmail,
			"role":  actorRole,
		},
	})

	if err != nil {
		return nil, apperrors.Internal("authorization check failed", err)
	}

	if !allowed {
		return nil, apperrors.Forbidden(
			fmt.Sprintf("user %s with role %s cannot update user roles",
				actorEmail, actorRole),
		)
	}

	u, err := s.userRepo.UpdateRole(ctx, userID, user.Role(role))
	if err != nil {
		return nil, apperrors.Internal("failed to update role", err)
	}

	return u, nil
}

// strconv import suppression — used elsewhere
var _ = strconv.Itoa
var _ = time.Now

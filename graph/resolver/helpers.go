package resolver

import (
	"context"
	"fmt"

	"cerberus/ent"
	"cerberus/graph/model"
	"cerberus/internal/auth"
	apperrors "cerberus/pkg/errors"
)

// requireAuth extracts claims from context or returns Unauthorized error
func requireAuth(ctx context.Context) (*auth.Claims, error) {
	claims := auth.ExtractClaims(ctx)
	if claims == nil {
		return nil, apperrors.Unauthorized("authentication required")
	}
	return claims, nil
}

func mapUser(u *ent.User) *model.User {
	if u == nil {
		return nil
	}
	return &model.User{
		ID:        fmt.Sprintf("%d", u.ID),
		Email:     u.Email,
		Name:      u.Name,
		Role:      model.Role(u.Role),
		IsActive:  u.IsActive,
		CreatedAt: u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

func mapAccessRequest(req *ent.AccessRequest) *model.AccessRequest {
	if req == nil {
		return nil
	}

	r := &model.AccessRequest{
		ID:           fmt.Sprintf("%d", req.ID),
		Resource:     req.Resource,
		Reason:       req.Reason,
		Status:       model.RequestStatus(req.Status),
		ManagerEmail: req.ManagerEmail,
		CreatedAt:    req.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    req.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if req.ScreenshotURL != nil {
		r.ScreenshotURL = req.ScreenshotURL
	}
	if req.ReviewComment != nil {
		r.ReviewComment = req.ReviewComment
	}
	if req.ResolvedAt != nil {
		t := req.ResolvedAt.Format("2006-01-02T15:04:05Z")
		r.ResolvedAt = &t
	}
	if req.Edges.Requester != nil {
		r.Requester = mapUser(req.Edges.Requester)
	}
	if req.Edges.Reviewer != nil {
		r.Reviewer = mapUser(req.Edges.Reviewer)
	}
	for _, l := range req.Edges.AuditLogs {
		r.AuditLogs = append(r.AuditLogs, mapAuditLog(l))
	}

	return r
}

func mapAuditLog(l *ent.AuditLog) *model.AuditLog {
	if l == nil {
		return nil
	}
	return &model.AuditLog{
		ID:         fmt.Sprintf("%d", l.ID),
		Action:     model.AuditAction(l.Action),
		ActorEmail: l.ActorEmail,
		ActorRole:  l.ActorRole,
		Metadata:   l.Metadata,
		CreatedAt:  l.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

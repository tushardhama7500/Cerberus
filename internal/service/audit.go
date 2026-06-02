package service

import (
	"context"

	"cerberus/ent"
	"cerberus/ent/auditlog"
	"cerberus/internal/repository"
	apperrors "cerberus/pkg/errors"
)

type AuditService struct {
	auditRepo *repository.AuditRepository
}

func NewAuditService(auditRepo *repository.AuditRepository) *AuditService {
	return &AuditService{auditRepo: auditRepo}
}

func (s *AuditService) GetByRequestID(ctx context.Context, requestID int) ([]*ent.AuditLog, error) {
	logs, err := s.auditRepo.ListByRequestID(ctx, requestID)
	if err != nil {
		return nil, apperrors.Internal("failed to fetch audit logs", err)
	}
	return logs, nil
}

func (s *AuditService) Create(
	ctx context.Context,
	action auditlog.Action,
	actorEmail, actorRole string,
	requestID int,
	metadata *string,
) (*ent.AuditLog, error) {
	log, err := s.auditRepo.Create(ctx, action, actorEmail, actorRole, requestID, metadata)
	if err != nil {
		return nil, apperrors.Internal("failed to create audit log", err)
	}
	return log, nil
}

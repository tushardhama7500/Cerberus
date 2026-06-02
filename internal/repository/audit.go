package repository

import (
	"context"
	"fmt"

	"cerberus/ent"
	"cerberus/ent/auditlog"
)

type AuditRepository struct {
	client *ent.Client
}

func NewAuditRepository(client *ent.Client) *AuditRepository {
	return &AuditRepository{client: client}
}

func (r *AuditRepository) Create(
	ctx context.Context,
	action auditlog.Action,
	actorEmail, actorRole string,
	requestID int,
	metadata *string,
) (*ent.AuditLog, error) {
	create := r.client.AuditLog.
		Create().
		SetAction(action).
		SetActorEmail(actorEmail).
		SetActorRole(actorRole).
		SetAccessRequestID(requestID)

	if metadata != nil {
		create = create.SetMetadata(*metadata)
	}

	log, err := create.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create audit log: %w", err)
	}
	return log, nil
}

func (r *AuditRepository) ListByRequestID(ctx context.Context, requestID int) ([]*ent.AuditLog, error) {
	logs, err := r.client.AuditLog.
		Query().
		Where(auditlog.AccessRequestIDEQ(requestID)).
		Order(ent.Asc(auditlog.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	return logs, nil
}

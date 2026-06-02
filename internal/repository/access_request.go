package repository

import (
	"context"
	"fmt"
	"time"

	"cerberus/ent"
	"cerberus/ent/accessrequest"
	"cerberus/ent/user"
)

type AccessRequestRepository struct {
	client *ent.Client
}

func NewAccessRequestRepository(client *ent.Client) *AccessRequestRepository {
	return &AccessRequestRepository{client: client}
}

type ListFilter struct {
	RequesterEmail *string
	Status         *accessrequest.Status
	Resource       *string
	Page           int
	PageSize       int
}

func (r *AccessRequestRepository) Create(
	ctx context.Context,
	resource, reason, managerEmail string,
	requesterID int,
) (*ent.AccessRequest, error) {
	req, err := r.client.AccessRequest.
		Create().
		SetResource(resource).
		SetReason(reason).
		SetManagerEmail(managerEmail).
		SetRequesterID(requesterID).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("create access request: %w", err)
	}
	return req, nil
}

func (r *AccessRequestRepository) FindByID(ctx context.Context, id int) (*ent.AccessRequest, error) {
	req, err := r.client.AccessRequest.
		Query().
		Where(accessrequest.IDEQ(id)).
		WithRequester().
		WithReviewer().
		WithAuditLogs().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find request by id: %w", err)
	}
	return req, nil
}

func (r *AccessRequestRepository) List(ctx context.Context, f ListFilter) ([]*ent.AccessRequest, int, error) {
	query := r.client.AccessRequest.Query().
		WithRequester().
		WithReviewer()

	if f.Status != nil {
		query = query.Where(accessrequest.StatusEQ(*f.Status))
	}
	if f.Resource != nil {
		query = query.Where(accessrequest.ResourceContainsFold(*f.Resource))
	}
	if f.RequesterEmail != nil {
		query = query.Where(
			accessrequest.HasRequesterWith(
				user.EmailContainsFold(*f.RequesterEmail),
			),
		)
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("count requests: %w", err)
	}

	offset := (f.Page - 1) * f.PageSize
	if offset < 0 {
		offset = 0
	}

	requests, err := query.
		Order(ent.Desc(accessrequest.FieldCreatedAt)).
		Offset(offset).
		Limit(f.PageSize).
		All(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("list requests: %w", err)
	}

	return requests, total, nil
}

func (r *AccessRequestRepository) UpdateStatus(
	ctx context.Context,
	id int,
	status accessrequest.Status,
	reviewerID int,
	comment *string,
) (*ent.AccessRequest, error) {
	update := r.client.AccessRequest.UpdateOneID(id).
		SetStatus(status).
		SetReviewerID(reviewerID)

	if comment != nil {
		update = update.SetReviewComment(*comment)
	}

	// Terminal states get a resolved timestamp
	if status == accessrequest.StatusAPPROVED || status == accessrequest.StatusREJECTED {
		update = update.SetResolvedAt(time.Now())
	}

	req, err := update.Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update request status: %w", err)
	}
	return req, nil
}

func (r *AccessRequestRepository) SetScreenshotURL(ctx context.Context, id int, url string) (*ent.AccessRequest, error) {
	req, err := r.client.AccessRequest.UpdateOneID(id).
		SetScreenshotURL(url).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("set screenshot url: %w", err)
	}
	return req, nil
}

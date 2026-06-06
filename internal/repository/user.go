package repository

import (
	"context"
	"fmt"

	"cerberus/ent"
	"cerberus/ent/user"
)

type UserRepository struct {
	client *ent.Client
}

func NewUserRepository(client *ent.Client) *UserRepository {
	fmt.Println("UserRepository initialized")
	return &UserRepository{client: client}
}

func (r *UserRepository) Create(ctx context.Context, email, name, passwordHash, department string) (*ent.User, error) {
	u, err := r.client.User.
		Create().
		SetEmail(email).
		SetName(name).
		SetPasswordHash(passwordHash).
		SetRole(user.RoleEMPLOYEE).                 // Default role — can be updated by admin
		SetDepartment(user.Department(department)). // Default department
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, fmt.Errorf("email already registered")
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*ent.User, error) {
	fmt.Println("11. UserRepository.FindByEmail called")

	u, err := r.client.User.
		Query().
		Where(user.EmailEQ(email)).
		Only(ctx)

	fmt.Printf("err = %#v\n", err)

	if err != nil {
		if ent.IsNotFound(err) {
			fmt.Printf("\n\n [5] There is a error inside err!=nil and it's a NotFound error. Returning nil, nil")
			return nil, nil
		}
		fmt.Printf("\n\n [6] Not inside the isnotfound error-->")
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	fmt.Printf("12. User found and email, password is: %v, %v\n",
		u.Email, u.PasswordHash)

	return u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id int) (*ent.User, error) {
	u, err := r.client.User.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return u, nil
}

func (r *UserRepository) ListAll(ctx context.Context) ([]*ent.User, error) {
	users, err := r.client.User.Query().
		Order(ent.Asc(user.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (r *UserRepository) UpdateRole(ctx context.Context, id int, role user.Role) (*ent.User, error) {
	u, err := r.client.User.UpdateOneID(id).
		SetRole(role).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("update user role: %w", err)
	}
	return u, nil
}

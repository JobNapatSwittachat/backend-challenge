package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"user-management-api/internal/core/domain"
	"user-management-api/internal/core/port"
)

const usersCollection = "users"

// userDocument is the persistence model. It is deliberately separate from
// domain.User so BSON concerns never leak into the core layer.
type userDocument struct {
	ID           bson.ObjectID `bson:"_id,omitempty"`
	Name         string        `bson:"name"`
	Email        string        `bson:"email"`
	PasswordHash string        `bson:"password_hash"`
	CreatedAt    time.Time     `bson:"created_at"`
}

func (d *userDocument) toDomain() *domain.User {
	return &domain.User{
		ID:           d.ID.Hex(),
		Name:         d.Name,
		Email:        d.Email,
		PasswordHash: d.PasswordHash,
		CreatedAt:    d.CreatedAt,
	}
}

// UserRepository implements port.UserRepository backed by MongoDB.
type UserRepository struct {
	collection *mongo.Collection
}

var _ port.UserRepository = (*UserRepository)(nil)

// NewUserRepository ensures the unique email index exists and returns the
// repository. Uniqueness is enforced by the database rather than by a
// read-then-write check, so it holds under concurrent registrations.
func NewUserRepository(ctx context.Context, db *mongo.Database) (*UserRepository, error) {
	collection := db.Collection(usersCollection)
	_, err := collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return nil, fmt.Errorf("creating unique email index: %w", err)
	}
	return &UserRepository{collection: collection}, nil
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	doc := userDocument{
		ID:           bson.NewObjectID(),
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    time.Now().UTC().Truncate(time.Millisecond),
	}
	if _, err := r.collection.InsertOne(ctx, doc); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, domain.ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("inserting user: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}
	var doc userDocument
	if err := r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("finding user by id: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var doc userDocument
	if err := r.collection.FindOne(ctx, bson.M{"email": email}).Decode(&doc); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("finding user by email: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *UserRepository) List(ctx context.Context) ([]domain.User, error) {
	cursor, err := r.collection.Find(ctx, bson.M{}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}
	defer cursor.Close(ctx)

	var docs []userDocument
	if err := cursor.All(ctx, &docs); err != nil {
		return nil, fmt.Errorf("decoding users: %w", err)
	}

	users := make([]domain.User, 0, len(docs))
	for i := range docs {
		users = append(users, *docs[i].toDomain())
	}
	return users, nil
}

func (r *UserRepository) Update(ctx context.Context, id string, update domain.UserUpdate) (*domain.User, error) {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return nil, domain.ErrUserNotFound
	}

	set := bson.M{}
	if update.Name != nil {
		set["name"] = *update.Name
	}
	if update.Email != nil {
		set["email"] = *update.Email
	}

	var doc userDocument
	err = r.collection.FindOneAndUpdate(
		ctx,
		bson.M{"_id": objectID},
		bson.M{"$set": set},
		options.FindOneAndUpdate().SetReturnDocument(options.After),
	).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, domain.ErrUserNotFound
		}
		if mongo.IsDuplicateKeyError(err) {
			return nil, domain.ErrEmailAlreadyExists
		}
		return nil, fmt.Errorf("updating user: %w", err)
	}
	return doc.toDomain(), nil
}

func (r *UserRepository) Delete(ctx context.Context, id string) error {
	objectID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return domain.ErrUserNotFound
	}
	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return fmt.Errorf("deleting user: %w", err)
	}
	if result.DeletedCount == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepository) Count(ctx context.Context) (int64, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return 0, fmt.Errorf("counting users: %w", err)
	}
	return count, nil
}

package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"user-management-api/internal/core/domain"
)

// JWTService implements port.TokenService using HMAC-SHA256 (HS256) signed
// JSON Web Tokens.
type JWTService struct {
	secret []byte
	issuer string
	ttl    time.Duration
	now    func() time.Time
}

func NewJWTService(secret string, issuer string, ttl time.Duration) *JWTService {
	return &JWTService{
		secret: []byte(secret),
		issuer: issuer,
		ttl:    ttl,
		now:    time.Now,
	}
}

// Generate issues an HS256 token whose subject is the user ID.
func (s *JWTService) Generate(userID string) (string, error) {
	now := s.now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		Issuer:    s.issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// Verify parses and validates the token and returns its subject (user ID).
func (s *JWTService) Verify(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(
		tokenString,
		&jwt.RegisteredClaims{},
		func(t *jwt.Token) (any, error) {
			// Restricting the algorithm prevents downgrade attacks such as
			// accepting an unsigned ("none") token.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method %q", t.Method.Alg())
			}
			return s.secret, nil
		},
		jwt.WithIssuer(s.issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(func() time.Time { return s.now() }),
	)
	if err != nil || !token.Valid {
		return "", domain.ErrInvalidToken
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || claims.Subject == "" {
		return "", domain.ErrInvalidToken
	}
	return claims.Subject, nil
}

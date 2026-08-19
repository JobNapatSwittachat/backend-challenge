package auth

import "testing"

func TestBcryptHashAndCompare(t *testing.T) {
	hasher := NewBcryptHasher()

	hash, err := hasher.Hash("supersecret")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "supersecret" {
		t.Fatal("hash must not equal the plaintext password")
	}
	if err := hasher.Compare(hash, "supersecret"); err != nil {
		t.Errorf("compare with correct password failed: %v", err)
	}
	if err := hasher.Compare(hash, "wrongpassword"); err == nil {
		t.Error("compare with wrong password should fail")
	}
}

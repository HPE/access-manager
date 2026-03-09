/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package metadata

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// this isn't a test as much as an example of how to decode a JWT
func TestParseJWT(t *testing.T) {
	// sample JWT token for testing purposes
	tokenString := "eyJhbGciOiJFUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJhY2Nlc3MtbWFuYWdlciIsInN1YiI6Iml" +
		"kIiwiYXVkIjpbImFjY2Vzcy1tYW5hZ2VyIl0sImV4cCI6MTc0OTE3MDg3NywibmJmIjoxNzQ5MTY4Nzc3LCJpYXQiOj" +
		"E3NDkxNjkwNzcsImtleSI6MTc0OTE4OTYwNSwiaWQiOiJoZHA6Ly91c2VyL3lveW9keW5lL2JvYiIsInJvbGVzIjpbI" +
		"mhkcDovL3JvbGUveW95b2R5bmUvZGFvcy9yZWYiLCJoZHA6Ly9yb2xlL3lveW9keW5lL2Rhb3MvcmF3Il19.Q9VZDG7" +
		"-ODAoHcXhVNQQYYncKmd2ef25muTg3flTw8FCUizssAz2EYG2QTx4TCLekV37ShYLr7v0Naw0m-L_Xg"
	claims := TokenClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		// return a bogus key for decoding purposes
		return []byte(""), nil
	})

	if token == nil {
		fmt.Println("token is nil, shouldn't happen")
		return
	}

	// token.Valid should be false because we provided a bogus key
	if token.Valid {
		fmt.Printf("%v", claims)
	} else {
		if err != nil {
			fmt.Printf("invalid token: %s\n  %v\n", err.Error(), claims)
		} else {
			fmt.Printf("invalid token: %v\n", claims)
		}
		fmt.Println(err)
	}
}

// TestKeyRotation verifies that key rotation happens correctly by accelerating
// the passage of time and watching the process in detail.
func TestKeyRotation(t *testing.T) {
	ms, err := OpenTestStore("bootstrap")
	assert.NoError(t, err)
	ctx := context.Background()

	marks := map[string]int{}

	// dt = 0..20min, k0 == new
	k0, _, err := ms.GetSigningKey(ctx, 20*time.Minute)
	m0 := mark(marks, s(k0))
	if err != nil {
		return
	}
	assert.Equal(t, 0, m0)

	ms.keyStore.keys.incrementTimeOffset(10 * time.Minute)

	// dt = 10..70min, k1 == k0
	k1, _, err := ms.GetSigningKey(ctx, 1*time.Hour)
	m1 := mark(marks, s(k1))
	if err != nil {
		return
	}
	assert.Equal(t, s(k0), s(k1))
	assert.Equal(t, m0, m1)

	ms.keyStore.keys.incrementTimeOffset(50 * time.Minute)

	// dt = 1Hr..4hr, k2 == k0
	k2, _, err := ms.GetSigningKey(ctx, 3*time.Hour)
	m2 := mark(marks, s(k2))
	if err != nil {
		return
	}
	assert.Equal(t, s(k0), s(k2))
	assert.Equal(t, m0, m2)

	ms.keyStore.keys.incrementTimeOffset(3 * time.Hour)

	// dt = 4..7hr, k3 == new
	k3, _, err := ms.GetSigningKey(ctx, 3*time.Hour)
	m3 := mark(marks, s(k3))
	if err != nil {
		return
	}
	assert.Equal(t, 1, m3)

	ms.keyStore.keys.incrementTimeOffset(3 * time.Hour)

	// dt = 7..13hr, k4 == new, k0 expires
	k4, _, err := ms.GetSigningKey(ctx, 6*time.Hour)
	m4 := mark(marks, s(k4))
	if err != nil {
		return
	}
	assert.Equal(t, 2, m4)

	rx, err := ms.keyStore.keys.getRange(ctx, "key-", "key-ffff", 100)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), rx.Count)

	ms.keyStore.keys.incrementTimeOffset(1 * time.Hour)

	// dt = 8..9hr, k5 == k3
	k5, _, err := ms.GetSigningKey(ctx, 1*time.Hour)
	m5 := mark(marks, s(k5))
	if err != nil {
		return
	}
	assert.Equal(t, m3, m5)
	rx, err = ms.keyStore.keys.getRange(ctx, "key-", "key-ffff", 100)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), rx.Count)

	ms.keyStore.keys.incrementTimeOffset(5 * time.Hour)

	// dt = 13..15hr, k6 == k4, k3, k5 expire
	k6, _, err := ms.GetSigningKey(ctx, 2*time.Hour)
	m6 := mark(marks, s(k6))
	assert.Equal(t, 3, m6)

	if err != nil {
		return
	}
	rx, err = ms.keyStore.keys.getRange(ctx, "key-", "key-ffff", 100)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), rx.Count)
}

func mark(m map[string]int, s string) int {
	r, ok := m[s]
	if ok {
		return r
	}
	m[s] = len(m)
	return m[s]
}

func s(k any) string {
	bytes, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: bytes,
	}))
}

// TestTokenValidation generates tokens, looks at their content and verifies their validity
func TestTokenValidation(t *testing.T) {
	var t1 *MetaStoreX
	var t2 *MetaStoreX
	var err error
	for {
		ms := time.Now().UnixMilli()
		if ms%1000 > 100 {
			time.Sleep(time.Duration(1000-ms%1000) * time.Millisecond)
		}
		// first, guarantee that we have keys with the same expiration for max confusion
		t1, err = OpenTestStore("new_sample")
		assert.NoError(t, err)

		t2, err = OpenTestStore("new_sample")
		assert.NoError(t, err)

		_, exp1, err := t2.GetSigningKey(context.Background(), 1*time.Hour)
		if err != nil {
			return
		}
		_, exp2, err := t2.GetSigningKey(context.Background(), 1*time.Hour)
		if err != nil {
			return
		}
		if exp1 == exp2 {
			break
		}
	}
	// this hack forces the jwt to accept our warped sense of test time
	jwt.TimeFunc = func() time.Time { return time.Now().Add(t1.keys.timeOffset()) }

	token1, err := t1.GetSignedJWTWithClaims(1*time.Hour, DefaultClaims("us", nil))
	assert.NoError(t, err)
	claims, err := t1.ValidateJWT(token1)
	assert.NoError(t, err)
	assert.Equal(t, "us", claims.Identity)

	token2, err := t2.GetSignedJWTWithClaims(1*time.Hour, DefaultClaims("us", nil))
	assert.NoError(t, err)
	claims, err = t2.ValidateJWT(token1)
	assert.ErrorContains(t, err, "crypto/ecdsa: verification error")

	token3, err := t1.GetSignedJWTWithClaims(30*time.Hour, DefaultClaims("later", nil))
	assert.NoError(t, err)
	claims, err = t1.ValidateJWT(token3)
	assert.NoError(t, err)
	assert.Equal(t, "later", claims.Identity)

	// crossing over should find a key, but it should fail crypto verification
	_, err = t1.ValidateJWT(token2)
	assert.ErrorContains(t, err, "crypto/ecdsa: verification error")

	t1.keys.incrementTimeOffset(58 * time.Minute)
	claims, err = t1.ValidateJWT(token1)
	assert.NoError(t, err)
	assert.Equal(t, "us", claims.Identity)

	t1.keys.incrementTimeOffset(65 * time.Minute)
	claims, err = t1.ValidateJWT(token1)
	assert.ErrorContains(t, err, "token is expired")

	t1.keys.incrementTimeOffset(6 * time.Hour)
	claims, err = t1.ValidateJWT(token1)
	assert.ErrorContains(t, err, "token is expired")

	claims, err = t1.ValidateJWT(token3)
	assert.NoError(t, err)
	assert.Equal(t, "later", claims.Identity)
}

/*
 * SPDX-FileCopyrightText:  Copyright Hewlett Packard Enterprise Development LP
 */

package metadata

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/hpe/access-manager/pkg/logger"
	"go.etcd.io/etcd/api/v3/mvccpb"
	v3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/protobuf/proto"
	"hash/fnv"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

type KeyStore interface {
	GetSignedJWTWithClaims(duration time.Duration, claims TokenClaims) (string, error)

	ValidateJWT(token string) (*TokenClaims, error)

	// GetSigningKey returns a credential signing key that has at least the required
	// life span remaining. Keys are shared across all access manager instances and
	// have a lifespan set on creation. Old keys are deleted when their life is over
	// and created when there is a need for a key that will outlive any current key.
	// Typically, the returned key will be an RSA key, but the type is not
	// guaranteed. It is guaranteed that the key type will correspond to the
	// advertised JWT signing method.
	GetSigningKey(ctx context.Context, requiredLife time.Duration) (any, int64, error)

	// GetJWTSigningMethod returns the signing method that corresponds to the keys
	// returned by this KeyStore instance. To use a KeyStore (ks) to sign a JWT with
	// a lifetime of at least one hour, you should use something like this:
	//
	// key, err = ks.GetSigningKey(ctx, time.Hour)
	// if err !=nil { ... }
	// token = jwt.NewWithClaims(ks.GetJWTSigningMethod(ctx)jwt.MapClaims{
	//"foo": "bar",
	//"nbf": time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
	//})
	// ... add more fields to the token ...
	// signedJWT = token.SignedString(key)
	GetJWTSigningMethod(ctx context.Context) jwt.SigningMethod

	// GetPublicSigningKeys returns a list of public keys that can be used to verify
	// JWTs signed by this KeyStore. The keys are returned in a format that is
	// compatible with the `jwt.Parse` function and any unexpired credential will be
	// verifiable with one of the keys returned by this function.
	GetPublicSigningKeys(ctx context.Context) (map[int64]string, error)
}

type TokenClaims struct {
	jwt.RegisteredClaims
	Key      int64    `json:"key"`
	Identity string   `json:"id"`
	Roles    []string `json:"roles"`
}

func DefaultClaims(id string, roles []string) TokenClaims {
	return TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "access-manager",
			Subject:   "id",
			Audience:  []string{"access-manager"},
			ExpiresAt: &jwt.NumericDate{time.Now().Add(60 * time.Minute)},
			NotBefore: &jwt.NumericDate{time.Now().Add(-5 * time.Minute)},
			IssuedAt:  &jwt.NumericDate{time.Now()},
		},
		Identity: id,
		Roles:    roles,
	}
}

// A keyStore centralizes the logic for key management. Persistence is handled by
// a `rawKeyStore` that can either send data to etcd or an in-memory table.
type keyStore struct {
	internalId string
	keys       rawKeyStore // this is where tests hook in
}

func (r *keyStore) GetPublicSigningKeys(ctx context.Context) (map[int64]string, error) {
	keys, err := r.keys.getRange(ctx, fmt.Sprintf("key-%016x", 0), "key-ffffffffffffffff", 100)
	if err != nil {
		return nil, err
	}
	publicKeys := make(map[int64]string, len(keys.Kvs))
	for _, kv := range keys.Kvs {
		exp := int64(0)
		n, err := fmt.Sscanf(string(kv.Key), "key-%x", &exp)
		if err != nil || n != 1 {
			return nil, errors.New("invalid name for key: " + string(kv.Key))
		}

		var signingKey SigningKey
		if err := proto.Unmarshal(kv.Value, &signingKey); err != nil {
			return nil, fmt.Errorf("error unmarshalling key: %s", err.Error())
		}
		key, err := x509.ParsePKCS8PrivateKey(signingKey.Key)
		if err != nil {
			return nil, fmt.Errorf("error parsing private signing key: %s", err.Error())
		}

		// Create PEM block for the public key
		bytes, err := x509.MarshalPKIXPublicKey(key.(*ecdsa.PrivateKey).Public())
		pemBlock := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: bytes,
		}

		// Encode to PEM format
		pemBytes := pem.EncodeToMemory(pemBlock)
		publicKeys[exp] = string(pemBytes)
	}
	return publicKeys, nil
}

func (r *keyStore) GetSignedJWTWithClaims(duration time.Duration, claims TokenClaims) (string, error) {
	ctx := context.Background()
	key, keyExpiration, err := r.GetSigningKey(ctx, duration)
	claims.Key = keyExpiration
	claims.ExpiresAt = &jwt.NumericDate{time.Now().Add(r.keys.timeOffset()).Add(duration)}
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(r.GetJWTSigningMethod(ctx), claims)
	rx, err := token.SignedString(key)
	if err != nil {
		return "", err
	}
	return rx, nil
}

var credentialPattern = regexp.MustCompile(`^[[:alpha:][:digit:]._-]+\s*$`)

func (r *keyStore) ValidateJWT(token string) (*TokenClaims, error) {
	if !credentialPattern.MatchString(token) {
		return nil, errors.New("invalid credential")
	}
	claims := TokenClaims{}
	_, err := jwt.ParseWithClaims(token, &claims, func(token *jwt.Token) (any, error) {
		specificKey, err := r.GetSpecificKey(context.Background(), claims.Key)
		if err != nil {
			return nil, err
		}
		private, err := x509.ParsePKCS8PrivateKey(specificKey)
		if err != nil {
			return nil, err
		}
		ecPrivateKey, ok := private.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("key is not an ECDSA private key")
		}
		return ecPrivateKey.Public(), nil
	})
	if err != nil {
		return nil, err
	}
	return &claims, nil
}

// `testKeyStore` provides key storage in a hash table for testing but acts (enough) like etcd
type testKeyStore struct {
	mu     sync.Mutex
	tbl    map[string]vType
	offset time.Duration
}

// GetSigningKey returns a new or cached key and the expiration time (unix seconds) of that key.
// The expiration time can be saved and used to recover the key directly using GetSpecificKey
func (r *keyStore) GetSigningKey(ctx context.Context, requiredLife time.Duration) (any, int64, error) {
	// scan for expired keys, we don't actually care if deletions work other than for logging
	end := fmt.Sprintf("key-%016x", (time.Now().Add(-1 * time.Minute).Add(r.keys.timeOffset())).Unix())
	response, err := r.keys.deleteRange(ctx, "key-", end)
	if err != nil {
		logger.GetLogger().Error().Msgf("error deleting expired keys: %s", err.Error())
	} else {
		if response.Deleted > 0 {
			logger.GetLogger().Info().Msgf("deleted %d expired keys", response.Deleted)
		}
	}

	// now look for at least one key that will outlive our needs
	requiredEnd := time.Now().Add(r.keys.timeOffset()).Add(requiredLife).Add(5 * time.Minute)
	keys, err := r.keys.getRange(ctx, fmt.Sprintf("key-%016x", requiredEnd.Unix()), "key-ffffffffffffffff", 1)
	if err != nil {
		return nil, 0, err
	}

	// if we found anything at all, the first one is the earliest to expire that will live long enough
	if keys.Count > 0 {
		var candidate SigningKey
		if err := proto.Unmarshal(keys.Kvs[0].Value, &candidate); err != nil {
			return nil, 0, fmt.Errorf("error unmarshalling key: %s", err.Error())
		}
		// this key is good enough
		key, err := x509.ParsePKCS8PrivateKey(candidate.Key)
		if err != nil {
			return nil, 0, fmt.Errorf("error parsing key: %s", err.Error())
		}
		var exp int64
		n, err := fmt.Sscanf(string(keys.Kvs[0].Key), "key-%x", &exp)
		if err != nil {
			return nil, 0, err
		}
		if n != 1 {
			return nil, 0, errors.New("invalid key")
		}
		return key, exp, nil
	}

	// no acceptable key found, let's make a new one

	// we want our new key to last at least a little while to minimize number of new keys
	var defaultKeyRotation = 6 * time.Hour
	expiration := time.Now().Add(r.keys.timeOffset()).Add(defaultKeyRotation)
	if expiration.Before(requiredEnd) {
		expiration = requiredEnd.Add(time.Hour)
	}

	newKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, 0, err
	}
	pubKey, err := x509.MarshalPKIXPublicKey(newKey.Public())
	if err != nil {
		return nil, 0, err
	}
	// Create PEM block
	pemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKey,
	}

	// Encode to PEM format
	pemBytes := pem.EncodeToMemory(pemBlock)

	logger.GetLogger().Info().Msgf("public key %d: << %s >>\n", expiration.Unix(), pemBytes)

	// We persist keys in PKCS#8 ASN.1 DER format with no need to encode them
	marshaledKey, err := x509.MarshalPKCS8PrivateKey(newKey)
	if err != nil {
		return nil, 0, err
	}
	holder := SigningKey{
		Created: time.Now().Unix(),
		Key:     marshaledKey,
	}
	h := fnv.New64()
	_, _ = h.Write(holder.Key)

	keyBytes, err := proto.Marshal(&holder)
	if err != nil {
		return nil, 0, err
	}
	exp := expiration.Unix()
	_, err = r.keys.put(ctx, fmt.Sprintf("key-%016x", exp), string(keyBytes))
	if err != nil {
		return nil, 0, err
	}
	return newKey, exp, nil
}

// GetSpecificKey returns a particular signing key (assuming it has not expired)
func (r *keyStore) GetSpecificKey(ctx context.Context, expiration int64) ([]byte, error) {
	if expiration < time.Now().Unix() {
		return nil, errors.New("key has already expired")
	}
	k0 := fmt.Sprintf("key-%016x", expiration)
	k1 := fmt.Sprintf("key-%016x", expiration+1)
	got, err := r.keys.getRange(ctx, k0, k1, 1)
	if err != nil {
		return nil, err
	}
	if got.Count > 0 {
		var key SigningKey
		if err := proto.Unmarshal(got.Kvs[0].Value, &key); err != nil {
			return nil, err
		}
		h := fnv.New64()
		_, _ = h.Write(key.Key)
		return key.Key, nil
	}
	return nil, errors.New("key not found")
}

func (r *keyStore) GetJWTSigningMethod(_ context.Context) jwt.SigningMethod {
	return jwt.SigningMethodES256
}

// `rawKeyStore` provides absolute minimal persistence semantics for the job of maintaining keys
type rawKeyStore interface {
	deleteRange(ctx context.Context, startKey, endKey string) (*v3.DeleteResponse, error)
	getRange(ctx context.Context, startKey, endKey string, limit int64) (*v3.GetResponse, error)
	put(ctx context.Context, key string, value string) (*v3.PutResponse, error)
	timeOffset() time.Duration
	incrementTimeOffset(duration time.Duration)
}

// `etcKeyStore` provides key storage in an etcd name space
type etcKeyStore struct {
	meta v3.KV
}

func (e *etcKeyStore) timeOffset() time.Duration {
	return 0
}

func (e *etcKeyStore) incrementTimeOffset(_ time.Duration) {
	panic("time offset only for testing")
}

func (e *etcKeyStore) deleteRange(ctx context.Context, startKey, endKey string) (*v3.DeleteResponse, error) {
	return e.meta.Delete(ctx, startKey, v3.WithPrefix(), v3.WithRange(endKey))
}

func (e *etcKeyStore) getRange(ctx context.Context, startKey, endKey string, limit int64) (*v3.GetResponse, error) {
	return e.meta.Get(ctx, startKey, v3.WithFromKey(), v3.WithRange(endKey), v3.WithLimit(limit))
}

func (e *etcKeyStore) put(ctx context.Context, key string, value string) (*v3.PutResponse, error) {
	return e.meta.Put(ctx, key, value)
}

func (t *testKeyStore) timeOffset() time.Duration {
	return t.offset
}

func (t *testKeyStore) incrementTimeOffset(duration time.Duration) {
	if duration < 0 {
		panic("time slippage should only move forward")
	}
	t.offset += duration
}

func (t *testKeyStore) deleteRange(_ context.Context, startKey, endKey string) (*v3.DeleteResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	deleteCount := int64(0)
	for k := range t.tbl {
		if k >= startKey && k <= endKey {
			delete(t.tbl, k)
			deleteCount++
		}
	}
	return &v3.DeleteResponse{
		Deleted: deleteCount,
	}, nil
}

func (t *testKeyStore) getRange(_ context.Context, startKey, endKey string, limit int64) (*v3.GetResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	type kv struct {
		k string
		v vType
	}
	r0 := []kv{}
	for k, v := range t.tbl {
		if k >= startKey && k <= endKey {
			r0 = append(r0, kv{k: k, v: v})
		}
	}
	slices.SortFunc(r0, func(a, b kv) int {
		return strings.Compare(a.k, b.k)
	})
	r := []*mvccpb.KeyValue{}
	if int64(len(r0)) < limit {
		limit = int64(len(r0))
	}
	for _, kv := range r0[:limit] {
		r = append(r, &mvccpb.KeyValue{
			Key:   []byte(kv.k),
			Value: kv.v.value,
		})
	}
	return &v3.GetResponse{
		Header: nil,
		Kvs:    r,
		More:   false,
		Count:  int64(len(r)),
	}, nil
}

func (t *testKeyStore) put(_ context.Context, key string, value string) (*v3.PutResponse, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tbl[key] = vType{
		version: 0,
		value:   []byte(value),
	}
	return &v3.PutResponse{}, nil
}

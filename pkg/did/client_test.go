/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package did

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/stretchr/testify/require"

	"github.com/trustbloc/trustbloc-did-method/pkg/did/doc"
	"github.com/trustbloc/trustbloc-did-method/pkg/did/option/create"
	"github.com/trustbloc/trustbloc-did-method/pkg/did/option/deactivate"
	"github.com/trustbloc/trustbloc-did-method/pkg/did/option/recovery"
	"github.com/trustbloc/trustbloc-did-method/pkg/did/option/update"
	mockconfig "github.com/trustbloc/trustbloc-did-method/pkg/internal/mock/config"
	mockdiscovery "github.com/trustbloc/trustbloc-did-method/pkg/internal/mock/discovery"
	mockendpoint "github.com/trustbloc/trustbloc-did-method/pkg/internal/mock/endpoint"
	mockselection "github.com/trustbloc/trustbloc-did-method/pkg/internal/mock/selection"
	"github.com/trustbloc/trustbloc-did-method/pkg/vdri/trustbloc/endpoint"
	"github.com/trustbloc/trustbloc-did-method/pkg/vdri/trustbloc/models"
)

func TestClient_DeactivateDID(t *testing.T) {
	t.Run("test domain is empty", func(t *testing.T) {
		v := New()

		_, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.DeactivateDID("did:ex:123", "", deactivate.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "domain is empty")
	})

	t.Run("test signing key empty", func(t *testing.T) {
		v := New()

		err := v.DeactivateDID("did:ex:123", "testnet")
		require.Error(t, err)
		require.Contains(t, err.Error(), "signing key is required")
	})

	t.Run("test error from get endpoints", func(t *testing.T) {
		v := New()

		_, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		v.endpointService = endpoint.NewService(
			discoveryMock([]*models.Endpoint{}, fmt.Errorf("discover error")),
			selectionMock([]*models.Endpoint{}, nil))

		err = v.DeactivateDID("did:ex:123", "testnet", deactivate.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "discover error")
	})

	t.Run("test unsupported signing key", func(t *testing.T) {
		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		err := v.DeactivateDID("did:ex:123", "testnet", deactivate.WithSigningKey("www"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "key not supported")
	})

	t.Run("test error from unique suffix", func(t *testing.T) {
		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		_, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.DeactivateDID("wrong", "testnet", deactivate.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "unique suffix not provided in id")
	})

	t.Run("test error from send request", func(t *testing.T) {
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer serv.Close()

		v := New(WithAuthToken("tk1"))

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		_, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.DeactivateDID("did:ex:123", "testnet", deactivate.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to send deactivate sidetree request")
	})

	t.Run("test success", func(t *testing.T) {
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer serv.Close()

		v := New(WithAuthToken("tk1"))

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		_, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.DeactivateDID("did:ex:123", "", deactivate.WithSigningKey(privKey),
			deactivate.WithSidetreeEndpoint(serv.URL), deactivate.WithSigningKeyID("k1"))
		require.NoError(t, err)
	})
}

func TestClient_RecoverDID(t *testing.T) {
	t.Run("test domain is empty", func(t *testing.T) {
		v := New()

		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.RecoverDID("did:ex:123", "", recovery.WithNextUpdatePublicKey(pubKey),
			recovery.WithNextRecoveryPublicKey(pubKey), recovery.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "domain is empty")
	})

	t.Run("test failed to get sidetree config", func(t *testing.T) {
		v := New()

		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return nil, fmt.Errorf("failed to get sidetree config")
			}}

		err = v.RecoverDID("did:ex:123", "testnet", recovery.WithNextUpdatePublicKey(pubKey),
			recovery.WithNextRecoveryPublicKey(pubKey), recovery.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get sidetree config")
	})

	t.Run("test next recovery key empty", func(t *testing.T) {
		v := New()

		err := v.RecoverDID("did:ex:123", "testnet")
		require.Error(t, err)
		require.Contains(t, err.Error(), "next recovery public key is required")
	})

	t.Run("test next update key empty", func(t *testing.T) {
		v := New()

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.RecoverDID("did:ex:123", "testnet", recovery.WithNextRecoveryPublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "next update public key is required")
	})

	t.Run("test signing key empty", func(t *testing.T) {
		v := New()

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.RecoverDID("did:ex:123", "testnet", recovery.WithNextRecoveryPublicKey(pubKey),
			recovery.WithNextUpdatePublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "signing key is required")
	})

	t.Run("test error from get endpoints", func(t *testing.T) {
		v := New()

		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		v.endpointService = endpoint.NewService(
			discoveryMock([]*models.Endpoint{}, fmt.Errorf("discover error")),
			selectionMock([]*models.Endpoint{}, nil))

		err = v.RecoverDID("did:ex:123", "testnet", recovery.WithNextUpdatePublicKey(pubKey),
			recovery.WithNextRecoveryPublicKey(pubKey), recovery.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "discover error")
	})

	t.Run("test failed to get next recovery key", func(t *testing.T) {
		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return &models.SidetreeConfig{MultiHashAlgorithm: 18}, nil
			}}

		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.RecoverDID("did:ex:123", "testnet", recovery.WithSigningKey(privKey),
			recovery.WithNextRecoveryPublicKey([]byte("wrong")), recovery.WithNextUpdatePublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get next recovery key")
	})

	t.Run("test failed to get next update key", func(t *testing.T) {
		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return &models.SidetreeConfig{MultiHashAlgorithm: 18}, nil
			}}

		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.RecoverDID("did:ex:123", "testnet", recovery.WithSigningKey(privKey),
			recovery.WithNextUpdatePublicKey([]byte("wrong")), recovery.WithNextRecoveryPublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get next update key")
	})

	t.Run("test unsupported signing key", func(t *testing.T) {
		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return &models.SidetreeConfig{MultiHashAlgorithm: 18}, nil
			}}

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.RecoverDID("did:ex:123", "testnet", recovery.WithSigningKey("www"),
			recovery.WithNextUpdatePublicKey(pubKey), recovery.WithNextRecoveryPublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "key not supported")
	})

	t.Run("test error from unique suffix", func(t *testing.T) {
		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return &models.SidetreeConfig{MultiHashAlgorithm: 18}, nil
			}}

		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.RecoverDID("wrong", "testnet", recovery.WithSigningKey(privKey),
			recovery.WithNextUpdatePublicKey(pubKey), recovery.WithNextRecoveryPublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "unique suffix not provided in id")
	})

	t.Run("test error parse public key", func(t *testing.T) {
		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return &models.SidetreeConfig{MultiHashAlgorithm: 18}, nil
			}}

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		ecPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		err = v.RecoverDID("did:ex:123", "testnet", recovery.WithSigningKey(ecPrivKey),
			recovery.WithSigningKeyID("k1"), recovery.WithNextRecoveryPublicKey(pubKey),
			recovery.WithNextUpdatePublicKey(pubKey), recovery.WithPublicKey(&doc.PublicKey{ID: "key3",
				Encoding: doc.PublicKeyEncodingJwk, KeyType: doc.Ed25519KeyType, Value: []byte(`{
          "kty":"EC",
          "crv":"P-256",
          "x":"MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
          "y":"4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
          "use":"enc",
          "kid":"1"}`)}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported PublicKey source key type")
	})

	t.Run("test error from send request", func(t *testing.T) {
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer serv.Close()

		v := New(WithAuthToken("tk1"))

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		ecPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		err = v.RecoverDID("did:ex:123", "",
			recovery.WithSidetreeEndpoint(serv.URL), recovery.WithSigningKey(ecPrivKey),
			recovery.WithSigningKeyID("k1"), recovery.WithNextRecoveryPublicKey(pubKey),
			recovery.WithNextUpdatePublicKey(pubKey), recovery.WithPublicKey(&doc.PublicKey{ID: "key3",
				Encoding: doc.PublicKeyEncodingJwk, KeyType: doc.Ed25519KeyType, Value: pubKey}),
			recovery.WithService(&did.Service{ID: "svc3"}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to send recover sidetree request")
	})

	t.Run("test success", func(t *testing.T) {
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes, err := (&did.Doc{ID: "did1", Context: []string{did.Context}}).JSONBytes()
			require.NoError(t, err)
			b, err := json.Marshal(didResolution{Context: "https://www.w3.org/ns/did-resolution/v1",
				DIDDocument: bytes})
			require.NoError(t, err)
			_, err = fmt.Fprint(w, string(b))
			require.NoError(t, err)
		}))
		defer serv.Close()

		v := New(WithAuthToken("tk1"))

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		ecPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		err = v.RecoverDID("did:ex:123", "",
			recovery.WithSidetreeEndpoint(serv.URL), recovery.WithSigningKey(ecPrivKey),
			recovery.WithSigningKeyID("k1"), recovery.WithNextRecoveryPublicKey(pubKey),
			recovery.WithNextUpdatePublicKey(pubKey), recovery.WithPublicKey(&doc.PublicKey{ID: "key3",
				Encoding: doc.PublicKeyEncodingJwk, KeyType: doc.Ed25519KeyType, Value: pubKey}),
			recovery.WithService(&did.Service{ID: "svc3"}))
		require.NoError(t, err)
	})
}

func TestClient_UpdateDID(t *testing.T) {
	t.Run("test domain is empty", func(t *testing.T) {
		v := New()

		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.UpdateDID("did:ex:123", "", update.WithNextUpdatePublicKey(pubKey),
			update.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "domain is empty")
	})

	t.Run("test failed to get sidetree config", func(t *testing.T) {
		v := New()

		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return nil, fmt.Errorf("failed to get sidetree config")
			}}

		err = v.UpdateDID("did:ex:123", "testnet", update.WithNextUpdatePublicKey(pubKey),
			update.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get sidetree config")
	})

	t.Run("test signing key empty", func(t *testing.T) {
		v := New()

		err := v.UpdateDID("did:ex:123", "testnet")
		require.Error(t, err)
		require.Contains(t, err.Error(), "signing public key is required")
	})

	t.Run("test next updates key empty", func(t *testing.T) {
		v := New()

		_, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.UpdateDID("did:ex:123", "testnet", update.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "next update public key is required")
	})

	t.Run("test error from get endpoints", func(t *testing.T) {
		v := New()

		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		v.endpointService = endpoint.NewService(
			discoveryMock([]*models.Endpoint{}, fmt.Errorf("discover error")),
			selectionMock([]*models.Endpoint{}, nil))

		err = v.UpdateDID("did:ex:123", "testnet", update.WithNextUpdatePublicKey(pubKey),
			update.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "discover error")

		v.endpointService = endpoint.NewService(
			discoveryMock(nil, nil),
			selectionMock(nil, fmt.Errorf("select error")))

		err = v.UpdateDID("did:ex:123", "testnet", update.WithNextUpdatePublicKey(pubKey),
			update.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "select error")

		v.endpointService = endpoint.NewService(
			discoveryMock(nil, nil),
			selectionMock(nil, nil))

		err = v.UpdateDID("did:ex:123", "testnet", update.WithNextUpdatePublicKey(pubKey),
			update.WithSigningKey(privKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "list of endpoints is empty")
	})

	t.Run("test failed to get next update key", func(t *testing.T) {
		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return &models.SidetreeConfig{MultiHashAlgorithm: 18}, nil
			}}

		_, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.UpdateDID("did:ex:123", "testnet", update.WithSigningKey(privKey),
			update.WithNextUpdatePublicKey([]byte("wrong")))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get next update key")
	})

	t.Run("test unsupported signing key", func(t *testing.T) {
		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return &models.SidetreeConfig{MultiHashAlgorithm: 18}, nil
			}}

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.UpdateDID("did:ex:123", "testnet", update.WithSigningKey("www"),
			update.WithNextUpdatePublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "key not supported")
	})

	t.Run("test error from unique suffix", func(t *testing.T) {
		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return &models.SidetreeConfig{MultiHashAlgorithm: 18}, nil
			}}

		pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		err = v.UpdateDID("wrong", "testnet", update.WithSigningKey(privKey),
			update.WithNextUpdatePublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "unique suffix not provided in id")
	})

	t.Run("test success", func(t *testing.T) {
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer serv.Close()

		v := New(WithAuthToken("tk1"))

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return &models.SidetreeConfig{MultiHashAlgorithm: 18}, nil
			}}

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		ecPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		err = v.UpdateDID("did:ex:123", "",
			update.WithSidetreeEndpoint(serv.URL), update.WithSigningKey(ecPrivKey),
			update.WithNextUpdatePublicKey(pubKey), update.WithRemoveService("svc1"),
			update.WithRemoveService("svc1"), update.WithRemovePublicKey("k1"),
			update.WithRemovePublicKey("k2"), update.WithAddPublicKey(&doc.PublicKey{ID: "key3",
				Encoding: doc.PublicKeyEncodingJwk, KeyType: doc.Ed25519KeyType, Value: pubKey}),
			update.WithAddService(&did.Service{ID: "svc3"}))
		require.NoError(t, err)
	})
}

func TestClient_CreateDID(t *testing.T) {
	t.Run("test domain is empty", func(t *testing.T) {
		v := New()

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		createDID, err := v.CreateDID("", create.WithUpdatePublicKey(pubKey), create.WithRecoveryPublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "domain is empty")
		require.Nil(t, createDID)
	})

	t.Run("test failed to get sidetree config", func(t *testing.T) {
		v := New()

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return nil, fmt.Errorf("failed to get sidetree config")
			}}

		createDID, err := v.CreateDID("testnet", create.WithUpdatePublicKey(pubKey),
			create.WithRecoveryPublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get sidetree config")
		require.Nil(t, createDID)
	})

	t.Run("test error from get endpoints", func(t *testing.T) {
		v := New()

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		v.endpointService = endpoint.NewService(
			discoveryMock(nil, fmt.Errorf("discover error")),
			selectionMock(nil, nil))

		createDID, err := v.CreateDID("testnet", create.WithUpdatePublicKey(pubKey), create.WithRecoveryPublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "discover error")
		require.Nil(t, createDID)

		v.endpointService = endpoint.NewService(
			discoveryMock(nil, nil),
			selectionMock(nil, fmt.Errorf("select error")))

		createDID, err = v.CreateDID("testnet", create.WithUpdatePublicKey(pubKey),
			create.WithRecoveryPublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "select error")
		require.Nil(t, createDID)

		v.endpointService = endpoint.NewService(
			discoveryMock(nil, nil),
			selectionMock(nil, nil))

		createDID, err = v.CreateDID("testnet", create.WithUpdatePublicKey(pubKey),
			create.WithRecoveryPublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "list of endpoints is empty")
		require.Nil(t, createDID)
	})

	t.Run("test error from send create sidetree request", func(t *testing.T) {
		v := New()

		ed25519RecoveryPubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		ed25519UpdatePubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		// failed to create http request
		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "http://[]%20%/"}}, nil
			}}

		v.configService = &mockconfig.MockConfigService{
			GetSidetreeConfigFunc: func(s string) (*models.SidetreeConfig, error) {
				return &models.SidetreeConfig{MultiHashAlgorithm: 18}, nil
			}}

		createDID, err := v.CreateDID("testnet", create.WithRecoveryPublicKey(ed25519RecoveryPubKey),
			create.WithUpdatePublicKey(ed25519UpdatePubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create http request")
		require.Nil(t, createDID)

		// test failed to send request
		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: "url"}}, nil
			}}

		createDID, err = v.CreateDID("testnet", create.WithRecoveryPublicKey(ed25519RecoveryPubKey),
			create.WithUpdatePublicKey(ed25519UpdatePubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to send request")
		require.Nil(t, createDID)

		// test http status not equal 200
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer serv.Close()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		createDID, err = v.CreateDID("testnet", create.WithRecoveryPublicKey(ed25519RecoveryPubKey),
			create.WithUpdatePublicKey(ed25519UpdatePubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "got unexpected response")
		require.Nil(t, createDID)

		// test failed to parse did
		serv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes, err1 := (&did.Doc{ID: "did1"}).JSONBytes()
			require.NoError(t, err1)
			_, err1 = fmt.Fprint(w, string(bytes))
			require.NoError(t, err1)
		}))
		defer serv.Close()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		createDID, err = v.CreateDID("testnet", create.WithRecoveryPublicKey(ed25519RecoveryPubKey),
			create.WithUpdatePublicKey(ed25519UpdatePubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to parse public DID document")
		require.Nil(t, createDID)
	})

	t.Run("test success", func(t *testing.T) {
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes, err := (&did.Doc{ID: "did1", Context: []string{did.Context}}).JSONBytes()
			require.NoError(t, err)
			b, err := json.Marshal(didResolution{Context: "https://www.w3.org/ns/did-resolution/v1",
				DIDDocument: bytes})
			require.NoError(t, err)
			_, err = fmt.Fprint(w, string(b))
			require.NoError(t, err)
		}))
		defer serv.Close()

		ed25519RecoveryPubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		ecUpdatePrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		ecPrivKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		ecPubKeyBytes := elliptic.Marshal(ecPrivKey.PublicKey.Curve, ecPrivKey.PublicKey.X, ecPrivKey.PublicKey.Y)

		v := New(WithTLSConfig(nil))

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		createDID, err := v.CreateDID("testnet", create.WithRecoveryPublicKey(ed25519RecoveryPubKey),
			create.WithUpdatePublicKey(ecUpdatePrivKey.Public()),
			create.WithPublicKey(&doc.PublicKey{ID: "key1",
				Type: doc.JWSVerificationKey2020, Encoding: doc.PublicKeyEncodingJwk, KeyType: doc.Ed25519KeyType,
				Value:    ed25519RecoveryPubKey,
				Purposes: []string{doc.KeyPurposeAuthentication}}),
			create.WithPublicKey(&doc.PublicKey{ID: "key2",
				Type:     doc.JWSVerificationKey2020,
				Encoding: doc.PublicKeyEncodingJwk,
				Value:    ecPubKeyBytes,
				KeyType:  doc.P256KeyType,
				Purposes: []string{doc.KeyPurposeAuthentication},
			}),
			create.WithService(&did.Service{ID: "srv1", Type: "type", ServiceEndpoint: "http://example.com",
				Properties: map[string]interface{}{"priority": "1"}}))
		require.NoError(t, err)
		require.Equal(t, "did1", createDID.ID)
	})

	t.Run("test create DID - invalid key type", func(t *testing.T) {
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes, err := (&did.Doc{ID: "did1", Context: []string{did.Context}}).JSONBytes()
			require.NoError(t, err)
			_, err = fmt.Fprint(w, string(bytes))
			require.NoError(t, err)
		}))
		defer serv.Close()

		ed25519RecoveryPubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		ed25519UpdatePubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		createDID, err := v.CreateDID("testnet", create.WithRecoveryPublicKey(ed25519RecoveryPubKey),
			create.WithUpdatePublicKey(ed25519UpdatePubKey), create.WithPublicKey(&doc.PublicKey{ID: "#key1",
				Type:     doc.JWSVerificationKey2020,
				Encoding: doc.PublicKeyEncodingJwk,
				KeyType:  "InvalidKeyType",
			}),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid key type: InvalidKeyType")
		require.Nil(t, createDID)
	})

	t.Run("test create DID - EC key error", func(t *testing.T) {
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes, err := (&did.Doc{ID: "did1", Context: []string{did.Context}}).JSONBytes()
			require.NoError(t, err)
			_, err = fmt.Fprint(w, string(bytes))
			require.NoError(t, err)
		}))
		defer serv.Close()

		ed25519RecoveryPubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		ed25519UpdatePubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		ed25519PubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		createDID, err := v.CreateDID("testnet", create.WithRecoveryPublicKey(ed25519RecoveryPubKey),
			create.WithUpdatePublicKey(ed25519UpdatePubKey),
			create.WithPublicKey(&doc.PublicKey{ID: "#key1",
				Type:     doc.JWSVerificationKey2020,
				Encoding: doc.PublicKeyEncodingJwk,
				KeyType:  doc.P256KeyType,
				Value:    ed25519PubKey,
			}),
		)
		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid EC key")
		require.Nil(t, createDID)
	})

	t.Run("test unsupported recovery public key type", func(t *testing.T) {
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes, err := (&did.Doc{ID: "did1", Context: []string{did.Context}}).JSONBytes()
			require.NoError(t, err)
			_, err = fmt.Fprint(w, string(bytes))
			require.NoError(t, err)
		}))
		defer serv.Close()

		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		createDID, err := v.CreateDID("testnet", create.WithRecoveryPublicKey("wrongkey"),
			create.WithUpdatePublicKey("wrongvalue"))
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get recovery key")
		require.Nil(t, createDID)
	})

	t.Run("test recovery public key empty", func(t *testing.T) {
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes, err := (&did.Doc{ID: "did1", Context: []string{did.Context}}).JSONBytes()
			require.NoError(t, err)
			_, err = fmt.Fprint(w, string(bytes))
			require.NoError(t, err)
		}))
		defer serv.Close()

		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		createDID, err := v.CreateDID("testnet")
		require.Error(t, err)
		require.Contains(t, err.Error(), "recovery public key is required")
		require.Nil(t, createDID)
	})

	t.Run("test update public key empty", func(t *testing.T) {
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes, err := (&did.Doc{ID: "did1", Context: []string{did.Context}}).JSONBytes()
			require.NoError(t, err)
			_, err = fmt.Fprint(w, string(bytes))
			require.NoError(t, err)
		}))
		defer serv.Close()

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		createDID, err := v.CreateDID("testnet", create.WithRecoveryPublicKey(pubKey))
		require.Error(t, err)
		require.Contains(t, err.Error(), "update public key is required")
		require.Nil(t, createDID)
	})

	t.Run("test unsupported public key encoding", func(t *testing.T) {
		serv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			bytes, err := (&did.Doc{ID: "did1", Context: []string{did.Context}}).JSONBytes()
			require.NoError(t, err)
			_, err = fmt.Fprint(w, string(bytes))
			require.NoError(t, err)
		}))
		defer serv.Close()

		pubKey, _, err := ed25519.GenerateKey(rand.Reader)
		require.NoError(t, err)

		v := New()

		v.endpointService = &mockendpoint.MockEndpointService{
			GetEndpointsFunc: func(domain string) (endpoints []*models.Endpoint, err error) {
				return []*models.Endpoint{{URL: serv.URL}}, nil
			}}

		createDID, err := v.CreateDID("testnet", create.WithRecoveryPublicKey(pubKey),
			create.WithUpdatePublicKey(pubKey), create.WithPublicKey(&doc.PublicKey{ID: "#key2",
				Type: doc.JWSVerificationKey2020, Encoding: "wrong", Value: []byte("wrongValue")}))
		require.Error(t, err)
		require.Contains(t, err.Error(), "public key encoding not supported")
		require.Nil(t, createDID)
	})
}

func Test_unwrapPubKeyJWK(t *testing.T) {
	t.Run("no wrapping", func(t *testing.T) {
		key := doc.PublicKey{Value: []byte("abcd")}
		key2, err := unwrapPubKeyJWK(key)
		require.NoError(t, err)
		require.Contains(t, string(key2.Value), "abcd")
	})

	t.Run("unwrap wrapped jwk", func(t *testing.T) {
		key := doc.PublicKey{Value: []byte(`{
  "kty": "OKP",
  "kid": "key1",
  "crv": "Ed25519",
  "x": "test value"
}`)}
		key2, err := unwrapPubKeyJWK(key)
		require.NoError(t, err)
		require.Contains(t, string(key2.Value), "test value")
	})

	t.Run("error unsupported type", func(t *testing.T) {
		key := doc.PublicKey{Value: []byte(`{
          "kty":"EC",
          "crv":"P-256",
          "x":"MKBCTNIcKUSDii11ySs3526iDZ8AiTo7Tu6KPAqv7D4",
          "y":"4Etl6SRW2YiLUrN5vfvVHuhp7x8PxltmWWlbbM4IFyM",
          "use":"enc",
          "kid":"1"}`)}
		_, err := unwrapPubKeyJWK(key)
		require.Error(t, err)
		require.Contains(t, err.Error(), "unsupported PublicKey source key type")
	})
}

func discoveryMock(endpoints []*models.Endpoint, err error) *mockdiscovery.MockDiscoveryService {
	return &mockdiscovery.MockDiscoveryService{
		GetEndpointsFunc: func(string) ([]*models.Endpoint, error) {
			return endpoints, err
		},
	}
}

func selectionMock(endpoints []*models.Endpoint, err error) *mockselection.MockSelectionService {
	return &mockselection.MockSelectionService{
		SelectEndpointsFunc: func(string, []*models.Endpoint) ([]*models.Endpoint, error) {
			return endpoints, err
		},
	}
}

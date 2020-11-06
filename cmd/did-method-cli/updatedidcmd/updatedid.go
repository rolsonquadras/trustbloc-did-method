/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package updatedidcmd

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"

	docdid "github.com/hyperledger/aries-framework-go/pkg/doc/did"
	"github.com/spf13/cobra"
	gojose "github.com/square/go-jose/v3"
	cmdutils "github.com/trustbloc/edge-core/pkg/utils/cmd"
	tlsutils "github.com/trustbloc/edge-core/pkg/utils/tls"

	"github.com/trustbloc/trustbloc-did-method/pkg/did"
	"github.com/trustbloc/trustbloc-did-method/pkg/restapi/didmethod/operation"
)

const (
	didURIFlagName  = "did-uri"
	didURIEnvKey    = "DID_METHOD_CLI_DID_URI"
	didURIFlagUsage = "DID URI. " +
		" Alternatively, this can be set with the following environment variable: " + didURIEnvKey

	domainFlagName      = "domain"
	domainFileEnvKey    = "DID_METHOD_CLI_DOMAIN"
	domainFileFlagUsage = "URL to the did:trustbloc consortium's domain. " +
		" Alternatively, this can be set with the following environment variable: " + domainFileEnvKey

	sidetreeURLFlagName  = "sidetree-url"
	sidetreeURLFlagUsage = "Comma-Separated list of sidetree url." +
		" Alternatively, this can be set with the following environment variable: " + sidetreeURLEnvKey
	sidetreeURLEnvKey = "DID_METHOD_CLI_SIDETREE_URL"

	tlsSystemCertPoolFlagName  = "tls-systemcertpool"
	tlsSystemCertPoolFlagUsage = "Use system certificate pool." +
		" Possible values [true] [false]. Defaults to false if not set." +
		" Alternatively, this can be set with the following environment variable: " + tlsSystemCertPoolEnvKey
	tlsSystemCertPoolEnvKey = "DID_METHOD_CLI_TLS_SYSTEMCERTPOOL"

	tlsCACertsFlagName  = "tls-cacerts"
	tlsCACertsFlagUsage = "Comma-Separated list of ca certs path." +
		" Alternatively, this can be set with the following environment variable: " + tlsCACertsEnvKey
	tlsCACertsEnvKey = "DID_METHOD_CLI_TLS_CACERTS"

	sidetreeWriteTokenFlagName  = "sidetree-write-token"
	sidetreeWriteTokenEnvKey    = "DID_METHOD_CLI_SIDETREE_WRITE_TOKEN" //nolint: gosec
	sidetreeWriteTokenFlagUsage = "The sidetree write token " +
		" Alternatively, this can be set with the following environment variable: " + sidetreeWriteTokenEnvKey

	addPublicKeyFileFlagName  = "add-publickey-file"
	addPublicKeyFileEnvKey    = "DID_METHOD_CLI_ADD_PUBLICKEY_FILE"
	addPublicKeyFileFlagUsage = "publickey file include public keys to be added for TrustBloc DID " +
		" Alternatively, this can be set with the following environment variable: " + addPublicKeyFileEnvKey

	addServiceFileFlagName = "add-service-file"
	addServiceFileEnvKey   = "DID_METHOD_CLI_ADD_SERVICE_FILE"
	addServiceFlagUsage    = "publickey file include services to be added for TrustBloc DID " +
		" Alternatively, this can be set with the following environment variable: " + addServiceFileEnvKey

	removePublicKeyIDFlagName  = "remove-publickey-id"
	removePublicKeyIDEnvKey    = "DID_METHOD_CLI_REMOVE_PUBLICKEY_ID"
	removePublicKeyIDFlagUsage = "Comma-Separated list of public key id's to be removed from TrustBloc DID. " +
		" Alternatively, this can be set with the following environment variable: " + removePublicKeyIDEnvKey

	removeServiceIDFlagName  = "remove-service-id"
	removeServiceIDEnvKey    = "DID_METHOD_CLI_REMOVE_SERVICE_ID"
	removeServiceIDFlagUsage = "Comma-Separated list of service id's to be removed from TrustBloc DID. " +
		" Alternatively, this can be set with the following environment variable: " + removeServiceIDEnvKey

	signingKeyFlagName  = "signingkey"
	signingKeyEnvKey    = "DID_METHOD_CLI_SIGNINGKEY"
	signingKeyFlagUsage = "The private key PEM used for signing the update of the index document. Example: --signingkey 'MHcCAQEEILmfa4yss8nsTJK2hKl+LAoiwW3p+eQzaHfITI9z8ptpoAoGCCqGSM49AwEHoUQDQgAEMd1/e/Nxh73bK12PEEcNSY9HxnP0N8er9ww9rjq1tNcsqfRjlL0bdTh9Basfn/4JrQHUHc6uS99yjQc+0u2bVg' " +
		" Alternatively, this can be set with the following environment variable: " + signingKeyEnvKey

	signingKeyFileFlagName  = "signingkey-file"
	signingKeyFileEnvKey    = "DID_METHOD_CLI_SIGNINGKEY_FILE"
	signingKeyFileFlagUsage = "The file that contains the private key PEM used for signing the update of the index document. Example: --signingkey-file ./keys/signing.key " +
		" Alternatively, this can be set with the following environment variable: " + signingKeyFileEnvKey

	signingKeyPasswordFlagName  = "signingkey-password"
	signingKeyPasswordEnvKey    = "DID_METHOD_CLI_SIGNINGKEY_PASSWORD"
	signingKeyPasswordFlagUsage = "signing key pem password. " +
		" Alternatively, this can be set with the following environment variable: " + signingKeyPasswordEnvKey

	nextUpdateKeyFlagName  = "nextupdatekey"
	nextUpdateKeyEnvKey    = "DID_METHOD_CLI_NEXTUPDATEKEY"
	nextUpdateKeyFlagUsage = "The public key PEM used for creating commitment for next update of the index document. Example: --nextupdatekey 'MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEFMy2n9jYZChYSjdhK9vUWvPjz9tzBcEa13Ye33haxFsT//3kGxOQhI7yb3MJsDvwLtdfLL6txM3RdOrmLABBvw' " +
		" Alternatively, this can be set with the following environment variable: " + nextUpdateKeyEnvKey

	nextUpdateKeyFileFlagName  = "nextupdatekey-file"
	nextUpdateKeyFileEnvKey    = "DID_METHOD_CLI_NEXTUPDATEKEY_FILE"
	nextUpdateKeyFileFlagUsage = "The file that contains the public key PEM used for creating commitment for next update of the index document. Example: --nextupdatekey-file ./next_update_public.key " +
		" Alternatively, this can be set with the following environment variable: " + nextUpdateKeyFileEnvKey
)

type publicKey struct {
	Type     string   `json:"type,omitempty"`
	Purposes []string `json:"purposes,omitempty"`
	JWKPath  string   `json:"jwkPath,omitempty"`
}

// GetUpdateDIDCmd returns the Cobra update did command.
func GetUpdateDIDCmd() *cobra.Command {
	updateDIDCmd := updateDIDCmd()

	createFlags(updateDIDCmd)

	return updateDIDCmd
}

func updateDIDCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update-did",
		Short: "Update TrustBloc DID",
		Long:  "Update TrustBloc DID",
		RunE: func(cmd *cobra.Command, args []string) error {
			rootCAs, err := getRootCAs(cmd)
			if err != nil {
				return err
			}

			didURI, err := cmdutils.GetUserSetVarFromString(cmd, didURIFlagName,
				didURIEnvKey, false)
			if err != nil {
				return err
			}

			sidetreeWriteToken, err := cmdutils.GetUserSetVarFromString(cmd, sidetreeWriteTokenFlagName,
				sidetreeWriteTokenEnvKey, true)
			if err != nil {
				return err
			}

			domain, err := cmdutils.GetUserSetVarFromString(cmd, domainFlagName,
				domainFileEnvKey, true)
			if err != nil {
				return err
			}

			client := did.New(did.WithAuthToken(sidetreeWriteToken),
				did.WithTLSConfig(&tls.Config{RootCAs: rootCAs}))

			opts, err := updateDIDOption(cmd)
			if err != nil {
				return err
			}

			err = client.UpdateDID(didURI, domain, opts...)
			if err != nil {
				return err
			}

			fmt.Println(fmt.Sprintf("successfully updated DID %s", didURI))

			return nil
		},
	}
}

func getSidetreeURL(cmd *cobra.Command) ([]did.UpdateDIDOption, error) {
	var opts []did.UpdateDIDOption

	sidetreeURL, err := cmdutils.GetUserSetVarFromArrayString(cmd, sidetreeURLFlagName,
		sidetreeURLEnvKey, true)
	if err != nil {
		return nil, err
	}

	for _, v := range sidetreeURL {
		opts = append(opts, did.WithUpdateSidetreeEndpoint(v))
	}

	return opts, nil
}

func updateDIDOption(cmd *cobra.Command) ([]did.UpdateDIDOption, error) {
	opts, err := getPublicKeys(cmd)
	if err != nil {
		return nil, err
	}

	signingKeyOpts, err := getKey(cmd, signingKeyFlagName, signingKeyEnvKey, signingKeyFileFlagName,
		signingKeyFileEnvKey, true)
	if err != nil {
		return nil, err
	}

	opts = append(opts, signingKeyOpts...)

	updateKeyOpts, err := getKey(cmd, nextUpdateKeyFlagName, nextUpdateKeyEnvKey, nextUpdateKeyFileFlagName,
		nextUpdateKeyFileEnvKey, false)
	if err != nil {
		return nil, err
	}

	opts = append(opts, updateKeyOpts...)

	serviceOpts, err := getServices(cmd)
	if err != nil {
		return nil, err
	}

	opts = append(opts, serviceOpts...)

	sidetreeURLOpts, err := getSidetreeURL(cmd)
	if err != nil {
		return nil, err
	}

	opts = append(opts, sidetreeURLOpts...)

	removePublicKeyOpts, err := getRemovePublicKeyID(cmd)
	if err != nil {
		return nil, err
	}

	opts = append(opts, removePublicKeyOpts...)

	removeServiceOpts, err := getRemoveServiceID(cmd)
	if err != nil {
		return nil, err
	}

	opts = append(opts, removeServiceOpts...)

	return opts, nil
}

func getRemoveServiceID(cmd *cobra.Command) ([]did.UpdateDIDOption, error) {
	var opts []did.UpdateDIDOption

	removeServices, err := cmdutils.GetUserSetVarFromArrayString(cmd, removeServiceIDFlagName,
		removeServiceIDEnvKey, true)
	if err != nil {
		return nil, err
	}

	for _, v := range removeServices {
		opts = append(opts, did.WithRemoveService(v))
	}

	return opts, nil
}

func getRemovePublicKeyID(cmd *cobra.Command) ([]did.UpdateDIDOption, error) {
	var opts []did.UpdateDIDOption

	removePublicKeys, err := cmdutils.GetUserSetVarFromArrayString(cmd, removePublicKeyIDFlagName,
		removePublicKeyIDEnvKey, true)
	if err != nil {
		return nil, err
	}

	for _, v := range removePublicKeys {
		opts = append(opts, did.WithRemovePublicKey(v))
	}

	return opts, nil
}

func getServices(cmd *cobra.Command) ([]did.UpdateDIDOption, error) {
	var opts []did.UpdateDIDOption

	serviceFile, err := cmdutils.GetUserSetVarFromString(cmd, addServiceFileFlagName,
		addServiceFileEnvKey, true)
	if err != nil {
		return nil, err
	}

	if serviceFile != "" {
		svcData, err := ioutil.ReadFile(filepath.Clean(serviceFile))
		if err != nil {
			return nil, fmt.Errorf("failed to service file '%s' : %w", serviceFile, err)
		}

		var services []operation.Service
		if err := json.Unmarshal(svcData, &services); err != nil {
			return nil, err
		}

		for _, v := range services {
			opts = append(opts, did.WithAddService(&docdid.Service{ID: v.ID, Type: v.Type,
				Priority: v.Priority, RecipientKeys: v.RecipientKeys, RoutingKeys: v.RoutingKeys,
				ServiceEndpoint: v.Endpoint}))
		}
	}

	return opts, nil
}

func getKey(cmd *cobra.Command, keyFlagName, keyEnvKey, keyFileFlagName, keyFileEnvKey string,
	signing bool) ([]did.UpdateDIDOption, error) {
	keyString, err := cmdutils.GetUserSetVarFromString(cmd, keyFlagName,
		keyEnvKey, true)
	if err != nil {
		return nil, err
	}

	keyFile, err := cmdutils.GetUserSetVarFromString(cmd, keyFileFlagName,
		keyFileEnvKey, true)
	if err != nil {
		return nil, err
	}

	if keyString == "" && keyFile == "" {
		return nil, fmt.Errorf("either key (--%s) or key file (--%s) is required", keyFlagName, keyFileFlagName)
	}

	if keyString != "" && keyFile != "" {
		return nil, fmt.Errorf("only one of key (--%s) or key file (--%s) may be specified", keyFlagName, keyFileFlagName)
	}

	var opts []did.UpdateDIDOption

	if signing {
		var privKey crypto.PrivateKey

		password, err := cmdutils.GetUserSetVarFromString(cmd, signingKeyPasswordFlagName,
			signingKeyPasswordEnvKey, true)
		if err != nil {
			return nil, err
		}

		if keyFile != "" {
			privKey, err = privateKeyFromFile(keyFile, []byte(password))
			if err != nil {
				return nil, err
			}
		} else {
			privKey, err = privateKeyFromPEM([]byte(keyString), []byte(password))
			if err != nil {
				return nil, err
			}
		}

		opts = append(opts, did.WithSigningKey(privKey))

		return opts, nil
	}
	var pubKey crypto.PublicKey
	if keyFile != "" {
		pubKey, err = publicKeyFromFile(keyFile)
		if err != nil {
			return nil, err
		}
	} else {
		pubKey, err = publicKeyFromPEM([]byte(keyString))
		if err != nil {
			return nil, err
		}
	}

	opts = append(opts, did.WithNextUpdatePublicKey(pubKey))

	return opts, nil
}

func getPublicKeys(cmd *cobra.Command) ([]did.UpdateDIDOption, error) {
	var opts []did.UpdateDIDOption

	publicKeyFile, err := cmdutils.GetUserSetVarFromString(cmd, addPublicKeyFileFlagName,
		addPublicKeyFileEnvKey, true)
	if err != nil {
		return nil, err
	}

	if publicKeyFile != "" {
		pkData, err := ioutil.ReadFile(filepath.Clean(publicKeyFile))
		if err != nil {
			return nil, fmt.Errorf("failed to public key file '%s' : %w", publicKeyFile, err)
		}

		var publicKeys []publicKey
		if err := json.Unmarshal(pkData, &publicKeys); err != nil {
			return nil, err
		}

		for _, v := range publicKeys {
			jwkData, err := ioutil.ReadFile(filepath.Clean(v.JWKPath))
			if err != nil {
				return nil, fmt.Errorf("failed to read jwk file '%s' : %w", v.JWKPath, err)
			}

			var jsonWebKey gojose.JSONWebKey
			if err := jsonWebKey.UnmarshalJSON(jwkData); err != nil {
				return nil, fmt.Errorf("failed to unmarshal to jwk: %w", err)
			}

			keyType := ""
			var value []byte

			switch key := jsonWebKey.Key.(type) {
			case ed25519.PublicKey:
				keyType = did.Ed25519KeyType
				value = []byte(fmt.Sprintf("%v", key))
			case *ecdsa.PublicKey:
				if key.Curve.Params().Name != elliptic.P256().Params().Name {
					return nil, fmt.Errorf("ec cruve %s key not supported", elliptic.P256().Params().Name)
				}
				keyType = did.P256KeyType
				value = elliptic.Marshal(key.Curve, key.X, key.Y)
			default:
				return nil, fmt.Errorf("key not supported")
			}

			opts = append(opts, did.WithAddPublicKey(&did.PublicKey{ID: jsonWebKey.KeyID, Type: v.Type,
				Value: value, Encoding: did.PublicKeyEncodingJwk, Purposes: v.Purposes, KeyType: keyType}))
		}
	}

	return opts, nil
}

func getRootCAs(cmd *cobra.Command) (*x509.CertPool, error) {
	tlsSystemCertPoolString, err := cmdutils.GetUserSetVarFromString(cmd, tlsSystemCertPoolFlagName,
		tlsSystemCertPoolEnvKey, true)
	if err != nil {
		return nil, err
	}

	tlsSystemCertPool := false
	if tlsSystemCertPoolString != "" {
		tlsSystemCertPool, err = strconv.ParseBool(tlsSystemCertPoolString)
		if err != nil {
			return nil, err
		}
	}

	tlsCACerts, err := cmdutils.GetUserSetVarFromArrayString(cmd, tlsCACertsFlagName,
		tlsCACertsEnvKey, true)
	if err != nil {
		return nil, err
	}

	return tlsutils.GetCertPool(tlsSystemCertPool, tlsCACerts)
}

func publicKeyFromFile(file string) (crypto.PublicKey, error) {
	keyBytes, err := ioutil.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	return publicKeyFromPEM(keyBytes)
}

func publicKeyFromPEM(pubKeyPEM []byte) (crypto.PublicKey, error) {
	block, _ := pem.Decode(pubKeyPEM)
	if block == nil {
		return nil, fmt.Errorf("public key not found in PEM")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	publicKey, ok := key.(crypto.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid public key")
	}

	return publicKey, nil
}

func privateKeyFromFile(file string, password []byte) (crypto.PrivateKey, error) {
	keyBytes, err := ioutil.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	return privateKeyFromPEM(keyBytes, password)
}

func privateKeyFromPEM(privateKeyPEM, password []byte) (crypto.PrivateKey, error) {
	privBlock, _ := pem.Decode(privateKeyPEM)
	if privBlock == nil {
		return nil, fmt.Errorf("private key not found in PEM")
	}

	bytes := privBlock.Bytes

	if len(password) != 0 {
		var err error
		bytes, err = x509.DecryptPEMBlock(privBlock, password)
		if err != nil {
			return nil, err
		}
	}

	privKey, err := parsePrivateKey(bytes)
	if err != nil {
		return nil, err
	}

	return privKey, nil
}

func parsePrivateKey(der []byte) (crypto.PrivateKey, error) {
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case ed25519.PrivateKey, *ecdsa.PrivateKey:
			return key, nil
		default:
			return nil, fmt.Errorf("found unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}
	return nil, fmt.Errorf("failed to parse private key")
}

func createFlags(startCmd *cobra.Command) {
	startCmd.Flags().StringP(didURIFlagName, "", "", didURIFlagUsage)
	startCmd.Flags().StringP(domainFlagName, "", "", domainFileFlagUsage)
	startCmd.Flags().StringP(tlsSystemCertPoolFlagName, "", "",
		tlsSystemCertPoolFlagUsage)
	startCmd.Flags().StringArrayP(tlsCACertsFlagName, "", []string{}, tlsCACertsFlagUsage)
	startCmd.Flags().StringP(sidetreeWriteTokenFlagName, "", "", sidetreeWriteTokenFlagUsage)
	startCmd.Flags().StringP(addPublicKeyFileFlagName, "", "", addPublicKeyFileFlagUsage)
	startCmd.Flags().StringP(addServiceFileFlagName, "", "", addServiceFlagUsage)
	startCmd.Flags().StringP(signingKeyFlagName, "", "", signingKeyFlagUsage)
	startCmd.Flags().StringP(signingKeyFileFlagName, "", "", signingKeyFileFlagUsage)
	startCmd.Flags().StringP(nextUpdateKeyFlagName, "", "", nextUpdateKeyFlagUsage)
	startCmd.Flags().StringP(nextUpdateKeyFileFlagName, "", "", nextUpdateKeyFileFlagUsage)
	startCmd.Flags().StringArrayP(sidetreeURLFlagName, "", []string{}, sidetreeURLFlagUsage)
	startCmd.Flags().StringArrayP(removePublicKeyIDFlagName, "", []string{}, removePublicKeyIDFlagUsage)
	startCmd.Flags().StringArrayP(removeServiceIDFlagName, "", []string{}, removeServiceIDFlagUsage)
	startCmd.Flags().StringP(signingKeyPasswordFlagName, "", "", signingKeyPasswordFlagUsage)
}
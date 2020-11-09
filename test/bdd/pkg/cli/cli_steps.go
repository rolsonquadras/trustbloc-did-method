/*
Copyright SecureKey Technologies Inc. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package cli

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/cucumber/godog"
	ariesdid "github.com/hyperledger/aries-framework-go/pkg/doc/did"

	"github.com/trustbloc/trustbloc-did-method/pkg/vdri/trustbloc"
	"github.com/trustbloc/trustbloc-did-method/test/bdd/pkg/context"
)

// Steps is steps for cli BDD tests.
type Steps struct {
	bddContext *context.BDDContext
	cliValue   string
	createdDID *ariesdid.Doc
}

// NewSteps returns new agent from client SDK.
func NewSteps(ctx *context.BDDContext) *Steps {
	return &Steps{bddContext: ctx}
}

// RegisterSteps registers agent steps.
func (e *Steps) RegisterSteps(s *godog.Suite) {
	s.Step(`^TrustBloc DID is created through cli using domain "([^"]*)", direct url "([^"]*)"$`, e.createDID)
	s.Step(`^TrustBloc DID is updated through cli using domain "([^"]*)", direct url "([^"]*)"$`, e.updateDID)
	s.Step(`^TrustBloc DID is recovered through cli using domain "([^"]*)", direct url "([^"]*)"$`, e.recoverDID)
	s.Step(`^check cli created valid DID$`, e.checkCreatedDID)
	s.Step(`^check cli recovered DID$`, e.checkRecoveredDID)
	s.Step(`^check cli updated DID$`, e.checkUpdatedDID)
}

func (e *Steps) resolveDID(did string) (*ariesdid.Doc, error) {
	const maxRetry = 10

	blocVDRI := trustbloc.New(trustbloc.WithTLSConfig(e.bddContext.TLSConfig),
		trustbloc.WithAuthToken("rw_token"), trustbloc.WithDomain("testnet.trustbloc.local"))

	var doc *ariesdid.Doc

	var err error

	for i := 1; i <= maxRetry; i++ {
		doc, err = blocVDRI.Read(did)

		if err != nil && (!strings.Contains(err.Error(), "DID does not exist") || i == maxRetry) {
			return nil, err
		}

		time.Sleep(1 * time.Second)
	}

	return doc, nil
}

func (e *Steps) checkCreatedDID() error {
	const numberOfPublicKeys = 2

	const numberOfServices = 2

	doc, err := ariesdid.ParseDocument([]byte(e.cliValue))
	if err != nil {
		return err
	}

	doc, err = e.resolveDID(doc.ID)
	if err != nil {
		return err
	}

	if len(doc.PublicKey) != numberOfPublicKeys {
		return fmt.Errorf("did doc public key is not equal to %d", numberOfPublicKeys)
	}

	if len(doc.Service) != numberOfServices {
		return fmt.Errorf("did doc services is not equal to %d", numberOfServices)
	}

	e.createdDID = doc

	return nil
}

func (e *Steps) checkRecoveredDID() error {
	const numberOfPublicKeys = 1

	const numberOfServices = 1

	doc, err := e.resolveDID(e.createdDID.ID)
	if err != nil {
		return err
	}

	if len(doc.PublicKey) != numberOfPublicKeys {
		return fmt.Errorf("did doc public key is not equal to %d", numberOfPublicKeys)
	}

	if len(doc.Service) != numberOfServices {
		return fmt.Errorf("did doc services is not equal to %d", numberOfServices)
	}

	if !strings.Contains(doc.PublicKey[0].ID, "key-recover-id") {
		return fmt.Errorf("wrong recoverd public key")
	}

	if !strings.Contains(doc.Service[0].ID, "svc-recover-id") {
		return fmt.Errorf("wrong recoverd service")
	}

	return nil
}

func (e *Steps) checkUpdatedDID() error { //nolint: gocyclo
	const numberOfPublicKeys = 2

	const numberOfServices = 1

	doc, err := e.resolveDID(e.createdDID.ID)
	if err != nil {
		return err
	}

	if len(doc.PublicKey) != numberOfPublicKeys {
		return fmt.Errorf("did doc public key is not equal to %d", numberOfPublicKeys)
	}

	key2ID := "key2"
	key3ID := "key3"
	svc3ID := "svc3"

	key2Exist := false
	key3Exist := false

	if len(doc.CapabilityInvocation) != 1 {
		return fmt.Errorf("did capability invocation is not equal to 1")
	}

	if !strings.Contains(doc.CapabilityInvocation[0].PublicKey.ID, key2ID) {
		return fmt.Errorf("wrong capability invocation key")
	}

	for _, v := range doc.PublicKey {
		if strings.Contains(v.ID, key2ID) {
			key2Exist = true
			continue
		}

		if strings.Contains(v.ID, key3ID) {
			key3Exist = true
			continue
		}
	}

	if !key2Exist || !key3Exist {
		return fmt.Errorf("wrong updated public key")
	}

	if len(doc.Service) != numberOfServices {
		return fmt.Errorf("did doc services is not equal to %d", numberOfServices)
	}

	if !strings.Contains(doc.Service[0].ID, svc3ID) {
		return fmt.Errorf("wrong updated service")
	}

	return nil
}

func (e *Steps) updateDID(domain, sidetreeURL string) error {
	var args []string

	if domain != "" {
		args = append(args, "--domain", domain)
	}

	if sidetreeURL != "" {
		args = append(args, "--sidetree-url", sidetreeURL)
	}

	args = append(args, "update-did", "--did-uri", e.createdDID.ID, "--tls-cacerts",
		"fixtures/keys/tls/ec-cacert.pem", "--add-publickey-file", "fixtures/did-keys/update/publickeys.json",
		"--sidetree-write-token", "rw_token", "--signingkey-file", "./fixtures/keys/update/key_encrypted.pem",
		"--signingkey-password", "123", "--nextupdatekey-file", "./fixtures/keys/update2/public.pem",
		"--remove-publickey-id", "key1", "--remove-service-id", "svc1", "--remove-service-id", "svc2",
		"--add-service-file", "fixtures/did-services/update/services.json")

	value, err := execCMD("../../.build/bin/cli", args...)

	if err != nil {
		return err
	}

	e.cliValue = value

	return nil
}

func (e *Steps) recoverDID(domain, sidetreeURL string) error {
	var args []string

	if domain != "" {
		args = append(args, "--domain", domain)
	}

	if sidetreeURL != "" {
		args = append(args, "--sidetree-url", sidetreeURL)
	}

	args = append(args, "recover-did", "--did-uri", e.createdDID.ID, "--signingkey-password", "123",
		"--tls-cacerts", "fixtures/keys/tls/ec-cacert.pem",
		"--publickey-file", "fixtures/did-keys/recover/publickeys.json", "--sidetree-write-token", "rw_token",
		"--service-file", "fixtures/did-services/recover/services.json",
		"--nextrecoverkey-file", "./fixtures/keys/recover2/public.pem", "--nextupdatekey-file",
		"./fixtures/keys/update3/public.pem", "--signingkey-file", "./fixtures/keys/recover/key_encrypted.pem")

	value, err := execCMD("../../.build/bin/cli", args...)

	if err != nil {
		return err
	}

	e.cliValue = value

	return nil
}

func (e *Steps) createDID(domain, sidetreeURL string) error {
	var args []string

	if domain != "" {
		args = append(args, "--domain", domain)
	}

	if sidetreeURL != "" {
		args = append(args, "--sidetree-url", sidetreeURL)
	}

	args = append(args, "create-did",
		"--tls-cacerts", "fixtures/keys/tls/ec-cacert.pem", "--publickey-file", "fixtures/did-keys/create/publickeys.json",
		"--sidetree-write-token", "rw_token", "--service-file", "fixtures/did-services/create/services.json",
		"--recoverykey-file", "./fixtures/keys/recover/public.pem", "--updatekey-file", "./fixtures/keys/update/public.pem")

	value, err := execCMD("../../.build/bin/cli", args...)

	if err != nil {
		return err
	}

	e.cliValue = value

	return nil
}

func execCMD(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...) // nolint: gosec

	var out bytes.Buffer

	var stderr bytes.Buffer

	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf(fmt.Sprint(err) + ": " + stderr.String())
	}

	return out.String(), nil
}

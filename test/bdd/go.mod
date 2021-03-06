// Copyright SecureKey Technologies Inc. All Rights Reserved.
//
// SPDX-License-Identifier: Apache-2.0

module github.com/trustbloc/trustbloc-did-method/test/bdd

replace github.com/trustbloc/trustbloc-did-method => ../..

replace github.com/kilic/bls12-381 => github.com/trustbloc/bls12-381 v0.0.0-20201104214312-31de2a204df8

go 1.15

require (
	github.com/cucumber/godog v0.9.0
	github.com/fsouza/go-dockerclient v1.6.0
	github.com/google/uuid v1.1.2
	github.com/hyperledger/aries-framework-go v0.1.5-0.20201110161050-249e1c428734
	github.com/sirupsen/logrus v1.6.0
	github.com/tidwall/gjson v1.6.0
	github.com/trustbloc/edge-core v0.1.5-0.20201106164919-76ecfeca954f
	github.com/trustbloc/trustbloc-did-method v0.0.0
	gotest.tools/v3 v3.0.3 // indirect
)

// https://github.com/ory/dockertest/issues/208#issuecomment-686820414
replace golang.org/x/sys => golang.org/x/sys v0.0.0-20200826173525-f9321e4c35a6

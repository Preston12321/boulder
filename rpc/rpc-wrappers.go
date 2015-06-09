// Copyright 2014 ISRG.  All rights reserved
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package rpc

import (
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	jose "github.com/letsencrypt/boulder/Godeps/_workspace/src/github.com/square/go-jose"
	"github.com/letsencrypt/boulder/core"
	blog "github.com/letsencrypt/boulder/log"
)

// This file defines RPC wrappers around the ${ROLE}Impl classes,
// where ROLE covers:
//  * RegistrationAuthority
//  * ValidationAuthority
//  * CertficateAuthority
//  * StorageAuthority
//
// For each one of these, the are ${ROLE}Client and ${ROLE}Server
// types.  ${ROLE}Server is to be run on the server side, as a more
// or less stand-alone component.  ${ROLE}Client is loaded by the
// code making use of the functionality.
//
// The WebFrontEnd role does not expose any functionality over RPC,
// so it doesn't need wrappers.

const (
	MethodNewRegistration             = "NewRegistration"             // RA, SA
	MethodNewAuthorization            = "NewAuthorization"            // RA
	MethodNewCertificate              = "NewCertificate"              // RA
	MethodUpdateRegistration          = "UpdateRegistration"          // RA, SA
	MethodUpdateAuthorization         = "UpdateAuthorization"         // RA
	MethodRevokeCertificate           = "RevokeCertificate"           // RA, CA
	MethodOnValidationUpdate          = "OnValidationUpdate"          // RA
	MethodUpdateValidations           = "UpdateValidations"           // VA
	MethodIssueCertificate            = "IssueCertificate"            // CA
	MethodGenerateOCSP                = "GenerateOCSP"                // CA
	MethodGetRegistration             = "GetRegistration"             // SA
	MethodGetRegistrationByKey        = "GetRegistrationByKey"        // RA, SA
	MethodGetAuthorization            = "GetAuthorization"            // SA
	MethodGetCertificate              = "GetCertificate"              // SA
	MethodGetCertificateByShortSerial = "GetCertificateByShortSerial" // SA
	MethodGetCertificateStatus        = "GetCertificateStatus"        // SA
	MethodMarkCertificateRevoked      = "MarkCertificateRevoked"      // SA
	MethodNewPendingAuthorization     = "NewPendingAuthorization"     // SA
	MethodUpdatePendingAuthorization  = "UpdatePendingAuthorization"  // SA
	MethodFinalizeAuthorization       = "FinalizeAuthorization"       // SA
	MethodAddCertificate              = "AddCertificate"              // SA
	MethodAlreadyDeniedCSR            = "AlreadyDeniedCSR"            // SA
)

// RegistrationAuthorityClient / Server
//  -> NewAuthorization
//  -> NewCertificate
//  -> UpdateAuthorization
//  -> RevokeCertificate
//  -> OnValidationUpdate
type registrationRequest struct {
	Reg core.Registration
}

type authorizationRequest struct {
	Authz core.Authorization
	RegID int64
}

type certificateRequest struct {
	Req   core.CertificateRequest
	RegID int64
}

func improperMessage(method string, err error, obj interface{}) {
	log := blog.GetAuditLogger()
	log.Audit(fmt.Sprintf("Improper message. method: %s err: %s data: %+v", method, err, obj))
}
func errorCondition(method string, err error, obj interface{}) {
	log := blog.GetAuditLogger()
	log.Audit(fmt.Sprintf("Error condition. method: %s err: %s data: %+v", method, err, obj))
}

func NewRegistrationAuthorityServer(rpc RPCServer, impl core.RegistrationAuthority) error {
	log := blog.GetAuditLogger()

	rpc.Handle(MethodNewRegistration, func(req []byte) (response []byte, err error) {
		var rr registrationRequest
		if err = json.Unmarshal(req, &rr); err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodNewRegistration, err, req)
			return
		}

		reg, err := impl.NewRegistration(rr.Reg)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodNewRegistration, err, reg)
			return
		}

		response, err = json.Marshal(reg)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodNewRegistration, err, req)
			return
		}
		return
	})

	rpc.Handle(MethodNewAuthorization, func(req []byte) (response []byte, err error) {
		var ar authorizationRequest
		if err = json.Unmarshal(req, &ar); err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodNewAuthorization, err, req)
			return
		}

		authz, err := impl.NewAuthorization(ar.Authz, ar.RegID)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodNewAuthorization, err, ar)
			return
		}

		response, err = json.Marshal(authz)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodNewAuthorization, err, req)
			return
		}
		return
	})

	rpc.Handle(MethodNewCertificate, func(req []byte) (response []byte, err error) {
		log.Info(fmt.Sprintf(" [.] Entering MethodNewCertificate"))
		var cr certificateRequest
		if err = json.Unmarshal(req, &cr); err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodNewCertificate, err, req)
			return
		}
		log.Info(fmt.Sprintf(" [.] No problem unmarshaling request"))

		cert, err := impl.NewCertificate(cr.Req, cr.RegID)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodNewCertificate, err, cr)
			return
		}
		log.Info(fmt.Sprintf(" [.] No problem issuing new cert"))

		response, err = json.Marshal(cert)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodNewCertificate, err, req)
			return
		}
		return
	})

	rpc.Handle(MethodUpdateRegistration, func(req []byte) (response []byte, err error) {
		var request struct {
			Base, Update core.Registration
		}
		err = json.Unmarshal(req, &request)
		if err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodUpdateRegistration, err, req)
			return
		}

		reg, err := impl.UpdateRegistration(request.Base, request.Update)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodUpdateRegistration, err, request)
			return
		}

		response, err = json.Marshal(reg)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodUpdateRegistration, err, req)
			return
		}
		return
	})

	rpc.Handle(MethodUpdateAuthorization, func(req []byte) (response []byte, err error) {
		var authz struct {
			Authz    core.Authorization
			Index    int
			Response core.Challenge
		}
		err = json.Unmarshal(req, &authz)
		if err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodUpdateAuthorization, err, req)
			return
		}

		newAuthz, err := impl.UpdateAuthorization(authz.Authz, authz.Index, authz.Response)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodUpdateAuthorization, err, authz)
			return
		}

		response, err = json.Marshal(newAuthz)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodUpdateAuthorization, err, req)
			return
		}
		return
	})

	rpc.Handle(MethodRevokeCertificate, func(req []byte) (response []byte, err error) {
		certs, err := x509.ParseCertificates(req)
		if err != nil || len(certs) == 0 {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodRevokeCertificate, err, req)
			return
		}

		err = impl.RevokeCertificate(*certs[0])
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodRevokeCertificate, err, certs)
		}
		return
	})

	rpc.Handle(MethodOnValidationUpdate, func(req []byte) (response []byte, err error) {
		var authz core.Authorization
		if err = json.Unmarshal(req, &authz); err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodOnValidationUpdate, err, req)
			return
		}

		if err = impl.OnValidationUpdate(authz); err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodOnValidationUpdate, err, authz)
		}
		return
	})

	return nil
}

type RegistrationAuthorityClient struct {
	rpc RPCClient
}

func NewRegistrationAuthorityClient(client RPCClient) (rac RegistrationAuthorityClient, err error) {
	rac = RegistrationAuthorityClient{rpc: client}
	return
}

func (rac RegistrationAuthorityClient) NewRegistration(reg core.Registration) (newReg core.Registration, err error) {
	data, err := json.Marshal(registrationRequest{reg})
	if err != nil {
		return
	}

	newRegData, err := rac.rpc.DispatchSync(MethodNewRegistration, data)
	if err != nil || len(newRegData) == 0 {
		return
	}

	err = json.Unmarshal(newRegData, &newReg)
	return
}

func (rac RegistrationAuthorityClient) NewAuthorization(authz core.Authorization, regID int64) (newAuthz core.Authorization, err error) {
	data, err := json.Marshal(authorizationRequest{authz, regID})
	if err != nil {
		return
	}

	newAuthzData, err := rac.rpc.DispatchSync(MethodNewAuthorization, data)
	if err != nil || len(newAuthzData) == 0 {
		return
	}

	err = json.Unmarshal(newAuthzData, &newAuthz)
	return
}

func (rac RegistrationAuthorityClient) NewCertificate(cr core.CertificateRequest, regID int64) (cert core.Certificate, err error) {
	data, err := json.Marshal(certificateRequest{cr, regID})
	if err != nil {
		return
	}

	certData, err := rac.rpc.DispatchSync(MethodNewCertificate, data)
	if err != nil {
		return
	}
	if len(certData) == 0 {
		err = errors.New("NewCertificate RPC to RA failed.")
		return
	}

	err = json.Unmarshal(certData, &cert)
	return
}

func (rac RegistrationAuthorityClient) UpdateRegistration(base core.Registration, update core.Registration) (newReg core.Registration, err error) {
	var toSend struct{ Base, Update core.Registration }
	toSend.Base = base
	toSend.Update = update

	data, err := json.Marshal(toSend)
	if err != nil {
		return
	}

	newRegData, err := rac.rpc.DispatchSync(MethodUpdateRegistration, data)
	if err != nil || len(newRegData) == 0 {
		return
	}

	err = json.Unmarshal(newRegData, &newReg)
	return
}

func (rac RegistrationAuthorityClient) UpdateAuthorization(authz core.Authorization, index int, response core.Challenge) (newAuthz core.Authorization, err error) {
	var toSend struct {
		Authz    core.Authorization
		Index    int
		Response core.Challenge
	}
	toSend.Authz = authz
	toSend.Index = index
	toSend.Response = response

	data, err := json.Marshal(toSend)
	if err != nil {
		return
	}

	newAuthzData, err := rac.rpc.DispatchSync(MethodUpdateAuthorization, data)
	if err != nil || len(newAuthzData) == 0 {
		return
	}

	err = json.Unmarshal(newAuthzData, &newAuthz)
	return
}

func (rac RegistrationAuthorityClient) RevokeCertificate(cert x509.Certificate) (err error) {
	_, err = rac.rpc.DispatchSync(MethodRevokeCertificate, cert.Raw)
	return
}

func (rac RegistrationAuthorityClient) OnValidationUpdate(authz core.Authorization) (err error) {
	data, err := json.Marshal(authz)
	if err != nil {
		return
	}

	_, err = rac.rpc.DispatchSync(MethodOnValidationUpdate, data)
	return
}

// ValidationAuthorityClient / Server
//  -> UpdateValidations
func NewValidationAuthorityServer(rpc RPCServer, impl core.ValidationAuthority) (err error) {
	rpc.Handle(MethodUpdateValidations, func(req []byte) (response []byte, err error) {
		var vaReq struct {
			Authz core.Authorization
			Index int
		}
		if err = json.Unmarshal(req, &vaReq); err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodUpdateValidations, err, req)
			return
		}

		if err = impl.UpdateValidations(vaReq.Authz, vaReq.Index); err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodUpdateValidations, err, vaReq)
		}
		return
	})

	return nil
}

type ValidationAuthorityClient struct {
	rpc RPCClient
}

func NewValidationAuthorityClient(client RPCClient) (vac ValidationAuthorityClient, err error) {
	vac = ValidationAuthorityClient{rpc: client}
	return
}

func (vac ValidationAuthorityClient) UpdateValidations(authz core.Authorization, index int) error {
	var vaReq struct {
		Authz core.Authorization
		Index int
	}
	vaReq.Authz = authz
	vaReq.Index = index
	data, err := json.Marshal(vaReq)
	if err != nil {
		return err
	}

	_, err = vac.rpc.DispatchSync(MethodUpdateValidations, data)
	return nil
}

// CertificateAuthorityClient / Server
//  -> IssueCertificate
func NewCertificateAuthorityServer(rpc RPCServer, impl core.CertificateAuthority) (err error) {
	rpc.Handle(MethodIssueCertificate, func(req []byte) (response []byte, err error) {
		var icReq struct {
			Bytes          []byte
			RegID          int64
			EarliestExpiry time.Time
		}
		err = json.Unmarshal(req, &icReq)
		if err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodIssueCertificate, err, req)
			return
		}

		csr, err := x509.ParseCertificateRequest(icReq.Bytes)
		if err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodIssueCertificate, err, req)
			return // XXX
		}

		cert, err := impl.IssueCertificate(*csr, icReq.RegID, icReq.EarliestExpiry)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodIssueCertificate, err, csr)
			return // XXX
		}

		response, err = json.Marshal(cert)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodGetRegistration, err, req)
			return // XXX
		}

		return
	})

	rpc.Handle(MethodRevokeCertificate, func(req []byte) (response []byte, err error) {
		var revokeReq struct {
			Serial     string
			ReasonCode int
		}
		err = json.Unmarshal(req, &revokeReq)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodRevokeCertificate, err, req)
			return
		}

		if err := impl.RevokeCertificate(revokeReq.Serial, revokeReq.ReasonCode); err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodRevokeCertificate, err, req)
		}

		return
	})

	rpc.Handle(MethodGenerateOCSP, func(req []byte) (response []byte, err error) {
		var xferObj core.OCSPSigningRequest
		err = json.Unmarshal(req, &xferObj)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodGenerateOCSP, err, req)
			return
		}

		response, err = impl.GenerateOCSP(xferObj)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodGenerateOCSP, err, req)
			return
		}

		return
	})

	return nil
}

type CertificateAuthorityClient struct {
	rpc RPCClient
}

func NewCertificateAuthorityClient(client RPCClient) (cac CertificateAuthorityClient, err error) {
	cac = CertificateAuthorityClient{rpc: client}
	return
}

func (cac CertificateAuthorityClient) IssueCertificate(csr x509.CertificateRequest, regID int64, earliestExpiry time.Time) (cert core.Certificate, err error) {
	var icReq struct {
		Bytes          []byte
		RegID          int64
		EarliestExpiry time.Time
	}
	icReq.Bytes = csr.Raw
	icReq.RegID = regID
	data, err := json.Marshal(icReq)
	if err != nil {
		return
	}

	jsonResponse, err := cac.rpc.DispatchSync(MethodIssueCertificate, data)
	if err != nil {
		return
	}
	if len(jsonResponse) == 0 {
		err = errors.New("IssueCertificate RPC to CA failed.")
		return
	}

	err = json.Unmarshal(jsonResponse, &cert)
	return
}

func (cac CertificateAuthorityClient) RevokeCertificate(serial string, reasonCode int) (err error) {
	var revokeReq struct {
		Serial     string
		ReasonCode int
	}
	revokeReq.Serial = serial
	revokeReq.ReasonCode = reasonCode

	data, err := json.Marshal(revokeReq)
	if err != nil {
		// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
		errorCondition(MethodRevokeCertificate, err, revokeReq)
		return
	}

	_, err = cac.rpc.DispatchSync(MethodRevokeCertificate, data)
	return
}

func (cac CertificateAuthorityClient) GenerateOCSP(signRequest core.OCSPSigningRequest) (resp []byte, err error) {
	data, err := json.Marshal(signRequest)
	if err != nil {
		// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
		errorCondition(MethodGetRegistration, err, signRequest)
		return
	}

	resp, err = cac.rpc.DispatchSync(MethodGenerateOCSP, data)
	return
}

func NewStorageAuthorityServer(rpc RPCServer, impl core.StorageAuthority) error {
	rpc.Handle(MethodUpdateRegistration, func(req []byte) (response []byte, err error) {
		var reg core.Registration
		if err = json.Unmarshal(req, &reg); err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodUpdateRegistration, err, req)
			return
		}

		if err = impl.UpdateRegistration(reg); err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodUpdateRegistration, err, req)
		}

		return
	})

	rpc.Handle(MethodGetRegistration, func(req []byte) (response []byte, err error) {
		var intReq struct {
			ID int64
		}
		err = json.Unmarshal(req, &intReq)
		if err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodGetRegistration, err, req)
			return
		}

		reg, err := impl.GetRegistration(intReq.ID)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodGetRegistration, err, req)
			return
		}

		response, err = json.Marshal(reg)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodGetRegistration, err, req)
			return
		}
		return
	})

	rpc.Handle(MethodGetRegistrationByKey, func(req []byte) (response []byte, err error) {
		var jwk jose.JsonWebKey
		if err = json.Unmarshal(req, &jwk); err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodGetRegistrationByKey, err, req)
			return
		}

		reg, err := impl.GetRegistrationByKey(jwk)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodGetRegistrationByKey, err, jwk)
			return
		}

		response, err = json.Marshal(reg)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodGetRegistrationByKey, err, req)
			return
		}
		return
	})

	rpc.Handle(MethodGetAuthorization, func(req []byte) (response []byte, err error) {
		authz, err := impl.GetAuthorization(string(req))
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodGetAuthorization, err, req)
			return
		}

		response, err = json.Marshal(authz)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodGetAuthorization, err, req)
			return
		}
		return
	})

	rpc.Handle(MethodAddCertificate, func(req []byte) (response []byte, err error) {
		var icReq struct {
			Bytes []byte
			RegID int64
		}
		err = json.Unmarshal(req, &icReq)
		if err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodAddCertificate, err, req)
			return
		}

		id, err := impl.AddCertificate(icReq.Bytes, icReq.RegID)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodAddCertificate, err, req)
			return
		}
		response = []byte(id)
		return
	})

	rpc.Handle(MethodNewRegistration, func(req []byte) (response []byte, err error) {
		var registration core.Registration
		err = json.Unmarshal(req, &registration)
		if err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodNewRegistration, err, req)
			return
		}

		output, err := impl.NewRegistration(registration)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodNewRegistration, err, registration)
			return
		}

		response, err = json.Marshal(output)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodNewRegistration, err, req)
			return
		}
		return
	})

	rpc.Handle(MethodNewPendingAuthorization, func(req []byte) (response []byte, err error) {
		var authz core.Authorization
		if err = json.Unmarshal(req, &authz); err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodNewPendingAuthorization, err, req)
			return
		}

		output, err := impl.NewPendingAuthorization(authz)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodNewPendingAuthorization, err, req)
			return
		}

		response, err = json.Marshal(output)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodNewPendingAuthorization, err, req)
			return
		}
		return
	})

	rpc.Handle(MethodUpdatePendingAuthorization, func(req []byte) (response []byte, err error) {
		var authz core.Authorization
		if err = json.Unmarshal(req, &authz); err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodUpdatePendingAuthorization, err, req)
			return
		}

		if err = impl.UpdatePendingAuthorization(authz); err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodUpdatePendingAuthorization, err, authz)
		}
		return
	})

	rpc.Handle(MethodFinalizeAuthorization, func(req []byte) (response []byte, err error) {
		var authz core.Authorization
		if err = json.Unmarshal(req, &authz); err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodFinalizeAuthorization, err, req)
			return
		}

		if err = impl.FinalizeAuthorization(authz); err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodFinalizeAuthorization, err, authz)
		}
		return
	})

	rpc.Handle(MethodGetCertificate, func(req []byte) (response []byte, err error) {
		cert, err := impl.GetCertificate(string(req))
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodGetCertificate, err, req)
		} else {
			response = []byte(cert)
		}
		return
	})

	rpc.Handle(MethodGetCertificateByShortSerial, func(req []byte) (response []byte, err error) {
		cert, err := impl.GetCertificateByShortSerial(string(req))
		if err != nil {
			if err != sql.ErrNoRows {
				// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
				errorCondition(MethodGetCertificateByShortSerial, err, req)
			}
		} else {
			response = []byte(cert)
		}
		return
	})

	rpc.Handle(MethodGetCertificateStatus, func(req []byte) (response []byte, err error) {
		status, err := impl.GetCertificateStatus(string(req))
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodGetCertificateStatus, err, req)
			return
		}

		response, err = json.Marshal(status)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodGetCertificateStatus, err, req)
			return
		}
		return
	})

	rpc.Handle(MethodMarkCertificateRevoked, func(req []byte) (response []byte, err error) {
		var revokeReq struct {
			Serial       string
			OCSPResponse []byte
			ReasonCode   int
		}

		if err = json.Unmarshal(req, &revokeReq); err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodMarkCertificateRevoked, err, req)
			return
		}

		err = impl.MarkCertificateRevoked(revokeReq.Serial, revokeReq.OCSPResponse, revokeReq.ReasonCode)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodMarkCertificateRevoked, err, revokeReq)
		}
		return
	})

	rpc.Handle(MethodAlreadyDeniedCSR, func(req []byte) (response []byte, err error) {
		var csrReq struct {
			Names []string
		}

		err = json.Unmarshal(req, &csrReq)
		if err != nil {
			// AUDIT[ Improper Messages ] 0786b6f2-91ca-4f48-9883-842a19084c64
			improperMessage(MethodAlreadyDeniedCSR, err, req)
			return
		}

		exists, err := impl.AlreadyDeniedCSR(csrReq.Names)
		if err != nil {
			// AUDIT[ Error Conditions ] 9cc4d537-8534-4970-8665-4b382abe82f3
			errorCondition(MethodAlreadyDeniedCSR, err, csrReq)
			return
		}

		if exists {
			response = []byte{1}
		} else {
			response = []byte{0}
		}
		return
	})

	return nil
}

type StorageAuthorityClient struct {
	rpc RPCClient
}

func NewStorageAuthorityClient(client RPCClient) (sac StorageAuthorityClient, err error) {
	sac = StorageAuthorityClient{rpc: client}
	return
}

func (cac StorageAuthorityClient) GetRegistration(id int64) (reg core.Registration, err error) {
	var intReq struct {
		ID int64
	}
	intReq.ID = id

	data, err := json.Marshal(intReq)
	if err != nil {
		return
	}

	jsonReg, err := cac.rpc.DispatchSync(MethodGetRegistration, data)
	if err != nil {
		return
	}

	err = json.Unmarshal(jsonReg, &reg)
	return
}

func (cac StorageAuthorityClient) GetRegistrationByKey(key jose.JsonWebKey) (reg core.Registration, err error) {
	jsonKey, err := key.MarshalJSON()
	if err != nil {
		return
	}

	jsonReg, err := cac.rpc.DispatchSync(MethodGetRegistrationByKey, jsonKey)
	if err != nil {
		return
	}

	err = json.Unmarshal(jsonReg, &reg)
	return
}

func (cac StorageAuthorityClient) GetAuthorization(id string) (authz core.Authorization, err error) {
	jsonAuthz, err := cac.rpc.DispatchSync(MethodGetAuthorization, []byte(id))
	if err != nil {
		return
	}

	err = json.Unmarshal(jsonAuthz, &authz)
	return
}

func (cac StorageAuthorityClient) GetCertificate(id string) (cert []byte, err error) {
	cert, err = cac.rpc.DispatchSync(MethodGetCertificate, []byte(id))
	return
}

func (cac StorageAuthorityClient) GetCertificateByShortSerial(id string) (cert []byte, err error) {
	cert, err = cac.rpc.DispatchSync(MethodGetCertificateByShortSerial, []byte(id))
	return
}

func (cac StorageAuthorityClient) GetCertificateStatus(id string) (status core.CertificateStatus, err error) {
	jsonStatus, err := cac.rpc.DispatchSync(MethodGetCertificateStatus, []byte(id))
	if err != nil {
		return
	}

	err = json.Unmarshal(jsonStatus, &status)
	return
}

func (cac StorageAuthorityClient) MarkCertificateRevoked(serial string, ocspResponse []byte, reasonCode int) (err error) {
	var revokeReq struct {
		Serial       string
		OCSPResponse []byte
		ReasonCode   int
	}

	revokeReq.Serial = serial
	revokeReq.OCSPResponse = ocspResponse
	revokeReq.ReasonCode = reasonCode

	data, err := json.Marshal(revokeReq)
	if err != nil {
		return
	}

	_, err = cac.rpc.DispatchSync(MethodMarkCertificateRevoked, data)
	return
}

func (cac StorageAuthorityClient) UpdateRegistration(reg core.Registration) (err error) {
	jsonReg, err := json.Marshal(reg)
	if err != nil {
		return
	}

	// XXX: Is this catching all the errors?
	_, err = cac.rpc.DispatchSync(MethodUpdateRegistration, jsonReg)
	return
}

func (cac StorageAuthorityClient) NewRegistration(reg core.Registration) (output core.Registration, err error) {
	jsonReg, err := json.Marshal(reg)
	if err != nil {
		err = errors.New("NewRegistration RPC failed")
		return
	}
	response, err := cac.rpc.DispatchSync(MethodNewRegistration, jsonReg)
	if err != nil || len(response) == 0 {
		err = errors.New("NewRegistration RPC failed") // XXX
		return
	}
	err = json.Unmarshal(response, &output)
	if err != nil {
		err = errors.New("NewRegistration RPC failed")
		return
	}
	return output, nil
}

func (cac StorageAuthorityClient) NewPendingAuthorization(authz core.Authorization) (output core.Authorization, err error) {
	jsonAuthz, err := json.Marshal(authz)
	if err != nil {
		return
	}
	response, err := cac.rpc.DispatchSync(MethodNewPendingAuthorization, jsonAuthz)
	if err != nil || len(response) == 0 {
		err = errors.New("NewPendingAuthorization RPC failed") // XXX
		return
	}
	err = json.Unmarshal(response, &output)
	if err != nil {
		err = errors.New("NewRegistration RPC failed")
		return
	}
	return
}

func (cac StorageAuthorityClient) UpdatePendingAuthorization(authz core.Authorization) (err error) {
	jsonAuthz, err := json.Marshal(authz)
	if err != nil {
		return
	}

	// XXX: Is this catching all the errors?
	_, err = cac.rpc.DispatchSync(MethodUpdatePendingAuthorization, jsonAuthz)
	return
}

func (cac StorageAuthorityClient) FinalizeAuthorization(authz core.Authorization) (err error) {
	jsonAuthz, err := json.Marshal(authz)
	if err != nil {
		return
	}

	// XXX: Is this catching all the errors?
	_, err = cac.rpc.DispatchSync(MethodFinalizeAuthorization, jsonAuthz)
	return
}

func (cac StorageAuthorityClient) AddCertificate(cert []byte, regID int64) (id string, err error) {
	var icReq struct {
		Bytes []byte
		RegID int64
	}
	icReq.Bytes = cert
	icReq.RegID = regID
	data, err := json.Marshal(icReq)
	if err != nil {
		return
	}

	response, err := cac.rpc.DispatchSync(MethodAddCertificate, data)
	if err != nil || len(response) == 0 {
		err = errors.New("AddCertificate RPC failed") // XXX
		return
	}
	id = string(response)
	return
}

func (cac StorageAuthorityClient) AlreadyDeniedCSR(names []string) (exists bool, err error) {
	var sliceReq struct {
		Names []string
	}
	sliceReq.Names = names

	data, err := json.Marshal(sliceReq)
	if err != nil {
		return
	}

	response, err := cac.rpc.DispatchSync(MethodAlreadyDeniedCSR, data)
	if err != nil || len(response) == 0 {
		err = errors.New("AlreadyDeniedCSR RPC failed") // XXX
		return
	}

	switch response[0] {
	case 0:
		exists = false
	case 1:
		exists = true
	}
	return
}

//
// Copyright 2019-2020 Jean-Francois Smigielski
//
// This software is supplied under the terms of the MIT License, a
// copy of which should be located in the distribution where this
// file was obtained (LICENSE.txt). A copy of the license may also be
// found online at https://opensource.org/licenses/MIT.
//

package helpers_grpc

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
)

func DialTLS(dirConfig, addrConnect string) (*grpc.ClientConn, error) {
	cert, _ := ioutil.ReadFile(dirConfig + "/cert.pem")
	//key, _ := ioutil.ReadFile(dirConfig + "/key.pem")

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM([]byte(cert))
	if !ok {
		return nil, errors.New("Invalid certificate")
	}

	creds := credentials.NewClientTLSFromCert(certPool, "")

	return grpc.Dial(addrConnect, grpc.WithTransportCredentials(creds))
}

func ServerTLS(dirConfig string, registerGrpc func(*grpc.Server)) *grpc.Server {
	certBytes, _ := ioutil.ReadFile(dirConfig + "/cert.pem")
	keyBytes, _ := ioutil.ReadFile(dirConfig + "/key.pem")

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(certBytes)
	if !ok {
		panic("bad certs")
	}

	cert, err := tls.X509KeyPair(certBytes, keyBytes)
	if err != nil {
		panic(err.Error())
	}

	creds := credentials.NewServerTLSFromCert(&cert)

	grpcServer := grpc.NewServer(grpc.Creds(creds))
	registerGrpc(grpcServer)
	return grpcServer
}

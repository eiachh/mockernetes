#!/bin/bash
set -e

mkdir -p certs

# CA
openssl genrsa -out certs/ca.key 2048
openssl req -x509 -new -nodes -key certs/ca.key -subj "/CN=mockernetes-ca" -days 3650 -out certs/ca.crt

# Server cert with SANs
openssl genrsa -out certs/server.key 2048
cat > certs/server.conf << EOF
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
subjectAltName = @alt_names
[alt_names]
DNS.1 = localhost
DNS.2 = kubernetes
IP.1 = 127.0.0.1
EOF
openssl req -new -key certs/server.key -subj "/CN=kube-apiserver" -out certs/server.csr
openssl x509 -req -in certs/server.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/server.crt -days 3650 -extensions v3_req -extfile certs/server.conf

# Client cert for admin
openssl genrsa -out certs/client.key 2048
openssl req -new -key certs/client.key -subj "/CN=admin/O=system:masters" -out certs/client.csr
openssl x509 -req -in certs/client.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/client.crt -days 3650

# Kubeconfig
CA_DATA=$(base64 -w 0 certs/ca.crt)
CLIENT_CERT=$(base64 -w 0 certs/client.crt)
CLIENT_KEY=$(base64 -w 0 certs/client.key)
cat > kubeconfig << EOF
apiVersion: v1
kind: Config
clusters:
- cluster:
    certificate-authority-data: $CA_DATA
    server: https://127.0.0.1:8443
  name: mockernetes
contexts:
- context:
    cluster: mockernetes
    user: admin
  name: mockernetes
current-context: mockernetes
users:
- name: admin
  user:
    client-certificate-data: $CLIENT_CERT
    client-key-data: $CLIENT_KEY
EOF

echo "Generated certs in certs/ and kubeconfig"

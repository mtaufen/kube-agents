#!/bin/bash
set -e

NAMESPACE="intent-controller-system"
SERVICE="intent-controller-webhook-service"
SECRET="webhook-server-cert"
WEBHOOK_NAME="intent-controller-validating-webhook-configuration"

echo "Generating self-signed certificate for ${SERVICE}.${NAMESPACE}.svc..."

TMPDIR=$(mktemp -d)
trap 'rm -rf ${TMPDIR}' EXIT

# Create OpenSSL config
cat <<EOF > ${TMPDIR}/csr.conf
[req]
req_extensions = v3_req
distinguished_name = req_distinguished_name
[req_distinguished_name]
[ v3_req ]
basicConstraints = CA:FALSE
keyUsage = nonRepudiation, digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names
[alt_names]
DNS.1 = ${SERVICE}
DNS.2 = ${SERVICE}.${NAMESPACE}
DNS.3 = ${SERVICE}.${NAMESPACE}.svc
EOF

# Generate CA/Cert
openssl genrsa -out ${TMPDIR}/tls.key 2048
openssl req -new -key ${TMPDIR}/tls.key -subj "/CN=${SERVICE}.${NAMESPACE}.svc" -config ${TMPDIR}/csr.conf -out ${TMPDIR}/tls.csr
openssl x509 -req -in ${TMPDIR}/tls.csr -signkey ${TMPDIR}/tls.key -days 3650 -extensions v3_req -extfile ${TMPDIR}/csr.conf -out ${TMPDIR}/tls.crt

echo "Creating TLS secret ${SECRET} in namespace ${NAMESPACE}..."
kubectl create namespace ${NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
kubectl create secret tls ${SECRET} \
    --cert=${TMPDIR}/tls.crt \
    --key=${TMPDIR}/tls.key \
    --namespace=${NAMESPACE} \
    --dry-run=client -o yaml | kubectl apply -f -

echo "Patching ValidatingWebhookConfiguration with the CA bundle..."
CA_BUNDLE=$(cat ${TMPDIR}/tls.crt | base64 | tr -d '\n')
# Wait for the webhook configuration to exist (in case it's newly deployed)
echo "Waiting for ValidatingWebhookConfiguration ${WEBHOOK_NAME} to be available..."
for i in {1..30}; do
    if kubectl get validatingwebhookconfiguration ${WEBHOOK_NAME} >/dev/null 2>&1; then
        break
    fi
    sleep 2
done

kubectl patch validatingwebhookconfiguration ${WEBHOOK_NAME} \
    --type='json' -p="[{'op': 'add', 'path': '/webhooks/0/clientConfig/caBundle', 'value':'${CA_BUNDLE}'}]"

echo "Webhook certificates installed and configured successfully!"

set -e
set -x

rm pods.json | true

witness run -s build -a k8smanifest -k testkey.pem -o k8s-att.json -- kubectl get po -A -o json > pods.json

echo "verify attestation offline"
witness verify -k testpub.pem -p policy-signed.json -a k8s-att.json -f pods.json

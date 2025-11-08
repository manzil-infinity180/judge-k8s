#/bin/bash

set -e
set -x

rm out.tar | true
rm -rf ./tmp

mkdir ./tmp

IMAGE_NAME="8bd07846-7d97-4758-b7c5-7060d2471217"
docker tag manzilrahul/k8s-custom-controller@sha256:8ad350198b851f138a0fd234abfecf9f668a1e776e396e8fe906092316dbc346 ttl.sh/$IMAGE_NAME:5h
docker push ttl.sh/$IMAGE_NAME:5h
# rekorserver="http://172.23.0.3:30331"

# ip=`kubectl get svc rekor-server --template="{{range .status.loadBalancer.ingress}}{{.ip}}{{end}}"`
# port=`kubectl get svc rekor-server --template="{{range .spec.ports}}{{.nodePort}}{{end}}"`


#test
witness run -s=build -k testkey.pem -a oci -o attestation.json -- bash -c "docker save ttl.sh/$IMAGE_NAME:5h > ./tmp/out.tar"
echo "verify attestation offline"
witness verify -k testpub.pem -p policy-signed.json -a attestation.json -f ./tmp/out.tar

echo "waiting 10 seconds for rekor to be ready"
sleep 10

echo "verify attestation online"
witness verify -k testpub.pem -p policy-signed.json -f ./tmp/out.tar

echo "verify attestation in online with kubernetes"
kubectl -n=judge-test delete deploy test || true
kubectl -n=judge-test create deployment --image=ttl.sh/8bd07846-7d97-4758-b7c5-7060d2471217:5h test

kubectl -n=judge-test get deploy test -o=json | jq ".status.conditions[0].message"

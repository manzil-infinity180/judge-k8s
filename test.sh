kubectl delete deploy test09 || true
kubectl create deployment --image=testifysec/scratch@sha256:bc80d794049b44d65eaafec43780b5a6a4d9084b9f46d4e5b189db496c91e357 test09
IMAGE_NAME="8bd07846-7d97-4758-b7c5-7060d2471217"
# docker save ttl.sh/$IMAGE_NAME:5h
# kubectl delete deploy test774 -n judge-test || true &&
# kubectl create deployment --image=ttl.sh/$IMAGE_NAME:5h test77489 -n judge-test
# witness run -s save -k testkey.pem -a oci -o attestation.json -- bash -c "docker save ttl.sh/$IMAGE_NAME:5h > tout.tar"
# witness run -s save -k testkey.pem -a oci -o attestation.json -- bash -c "docker save ttl.sh/$IMAGE_NAME:5h > tout.tar"

witness run -s save -k testkey.pem -a oci -o attestation.json -- bash -c "docker save docker.io/testifysec/scratch@sha256:bc80d794049b44d65eaafec43780b5a6a4d9084b9f46d4e5b189db496c91e357 > t090out.tar"

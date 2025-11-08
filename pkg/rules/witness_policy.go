package rules

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	witness_dev "github.com/in-toto/go-witness"
	"github.com/in-toto/go-witness/cryptoutil"
	"github.com/in-toto/go-witness/dsse"
	"github.com/in-toto/go-witness/log"
	"github.com/regclient/regclient"
	"github.com/regclient/regclient/config"
	"github.com/regclient/regclient/types/manifest"
	"github.com/regclient/regclient/types/platform"
	"github.com/regclient/regclient/types/ref"

	// "github.com/sigstore/rekor/pkg/api"
	"github.com/testifysec/judge-k8s/cmd/options"

	// imported so their init functions run
	_ "github.com/in-toto/go-witness/attestation/aws-iid"
	_ "github.com/in-toto/go-witness/attestation/commandrun"
	_ "github.com/in-toto/go-witness/attestation/environment"
	_ "github.com/in-toto/go-witness/attestation/gcp-iit"
	_ "github.com/in-toto/go-witness/attestation/git"
	_ "github.com/in-toto/go-witness/attestation/gitlab"
	_ "github.com/in-toto/go-witness/attestation/jwt"
	_ "github.com/in-toto/go-witness/attestation/maven"
	_ "github.com/in-toto/go-witness/attestation/oci"
)

func init() {
	log.SetLogger(logger{})
}

type WitnessPolicy struct {
	Manifest manifest.Manifest
	// rekorServer string
	Envelopes []dsse.Envelope
	RegClient *regclient.RegClient
	Policy    []byte
	PublicKey []byte
}

func New(o *options.ServeOptions) (*WitnessPolicy, error) {

	wp := &WitnessPolicy{}

	f, err := os.Open(o.PublicKey)

	if err != nil {
		return nil, fmt.Errorf("failed to open public key file: %v", err)
	}
	defer f.Close()

	pubKeyB64Bytes, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %v", err)
	}

	pubKeyB64 := string(pubKeyB64Bytes)

	pubKey, err := base64.StdEncoding.DecodeString(pubKeyB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key file: %v", err)
	}

	wp.PublicKey = pubKey

	b, err := os.ReadFile(o.PolicyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read policy file: %v", err)
	}

	wp.Policy = b
	log.Info("Initializing Registry Client...")
	// wp.RegClient = regclient.NewRegClient()
	// wp.RegClient = regclient.New()
	wp.RegClient = regclient.New(
		regclient.WithConfigHost(config.Host{
			Name: "k3d-myregistry:5000",
			TLS:  config.TLSDisabled, // disables HTTPS
			// DNS:  []string{"k3d-myregistry"},
		}),
	)
	// wp.rekorServer = o.RekorServer
	return wp, nil
}

func (wp *WitnessPolicy) Verify(imageRef string) ([]string, error) {
	log.Info("witness policy verify")
	m, err := wp.getManifest(imageRef)
	if err != nil {
		fmt.Printf("failed to get manifest: %v\n", err)
		return nil, fmt.Errorf("failed to get manifest: %v", err)
	}

	digestx, err := m.GetConfigDigest()
	if err != nil {
		fmt.Printf("failed to get config digest: %v\n", err)
		return nil, fmt.Errorf("failed to get config digest: %v", err)
	}
	log.Info(digestx.String())

	// err = wp.getRekorEntries(configDigest.String())
	// if err != nil {
	// 	fmt.Printf("failed to get rekor entries: %v=n", err)
	// 	return nil, fmt.Errorf("failed to get rekor entries: %v", err)
	// }
	err = wp.doesPassWitnessPolicy()
	if err != nil {
		fmt.Printf("failed to pass witness policy: %v\n", err)
		return []string{}, err
	}
	log.Info("witness policy: all checked passed")
	//Passes all checks
	return []string{}, nil

}

func (wp *WitnessPolicy) getManifest(imageRef string) (manifest.Manifest, error) {
	ctx := context.Background()
	log.Info("into the getManifest!")
	// Replace localhost with cluster registry
	imageRef = strings.Replace(imageRef, "localhost:34751", "k3d-myregistry:5000", 1)

	r, err := ref.New(imageRef)
	if err != nil {
		return nil, fmt.Errorf("failed to create ref: %v", err)
	}
	fmt.Println(r.Digest)
	fmt.Println(r.Repository)
	fmt.Println(r)

	manifest, err := wp.RegClient.ManifestGet(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("failed to get manifest: %v", err)
	}

	if manifest.IsList() {

		// plat := ociv1.Platform{
		// 	Architecture: "amd64",
		// 	OS:           "linux",
		// }
		plat := platform.Platform{
			Architecture: "amd64",
			OS:           "linux",
		}

		desc, err := manifest.GetPlatformDesc(&plat)
		if err != nil {
			return nil, err
		}
		fmt.Println(desc.ArtifactType)
		fmt.Println(desc.URLs)
		fmt.Println(desc.Digest.String())
		fmt.Println(desc)

		r.Digest = desc.Digest.String()
		manifest, err = wp.RegClient.ManifestGet(ctx, r)
		if err != nil {
			return nil, err
		}
	}
	log.Info("returning from getManifest")
	return manifest, nil
}

// func (wp *WitnessPolicy) getRekorEntries(containerID string) error {
// 	ds := cryptoutil.DigestSet{}
// 	containerID = strings.Replace(containerID, "sha256:", "", -1)
// 	ds[crypto.SHA256] = containerID
// 	fmt.Printf("looking up rekor entry for container id: %v", containerID)

// 	entries, err := loadEnvelopesFromRekor(wp.rekorServer, ds)

// 	if err != nil {
// 		return fmt.Errorf("failed to get rekor entries: %v", err)
// 	}

// 	if len(entries) == 0 {
// 		fmt.Printf("No entries found for ContainerID: %s", containerID)
// 		return fmt.Errorf("no entries found")
// 	}

// 	wp.Envelopes = entries

// 	return nil

// }

func (wp *WitnessPolicy) doesPassWitnessPolicy() error {
	log.Info("into the doesPassWitnessPolicy")
	policyEnvelope := dsse.Envelope{}
	err := json.Unmarshal(wp.Policy, &policyEnvelope)

	if err != nil {
		return fmt.Errorf("failed to unmarshal policy: %v", err)
	}

	pubKeyReader := strings.NewReader(string(wp.PublicKey))
	log.Info("pubKeyReader")
	verifier, err := cryptoutil.NewVerifierFromReader(pubKeyReader)
	if err != nil {
		return fmt.Errorf("failed to load key: %v", err)
	}
	// veropt := witness_dev.VerifyWithCollectionEnvelopes(wp.Envelopes)
	// spew.Dump(veropt)
	ctx := context.Background()
	log.Info("witness_dev.Verify")
	reason, err := witness_dev.Verify(ctx, policyEnvelope, []cryptoutil.Verifier{verifier})
	if err != nil {

		return fmt.Errorf("policy failed to verify: %v %v", reason, err)
	}

	keyid, err := verifier.KeyID()
	if err != nil {
		return fmt.Errorf("failed to get key id: %v", err)
	}

	fmt.Printf("Keyid: %v\n", keyid)
	fmt.Printf("PolicyKeyid: %v\n", policyEnvelope.Signatures[0].KeyID)
	fmt.Printf("attestation keyid: %v\n", wp.Envelopes[0].Signatures[0].KeyID)

	fmt.Printf("policy bytes:\n%s\n", wp.Policy)
	fmt.Printf("public key:\n%s\n", wp.PublicKey)
	for _, e := range wp.Envelopes {
		out, err := json.Marshal(e)
		if err != nil {
			fmt.Printf("failed to marshal envelope: %v", err)
		}

		fmt.Printf("envelope:\n %s\n", out)
	}
	// https://github.com/in-toto/go-witness/blob/c0c02fa4fa1d7884f3438b9ce5774cc698939872/policy/policy.go#L192
	/**
	 [judge-k8s-webhook] failed to pass witness policy: policy failed to verify: {{{ []} {[]  []} } {{} 0001-01-01 00:00:00 +0000 UTC { map[]} [] } map[]} attestors failed with error messages
	[judge-k8s-webhook] attestor policyverify failed: failed to verify policy: invalid option (subject digests): at least one subject digest is required
	*/
	/***
	 - judge-test:deployment/judge-k8s-webhook: replica set creation failed: following errors occurred ReplicaFailureAdmissionErr: admission webhook "judge-k8s-webhook.judge-test.svc" denied the request: policy failed to verify: {{{ []} {[]  []} } {{} 0001-01-01 00:00:00 +0000 UTC { map[]} [] } map[]} attestors failed with error messages
	attestor policyverify failed: failed to verify policy: invalid option (subject digests): at least one subject digest is required
	 - judge-test:deployment/judge-k8s-webhook failed. Error: replica set creation failed: following errors occurred ReplicaFailureAdmissionErr: admission webhook "judge-k8s-webhook.judge-test.svc" denied the request: policy failed to verify: {{{ []} {[]  []} } {{} 0001-01-01 00:00:00 +0000 UTC { map[]} [] } map[]} attestors failed with error messages
	attestor policyverify failed: failed to verify policy: invalid option (subject digests): at least one subject digest is required.
	*/
	return fmt.Errorf("policy failed to verify: %v", reason)
}

// func loadEnvelopesFromRekor(rekorServer string, artifactDigestSet cryptoutil.DigestSet) ([]dsse.Envelope, error) {
// 	envelopes := make([]dsse.Envelope, 0)
// 	rc, err := rekor.New(rekorServer)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get initialize Rekor client: %w", err)
// 	}

// 	entries, err := rc.FindEntriesBySubject(artifactDigestSet)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to find any entries in rekor: %w", err)
// 	}

// 	for _, entry := range entries {
// 		env, err := rekor.ParseEnvelopeFromEntry(entry)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to parse dsse envelope from rekor entry: %w", err)
// 		}

// 		envelopes = append(envelopes, env)
// 	}

// 	return envelopes, nil
// }

type logger struct{}

func (logger) Errorf(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...))
}

func (logger) Error(args ...interface{}) {
	fmt.Println(args...)
}

func (logger) Warnf(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...))
}

func (logger) Warn(args ...interface{}) {
	fmt.Println(args...)
}

func (logger) Debugf(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...))
}

func (logger) Debug(args ...interface{}) {
	fmt.Println(args...)
}

func (logger) Infof(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...))
}

func (logger) Info(args ...interface{}) {
	fmt.Println(args...)
}

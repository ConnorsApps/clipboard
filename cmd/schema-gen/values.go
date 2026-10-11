package main

import (
	jsonschema "github.com/swaggest/jsonschema-go"
	corev1 "k8s.io/api/core/v1"
)

// These types mirror charts/clipboard/values.yaml by hand. The schema rejects
// unknown keys on the chart's own objects, so a key added to values.yaml
// without a field here fails `helm lint`.

// pullPolicy: "" renders as null, which leaves the field to the cluster default.
type pullPolicy string

func (pullPolicy) Enum() []any {
	return []any{"", string(corev1.PullAlways), string(corev1.PullIfNotPresent), string(corev1.PullNever)}
}

// serviceType: "" renders as null, which Kubernetes defaults to ClusterIP.
type serviceType string

func (serviceType) Enum() []any {
	return []any{
		"", string(corev1.ServiceTypeClusterIP), string(corev1.ServiceTypeNodePort),
		string(corev1.ServiceTypeLoadBalancer), string(corev1.ServiceTypeExternalName),
	}
}

// imageTag is a string, but a CI-supplied `--set image.tag=1234` parses as an
// integer and the template prints it as is, so both must keep working. A float
// stays rejected: 1.10 would render as 1.1.
type imageTag string

func (imageTag) PrepareJSONSchema(s *jsonschema.Schema) error {
	s.Type = &jsonschema.Type{SliceOfSimpleTypeValues: []jsonschema.SimpleType{jsonschema.String, jsonschema.Integer}}
	s.WithMinLength(1)
	return nil
}

type accessMode string

func (accessMode) Enum() []any {
	return []any{
		string(corev1.ReadWriteOnce), string(corev1.ReadOnlyMany),
		string(corev1.ReadWriteMany), string(corev1.ReadWriteOncePod),
	}
}

// secret is a string, but `--set config.passwords=1234` parses as a number and
// the template quotes it, so both must keep working.
type secret string

func (secret) PrepareJSONSchema(s *jsonschema.Schema) error {
	s.Type = &jsonschema.Type{SliceOfSimpleTypeValues: []jsonschema.SimpleType{jsonschema.String, jsonschema.Number}}
	return nil
}

// mongoURI is what the MongoDB driver accepts; "" keeps tokens in memory.
type mongoURI string

const mongoURIPattern = `^(mongodb(\+srv)?://.+)?$`

func (mongoURI) PrepareJSONSchema(s *jsonschema.Schema) error {
	s.WithPattern(mongoURIPattern)
	return nil
}

// tokenExpiry is what the server's ParseTokenExpiry (internal/config) accepts:
// "never", or a positive number of days with a "d" suffix. "" uses the app
// default.
type tokenExpiry string

const tokenExpiryPattern = `^(\s*\+?0*[1-9][0-9]*[dD]\s*|[Nn][Ee][Vv][Ee][Rr])?$`

func (tokenExpiry) PrepareJSONSchema(s *jsonschema.Schema) error {
	s.WithPattern(tokenExpiryPattern)
	return nil
}

type imageValues struct {
	Repository string     `json:"repository" minLength:"1" description:"Image repository"`
	Tag        imageTag   `json:"tag" description:"Image tag. Only :latest is published."`
	PullPolicy pullPolicy `json:"pullPolicy" description:"Image pull policy. Defaults to Always because only :latest is published. Empty: the cluster default."`
}

type configValues struct {
	Passwords   secret      `json:"passwords" description:"Comma-separated; each password is its own user. In --set, escape the comma (one\\,two). Required unless existingSecret is set: the app would otherwise accept 1234."`
	MongoDBURI  mongoURI    `json:"mongodbUri" description:"MongoDB URI (mongodb:// or mongodb+srv://) that keeps logins across restarts. Empty: in-memory, so everyone logs in again after a restart."`
	TokenExpiry tokenExpiry `json:"tokenExpiry" description:"Login token lifetime: a number of days such as 10d, or never. Empty: the app default (30d). MongoDB only."`
}

type serviceValues struct {
	Type        serviceType       `json:"type" description:"Kubernetes Service type. Empty: ClusterIP."`
	Port        int               `json:"port" minimum:"1" maximum:"65535" description:"Service port. The container always listens on 8080."`
	Annotations map[string]string `json:"annotations" description:"Service annotations"`
}

type parentRef struct {
	Name        string `json:"name" description:"Gateway name. Required when httpRoute.enabled."`
	Namespace   string `json:"namespace" description:"Gateway namespace. Empty: the release namespace."`
	SectionName string `json:"sectionName" description:"Listener name on the Gateway. Empty: all listeners."`
}

type httpRouteValues struct {
	Enabled     bool              `json:"enabled" description:"Render a Gateway API HTTPRoute for the Service. Needs the Gateway API CRDs; the Gateway's timeouts must allow long-lived WebSockets (/ws) and large uploads."`
	Annotations map[string]string `json:"annotations" description:"HTTPRoute annotations"`
	ParentRef   parentRef         `json:"parentRef" description:"The Gateway the route attaches to"`
	Hostnames   []string          `json:"hostnames" description:"Hostnames the route answers for"`
}

type persistenceValues struct {
	Enabled       bool              `json:"enabled" description:"false: an emptyDir, so uploaded files are lost when the pod is rescheduled"`
	ExistingClaim string            `json:"existingClaim" description:"Use this PersistentVolumeClaim instead of creating one"`
	StorageClass  string            `json:"storageClass" description:"StorageClass name. Empty: the cluster default."`
	AccessModes   []accessMode      `json:"accessModes" description:"PVC access modes. The Deployment runs one replica with the Recreate strategy, so ReadWriteOnce is enough."`
	Size          quantity          `json:"size" description:"Requested volume size, e.g. 10Gi"`
	Annotations   map[string]string `json:"annotations" description:"PVC annotations"`
}

type values struct {
	Global             map[string]any                `json:"global" description:"Values shared with parent charts. Helm adds this key to every subchart, so it must be allowed; the chart reads none of it."`
	NameOverride       string                        `json:"nameOverride" pattern:"^([a-z0-9]([-a-z0-9]*[a-z0-9])?)?$" description:"Override the chart name in resource names"`
	FullnameOverride   string                        `json:"fullnameOverride" pattern:"^([a-z0-9]([-a-z0-9]*[a-z0-9])?)?$" description:"Override the fully qualified resource name. Changing it on an existing release renames the files PVC."`
	CommonLabels       map[string]string             `json:"commonLabels" description:"Labels added to every object this chart renders"`
	Image              imageValues                   `json:"image"`
	ImagePullSecrets   []corev1.LocalObjectReference `json:"imagePullSecrets" description:"Secrets used to pull the image"`
	Config             configValues                  `json:"config" description:"Application settings, rendered into the chart's Secret"`
	ExistingSecret     string                        `json:"existingSecret" description:"Name of a Secret with CLIPBOARD_PASSWORDS and optionally MONGODB_URI, used instead of the chart's own. Then config.passwords and config.mongodbUri are ignored."`
	ExtraEnv           []corev1.EnvVar               `json:"extraEnv" description:"Extra container env vars"`
	Service            serviceValues                 `json:"service"`
	HTTPRoute          httpRouteValues               `json:"httpRoute" description:"Optional Gateway API HTTPRoute"`
	Persistence        persistenceValues             `json:"persistence" description:"Uploaded files, mounted at /data"`
	Resources          corev1.ResourceRequirements   `json:"resources" description:"Container resources. Maps merge; drop a default with null (--set resources.limits=null)."`
	PodAnnotations     map[string]string             `json:"podAnnotations" description:"Extra pod annotations"`
	PodLabels          map[string]string             `json:"podLabels" description:"Extra pod labels"`
	PodSecurityContext corev1.PodSecurityContext     `json:"podSecurityContext" description:"Pod-level security context"`
	SecurityContext    corev1.SecurityContext        `json:"securityContext" description:"Container security context"`
	NodeSelector       map[string]string             `json:"nodeSelector" description:"Pod node selector"`
	Tolerations        []corev1.Toleration           `json:"tolerations" description:"Pod tolerations"`
	Affinity           corev1.Affinity               `json:"affinity" description:"Pod affinity rules"`
}

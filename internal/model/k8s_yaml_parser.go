package model

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"io"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"reflect"
	"testing"
)

func splitYAML(resources []byte) ([][]byte, error) {

	dec := yaml.NewDecoder(bytes.NewReader(resources))

	var res [][]byte
	for {
		var value interface{}
		err := dec.Decode(&value)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		valueBytes, err := yaml.Marshal(value)
		if err != nil {
			return nil, err
		}
		res = append(res, valueBytes)
	}
	return res, nil
}

func NewK8sResources(objects []runtime.Object, schemas []schema.GroupVersionKind) *K8sResources {
	if objects == nil {
		objects = []runtime.Object{}
	}
	manifest := K8sResources{objects, schemas}
	return &manifest
}

type K8sResources struct {
	objects []runtime.Object
	schemas []schema.GroupVersionKind
}

func (k *K8sResources) Add(i runtime.Object, more ...runtime.Object) {
	expectedKind := i.GetObjectKind().GroupVersionKind()

	if k.checkSchema(expectedKind) {
		k.objects = append(k.objects, i)
	}
	for _, o := range more {
		if expectedKind == o.GetObjectKind().GroupVersionKind() {
			k.objects = append(k.objects, o)
		} else {
			k.Add(o)
			expectedKind = o.GetObjectKind().GroupVersionKind()
		}
	}
}

func (k *K8sResources) checkSchema(expectedKind schema.GroupVersionKind) bool {
	for _, schema := range k.schemas {
		if schema == expectedKind {
			return true
		}
	}
	panic(expectedKind.String() + " missing from schemas")
}

func (k *K8sResources) Only(t *testing.T, i runtime.Object) runtime.Object {

	var match runtime.Object = nil
	expectedType := reflect.TypeOf(i)
	for _, o := range k.objects {
		if expectedType == reflect.TypeOf(o) {
			if match == nil {
				match = o
			} else {
				assert.Failf(t, "Multiple %s items found in manifest. Expecting only one.", expectedType.String())
			}
		}
	}
	assert.NotNil(t, match)
	return match
}

func (k *K8sResources) First(i runtime.Object) runtime.Object {
	expectedType := reflect.TypeOf(i)
	for _, o := range k.objects {
		if expectedType == reflect.TypeOf(o) {
			return o
		}
	}
	return nil
}

func (k *K8sResources) OfTypeWithName(i metav1.Object, name string) metav1.Object {
	expectedType := reflect.TypeOf(i)
	for _, o := range k.objects {
		if expectedType == reflect.TypeOf(o) {
			asMetaObject := o.(metav1.Object)
			if asMetaObject.GetName() == name {
				return asMetaObject
			}
		}
	}
	return nil
}

func (k *K8sResources) All() (out []runtime.Object) {

	return k.objects
}

func (k *K8sResources) AllWithMetadata() (out []metav1.Object) {

	for _, o := range k.objects {
		meta, ok := o.(metav1.Object)
		if ok {
			out = append(out, meta)
		}
	}
	return out
}

func (k *K8sResources) OfType(i runtime.Object) (out []runtime.Object) {
	expectedType := reflect.TypeOf(i)
	for _, o := range k.objects {
		if expectedType == reflect.TypeOf(o) {
			out = append(out, o)
		}
	}
	return out
}

func (k *K8sResources) AddPods(pods []v1.Pod) {
	N := len(pods)

	if N == 0 {
		return
	}

	if N == 1 {
		k.Add(&pods[0])
		return
	}

	obj := make([]runtime.Object, N-1)

	for i := 1; i < N; i++ {
		obj[i-1] = &pods[i]
	}

	k.Add(&pods[0], obj...)
}

func (k *K8sResources) AddServices(services []v1.Service) {
	N := len(services)

	if N == 0 {
		return
	}

	if N == 1 {
		k.Add(&services[0])
		return
	}

	obj := make([]runtime.Object, N-1)

	for i := 1; i < N; i++ {
		obj[i-1] = &services[i]
	}

	k.Add(&services[0], obj...)
}

func (k *K8sResources) AddEndpoints(items []v1.Endpoints) {
	N := len(items)

	if N == 0 {
		return
	}

	if N == 1 {
		k.Add(&items[0])
		return
	}

	obj := make([]runtime.Object, N-1)

	for i := 1; i < N; i++ {
		obj[i-1] = &items[i]
	}

	k.Add(&items[0], obj...)
}

func decodeK8s(in []byte) (*K8sResources, error) {
	yamls, err := splitYAML(in)
	if err != nil {
		return nil, err
	}

	objects := make([]runtime.Object, len(yamls))
	schemas := make([]schema.GroupVersionKind, len(yamls))
	for i, yaml := range yamls {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, schema, err := decode(yaml, nil, nil)
		if err != nil {
			return nil, err
		}
		objects[i] = obj
		schemas[i] = *schema
	}

	return &K8sResources{objects, schemas}, err
}

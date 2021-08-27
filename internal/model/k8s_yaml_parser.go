package model

import (
	"bytes"
	"gopkg.in/yaml.v2"
	"io"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"reflect"
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

type K8sResources struct {
	objects []runtime.Object
	schemas []schema.GroupVersionKind
}

func (k *K8sResources) First(i runtime.Object) runtime.Object {

	for _, o := range k.objects {
		if reflect.TypeOf(i) == reflect.TypeOf(o) {
			return o
		}
	}
	return nil
}

func (k *K8sResources) OfTypeWithName(i metav1.Object, name string) metav1.Object {

	for _, o := range k.objects {
		if reflect.TypeOf(i) == reflect.TypeOf(o) {
			asMetaObject := o.(metav1.Object)
			if asMetaObject.GetName() == name {
				return asMetaObject
			}
		}
	}
	return nil
}

func (k *K8sResources) OfType(i runtime.Object) (out []runtime.Object) {

	for _, o := range k.objects {
		if reflect.TypeOf(i) == reflect.TypeOf(o) {
			out = append(out, o)
		}
	}
	return out
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

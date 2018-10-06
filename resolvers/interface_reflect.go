// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Most of this file was reused from the ecoding/json package,
// since we want to use the same json encoding.
package resolvers

import (
	"context"
	"github.com/graph-gophers/graphql-go/internal/common"
	"reflect"
	"strings"
)

var childMethodTypeCache common.Cache

type methodInfo struct {
	Index               int
	hasContext          bool
	hasExecutionContext bool
	argumentsType       *reflect.Type
	hasError            bool
}

func getChildMethod(parent *reflect.Value, fieldName string) *methodInfo {

	var key struct{
		parentType reflect.Type
		fieldName string
	}
	key.parentType = parent.Type()
	key.fieldName = fieldName

	// use a cache to make subsequent lookups cheap
	method := childMethodTypeCache.GetOrElseUpdate(key, func() interface{} {
		methods := typeMethods(key.parentType)
		return methods[strings.Replace(strings.ToLower(fieldName), "_", "", -1)]
	}).(*methodInfo)

	return method
}

var contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
var executionContextType = reflect.TypeOf((*ExecutionContext)(nil)).Elem()
var errorType = reflect.TypeOf((*error)(nil)).Elem()

func typeMethods(t reflect.Type) map[string]*methodInfo {
	methods := map[string]*methodInfo{}
	for i := 0; i < t.NumMethod(); i++ {
		methodInfo := methodInfo{}
		methodInfo.Index = i
		typeMethod := t.Method(i)

		in := make([]reflect.Type, typeMethod.Type.NumIn())
		for i := range in {
			in[i] = typeMethod.Type.In(i)
		}

		methodHasReceiver := unwrapIfPtr(t).Kind() != reflect.Interface
		if methodHasReceiver {
			in = in[1:] // first parameter is receiver
		}

		methodInfo.hasContext = len(in) > 0 && in[0] == contextType
		if methodInfo.hasContext {
			in = in[1:]
		}

		methodInfo.hasExecutionContext = len(in) > 0 && in[0] == executionContextType
		if methodInfo.hasExecutionContext {
			in = in[1:]
		}

		if len(in) > 0 && ( in[0].Kind() == reflect.Struct || ( in[0].Kind() == reflect.Ptr && in[0].Elem().Kind()== reflect.Struct )) {
			methodInfo.argumentsType = &in[0]
			in = in[1:]
		}

		if len(in) > 0 {
			continue
		}

		if typeMethod.Type.NumOut() > 2 {
			continue
		}

		methodInfo.hasError = typeMethod.Type.NumOut() == 2
		if methodInfo.hasError {
			if typeMethod.Type.Out(1) != errorType {
				continue;
			}
		}
		methods[strings.ToLower(typeMethod.Name)] = &methodInfo
	}

	return methods
}
func unwrapIfPtr(t reflect.Type) reflect.Type {
	if (t.Kind() == reflect.Ptr) {
		return t.Elem()
	}
	return t;
}



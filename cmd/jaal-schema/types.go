package main

import "fmt"

type IntrospectionResponse struct {
	Schema IntrospectionSchema `json:"__schema"`
}

type IntrospectionSchema struct {
	QueryType        *IntrospectionType      `json:"queryType"`
	MutationType     *IntrospectionType      `json:"mutationType"`
	SubscriptionType *IntrospectionType      `json:"subscriptionType"`
	Types            []IntrospectionFullType `json:"types"`
	Directives       []IntrospectionDirective `json:"directives"`
}

type IntrospectionType struct {
	Name string `json:"name"`
}

type IntrospectionFullType struct {
	Kind          string                   `json:"kind"`
	Name          string                   `json:"name"`
	Description   string                   `json:"description"`
	Fields        []IntrospectionField     `json:"fields"`
	InputFields   []IntrospectionInputValue `json:"inputFields"`
	Interfaces    []IntrospectionTypeRef   `json:"interfaces"`
	EnumValues    []IntrospectionEnumValue `json:"enumValues"`
	PossibleTypes []IntrospectionTypeRef   `json:"possibleTypes"`
	Directives    []IntrospectionDirective `json:"directives"`
	SpecifiedBy   *string                  `json:"specifiedByURL"`
}

type IntrospectionField struct {
	Name              string                   `json:"name"`
	Description       string                   `json:"description"`
	Args              []IntrospectionInputValue `json:"args"`
	Type              IntrospectionTypeRef     `json:"type"`
	IsDeprecated      bool                     `json:"isDeprecated"`
	DeprecationReason string                   `json:"deprecationReason"`
}

type IntrospectionInputValue struct {
	Name              string               `json:"name"`
	Description       string               `json:"description"`
	Type              IntrospectionTypeRef `json:"type"`
	DefaultValue      *string              `json:"defaultValue"`
	IsDeprecated      bool                 `json:"isDeprecated"`
	DeprecationReason string               `json:"deprecationReason"`
}

type IntrospectionEnumValue struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	IsDeprecated      bool   `json:"isDeprecated"`
	DeprecationReason string `json:"deprecationReason"`
}

type IntrospectionTypeRef struct {
	Kind   string                `json:"kind"`
	Name   *string               `json:"name"`
	OfType *IntrospectionTypeRef `json:"ofType"`
}

type IntrospectionDirective struct {
	Name        string                   `json:"name"`
	Description string                   `json:"description"`
	Locations   []string                 `json:"locations"`
	Args        []IntrospectionInputValue `json:"args"`
}

func (tr IntrospectionTypeRef) String() string {
	if tr.Kind == "NON_NULL" {
		if tr.OfType == nil {
			return "!"
		}
		return fmt.Sprintf("%s!", tr.OfType.String())
	}
	if tr.Kind == "LIST" {
		if tr.OfType == nil {
			return "[]"
		}
		return fmt.Sprintf("[%s]", tr.OfType.String())
	}
	if tr.Name != nil {
		return *tr.Name
	}
	return ""
}

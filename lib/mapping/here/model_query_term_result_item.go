/*
 * Geocoding and Search API v7
 *
 * This document describes the Geocoding and Search API.
 *
 * API version: 7.78
 */

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package here

import (
	"encoding/json"
)

// QueryTermResultItem struct for QueryTermResultItem
type QueryTermResultItem struct {
	// The term that will be suggested to the user.
	Term string `json:"term"`
	// The sub-string of the original query that is replaced by this Query Term.
	Replaces string `json:"replaces"`
	// The start index in codepoints (inclusive) of the text replaced in the original query.
	Start int32 `json:"start"`
	// The end index in codepoints (exclusive) of the text replaced in the original query.
	End int32 `json:"end"`
}

// NewQueryTermResultItem instantiates a new QueryTermResultItem object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewQueryTermResultItem(term string, replaces string, start int32, end int32, ) *QueryTermResultItem {
	this := QueryTermResultItem{}
	this.Term = term
	this.Replaces = replaces
	this.Start = start
	this.End = end
	return &this
}

// NewQueryTermResultItemWithDefaults instantiates a new QueryTermResultItem object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewQueryTermResultItemWithDefaults() *QueryTermResultItem {
	this := QueryTermResultItem{}
	return &this
}

// GetTerm returns the Term field value
func (o *QueryTermResultItem) GetTerm() string {
	if o == nil  {
		var ret string
		return ret
	}

	return o.Term
}

// GetTermOk returns a tuple with the Term field value
// and a boolean to check if the value has been set.
func (o *QueryTermResultItem) GetTermOk() (*string, bool) {
	if o == nil  {
		return nil, false
	}
	return &o.Term, true
}

// SetTerm sets field value
func (o *QueryTermResultItem) SetTerm(v string) {
	o.Term = v
}

// GetReplaces returns the Replaces field value
func (o *QueryTermResultItem) GetReplaces() string {
	if o == nil  {
		var ret string
		return ret
	}

	return o.Replaces
}

// GetReplacesOk returns a tuple with the Replaces field value
// and a boolean to check if the value has been set.
func (o *QueryTermResultItem) GetReplacesOk() (*string, bool) {
	if o == nil  {
		return nil, false
	}
	return &o.Replaces, true
}

// SetReplaces sets field value
func (o *QueryTermResultItem) SetReplaces(v string) {
	o.Replaces = v
}

// GetStart returns the Start field value
func (o *QueryTermResultItem) GetStart() int32 {
	if o == nil  {
		var ret int32
		return ret
	}

	return o.Start
}

// GetStartOk returns a tuple with the Start field value
// and a boolean to check if the value has been set.
func (o *QueryTermResultItem) GetStartOk() (*int32, bool) {
	if o == nil  {
		return nil, false
	}
	return &o.Start, true
}

// SetStart sets field value
func (o *QueryTermResultItem) SetStart(v int32) {
	o.Start = v
}

// GetEnd returns the End field value
func (o *QueryTermResultItem) GetEnd() int32 {
	if o == nil  {
		var ret int32
		return ret
	}

	return o.End
}

// GetEndOk returns a tuple with the End field value
// and a boolean to check if the value has been set.
func (o *QueryTermResultItem) GetEndOk() (*int32, bool) {
	if o == nil  {
		return nil, false
	}
	return &o.End, true
}

// SetEnd sets field value
func (o *QueryTermResultItem) SetEnd(v int32) {
	o.End = v
}

func (o QueryTermResultItem) MarshalJSON() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["term"] = o.Term
	}
	if true {
		toSerialize["replaces"] = o.Replaces
	}
	if true {
		toSerialize["start"] = o.Start
	}
	if true {
		toSerialize["end"] = o.End
	}
	return json.Marshal(toSerialize)
}

type NullableQueryTermResultItem struct {
	value *QueryTermResultItem
	isSet bool
}

func (v NullableQueryTermResultItem) Get() *QueryTermResultItem {
	return v.value
}

func (v *NullableQueryTermResultItem) Set(val *QueryTermResultItem) {
	v.value = val
	v.isSet = true
}

func (v NullableQueryTermResultItem) IsSet() bool {
	return v.isSet
}

func (v *NullableQueryTermResultItem) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableQueryTermResultItem(val *QueryTermResultItem) *NullableQueryTermResultItem {
	return &NullableQueryTermResultItem{value: val, isSet: true}
}

func (v NullableQueryTermResultItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableQueryTermResultItem) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


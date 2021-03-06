// Copyright 2015 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bigquery

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	bq "google.golang.org/api/bigquery/v2"
)

func TestConvertBasicValues(t *testing.T) {
	schema := []*FieldSchema{
		{Type: StringFieldType},
		{Type: IntegerFieldType},
		{Type: FloatFieldType},
		{Type: BooleanFieldType},
	}
	row := &bq.TableRow{
		F: []*bq.TableCell{
			{V: "a"},
			{V: "1"},
			{V: "1.2"},
			{V: "true"},
		},
	}
	got, err := convertRow(row, schema)
	if err != nil {
		t.Fatalf("error converting: %v", err)
	}
	want := []Value{"a", int64(1), 1.2, true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("converting basic values: got:\n%v\nwant:\n%v", got, want)
	}
}

func TestConvertTime(t *testing.T) {
	schema := []*FieldSchema{
		{Type: TimestampFieldType},
	}
	thyme := time.Date(1970, 1, 1, 10, 0, 0, 10, time.UTC)
	row := &bq.TableRow{
		F: []*bq.TableCell{
			{V: fmt.Sprintf("%.10f", float64(thyme.UnixNano())/1e9)},
		},
	}
	got, err := convertRow(row, schema)
	if err != nil {
		t.Fatalf("error converting: %v", err)
	}
	if !got[0].(time.Time).Equal(thyme) {
		t.Errorf("converting basic values: got:\n%v\nwant:\n%v", got, thyme)
	}
}

func TestConvertNullValues(t *testing.T) {
	schema := []*FieldSchema{
		{Type: StringFieldType},
	}
	row := &bq.TableRow{
		F: []*bq.TableCell{
			{V: nil},
		},
	}
	got, err := convertRow(row, schema)
	if err != nil {
		t.Fatalf("error converting: %v", err)
	}
	want := []Value{nil}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("converting null values: got:\n%v\nwant:\n%v", got, want)
	}
}

func TestBasicRepetition(t *testing.T) {
	schema := []*FieldSchema{
		{Type: IntegerFieldType, Repeated: true},
	}
	row := &bq.TableRow{
		F: []*bq.TableCell{
			{
				V: []interface{}{
					map[string]interface{}{
						"v": "1",
					},
					map[string]interface{}{
						"v": "2",
					},
					map[string]interface{}{
						"v": "3",
					},
				},
			},
		},
	}
	got, err := convertRow(row, schema)
	if err != nil {
		t.Fatalf("error converting: %v", err)
	}
	want := []Value{[]Value{int64(1), int64(2), int64(3)}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("converting basic repeated values: got:\n%v\nwant:\n%v", got, want)
	}
}

func TestNestedRecordContainingRepetition(t *testing.T) {
	schema := []*FieldSchema{
		{
			Type: RecordFieldType,
			Schema: Schema{
				{Type: IntegerFieldType, Repeated: true},
			},
		},
	}
	row := &bq.TableRow{
		F: []*bq.TableCell{
			{
				V: map[string]interface{}{
					"f": []interface{}{
						map[string]interface{}{
							"v": []interface{}{
								map[string]interface{}{"v": "1"},
								map[string]interface{}{"v": "2"},
								map[string]interface{}{"v": "3"},
							},
						},
					},
				},
			},
		},
	}

	got, err := convertRow(row, schema)
	if err != nil {
		t.Fatalf("error converting: %v", err)
	}
	want := []Value{[]Value{[]Value{int64(1), int64(2), int64(3)}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("converting basic repeated values: got:\n%v\nwant:\n%v", got, want)
	}
}

func TestRepeatedRecordContainingRepetition(t *testing.T) {
	schema := []*FieldSchema{
		{
			Type:     RecordFieldType,
			Repeated: true,
			Schema: Schema{
				{Type: IntegerFieldType, Repeated: true},
			},
		},
	}
	row := &bq.TableRow{F: []*bq.TableCell{
		{
			V: []interface{}{ // repeated records.
				map[string]interface{}{ // first record.
					"v": map[string]interface{}{ // pointless single-key-map wrapper.
						"f": []interface{}{ // list of record fields.
							map[string]interface{}{ // only record (repeated ints)
								"v": []interface{}{ // pointless wrapper.
									map[string]interface{}{
										"v": "1",
									},
									map[string]interface{}{
										"v": "2",
									},
									map[string]interface{}{
										"v": "3",
									},
								},
							},
						},
					},
				},
				map[string]interface{}{ // second record.
					"v": map[string]interface{}{
						"f": []interface{}{
							map[string]interface{}{
								"v": []interface{}{
									map[string]interface{}{
										"v": "4",
									},
									map[string]interface{}{
										"v": "5",
									},
									map[string]interface{}{
										"v": "6",
									},
								},
							},
						},
					},
				},
			},
		},
	}}

	got, err := convertRow(row, schema)
	if err != nil {
		t.Fatalf("error converting: %v", err)
	}
	want := []Value{ // the row is a list of length 1, containing an entry for the repeated record.
		[]Value{ // the repeated record is a list of length 2, containing an entry for each repetition.
			[]Value{ // the record is a list of length 1, containing an entry for the repeated integer field.
				[]Value{int64(1), int64(2), int64(3)}, // the repeated integer field is a list of length 3.
			},
			[]Value{ // second record
				[]Value{int64(4), int64(5), int64(6)},
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("converting repeated records with repeated values: got:\n%v\nwant:\n%v", got, want)
	}
}

func TestRepeatedRecordContainingRecord(t *testing.T) {
	schema := []*FieldSchema{
		{
			Type:     RecordFieldType,
			Repeated: true,
			Schema: Schema{
				{
					Type: StringFieldType,
				},
				{
					Type: RecordFieldType,
					Schema: Schema{
						{Type: IntegerFieldType},
						{Type: StringFieldType},
					},
				},
			},
		},
	}
	row := &bq.TableRow{F: []*bq.TableCell{
		{
			V: []interface{}{ // repeated records.
				map[string]interface{}{ // first record.
					"v": map[string]interface{}{ // pointless single-key-map wrapper.
						"f": []interface{}{ // list of record fields.
							map[string]interface{}{ // first record field (name)
								"v": "first repeated record",
							},
							map[string]interface{}{ // second record field (nested record).
								"v": map[string]interface{}{ // pointless single-key-map wrapper.
									"f": []interface{}{ // nested record fields
										map[string]interface{}{
											"v": "1",
										},
										map[string]interface{}{
											"v": "two",
										},
									},
								},
							},
						},
					},
				},
				map[string]interface{}{ // second record.
					"v": map[string]interface{}{
						"f": []interface{}{
							map[string]interface{}{
								"v": "second repeated record",
							},
							map[string]interface{}{
								"v": map[string]interface{}{
									"f": []interface{}{
										map[string]interface{}{
											"v": "3",
										},
										map[string]interface{}{
											"v": "four",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}}

	got, err := convertRow(row, schema)
	if err != nil {
		t.Fatalf("error converting: %v", err)
	}
	// TODO: test with flattenresults.
	want := []Value{ // the row is a list of length 1, containing an entry for the repeated record.
		[]Value{ // the repeated record is a list of length 2, containing an entry for each repetition.
			[]Value{ // record contains a string followed by a nested record.
				"first repeated record",
				[]Value{
					int64(1),
					"two",
				},
			},
			[]Value{ // second record.
				"second repeated record",
				[]Value{
					int64(3),
					"four",
				},
			},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("converting repeated records containing record : got:\n%v\nwant:\n%v", got, want)
	}
}

func TestValuesSaverConvertsToMap(t *testing.T) {
	testCases := []struct {
		vs   ValuesSaver
		want *insertionRow
	}{
		{
			vs: ValuesSaver{
				Schema: []*FieldSchema{
					{Name: "intField", Type: IntegerFieldType},
					{Name: "strField", Type: StringFieldType},
				},
				InsertID: "iid",
				Row:      []Value{1, "a"},
			},
			want: &insertionRow{
				InsertID: "iid",
				Row:      map[string]Value{"intField": 1, "strField": "a"},
			},
		},
		{
			vs: ValuesSaver{
				Schema: []*FieldSchema{
					{Name: "intField", Type: IntegerFieldType},
					{
						Name: "recordField",
						Type: RecordFieldType,
						Schema: []*FieldSchema{
							{Name: "nestedInt", Type: IntegerFieldType, Repeated: true},
						},
					},
				},
				InsertID: "iid",
				Row:      []Value{1, []Value{[]Value{2, 3}}},
			},
			want: &insertionRow{
				InsertID: "iid",
				Row: map[string]Value{
					"intField": 1,
					"recordField": map[string]Value{
						"nestedInt": []Value{2, 3},
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		data, insertID, err := tc.vs.Save()
		if err != nil {
			t.Errorf("Expected successful save; got: %v", err)
		}
		got := &insertionRow{insertID, data}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("saving ValuesSaver: got:\n%v\nwant:\n%v", got, tc.want)
		}
	}
}

func TestConvertRows(t *testing.T) {
	schema := []*FieldSchema{
		{Type: StringFieldType},
		{Type: IntegerFieldType},
		{Type: FloatFieldType},
		{Type: BooleanFieldType},
	}
	rows := []*bq.TableRow{
		{F: []*bq.TableCell{
			{V: "a"},
			{V: "1"},
			{V: "1.2"},
			{V: "true"},
		}},
		{F: []*bq.TableCell{
			{V: "b"},
			{V: "2"},
			{V: "2.2"},
			{V: "false"},
		}},
	}
	want := [][]Value{
		{"a", int64(1), 1.2, true},
		{"b", int64(2), 2.2, false},
	}
	got, err := convertRows(rows, schema)
	if err != nil {
		t.Fatalf("got %v, want nil", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("\ngot  %v\nwant %v", got, want)
	}
}

func TestValueMap(t *testing.T) {
	schema := Schema{
		{Name: "s", Type: StringFieldType},
		{Name: "i", Type: IntegerFieldType},
		{Name: "f", Type: FloatFieldType},
		{Name: "b", Type: BooleanFieldType},
	}
	var vm valueMap
	if err := vm.Load([]Value{"x", 7, 3.14, true}, schema); err != nil {
		t.Fatal(err)
	}
	want := map[string]Value{
		"s": "x",
		"i": 7,
		"f": 3.14,
		"b": true,
	}
	if !reflect.DeepEqual(vm, valueMap(want)) {
		t.Errorf("got %+v, want %+v", vm, want)
	}
}

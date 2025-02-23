// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package api

import (
	"fmt"
	"github.com/SeanCondon/xpath"
	"github.com/onosproject/config-models/pkg/xpath/navigator"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"testing"
)

// Test_XPathSelectRelativeStart - start each test from switch[1] - the thing that contains all the port entries
func Test_XPathSelectRelativeStart(t *testing.T) {
	sampleConfig, err := ioutil.ReadFile("testdata/full-config-example.json")
	if err != nil {
		assert.NoError(t, err)
	}
	device := new(Device)

	schema, err := Schema()
	assert.NoError(t, err)

	if err := schema.Unmarshal(sampleConfig, device); err != nil {
		assert.NoError(t, err)
	}
	schema.Root = device
	assert.NotNil(t, device)
	ynn := navigator.NewYangNodeNavigator(schema.RootSchema(), device, true)
	assert.NotNil(t, ynn)

	tests := []navigator.XpathSelect{
		{
			Name:  "test name",
			XPath: ".",
			Expected: []string{
				"Iter Value: default: source1-2",
			},
		},
		{
			Name:  "retail areas of this shopper-monitoring",
			XPath: "../retail-area/@area-ref",
			Expected: []string{
				"Iter Value: area-ref: area1",
				"Iter Value: area-ref: area3",
			},
		},
		{
			Name:  "retail areas referenced by this shopper-monitoring",
			XPath: "/retail-area[@area-id=$this/../retail-area/@area-ref]/@area-id",
			Expected: []string{
				"Iter Value: area-id: area1",
				"Iter Value: area-id: area3",
			},
		},
	}

	for _, test := range tests {
		expr, err1 := xpath.Compile(test.XPath)
		assert.NoError(t, err1, test.Name)
		if err1 != nil {
			t.FailNow()
		}
		assert.NotNil(t, expr, test.Name)

		ynn.MoveToRoot()
		assert.True(t, ynn.MoveToChild()) // retail-area[1]
		assert.True(t, ynn.MoveToNext())  // retail-area[2]
		assert.True(t, ynn.MoveToNext())  // retail-area[3]
		assert.True(t, ynn.MoveToNext())  // store-traffic-monitoring
		assert.True(t, ynn.MoveToChild()) // default

		iter := expr.Select(ynn)
		resultCount := 0
		for iter.MoveNext() {
			assert.LessOrEqual(t, resultCount, len(test.Expected)-1, test.Name, ". More results than expected")
			assert.Equal(t, test.Expected[resultCount], fmt.Sprintf("Iter Value: %s: %s",
				iter.Current().LocalName(), iter.Current().Value()), test.Name)
			resultCount++
		}
		assert.Equal(t, len(test.Expected), resultCount, "%s. Did not receive all the expected results", test.Name)
	}
}

func Test_XPathEvaluateDefault(t *testing.T) {
	sampleConfig, err := ioutil.ReadFile("testdata/full-config-example.json")
	if err != nil {
		assert.NoError(t, err)
	}
	device := new(Device)

	schema, err := Schema()
	assert.NoError(t, err)

	if err := schema.Unmarshal(sampleConfig, device); err != nil {
		assert.NoError(t, err)
	}
	schema.Root = device
	assert.NotNil(t, device)
	ynn := navigator.NewYangNodeNavigator(schema.RootSchema(), device, true)
	assert.NotNil(t, ynn)

	tests := []navigator.XpathEvaluate{
		{
			Name:     "test this",
			XPath:    "string(.)",
			Expected: "source1-2",
		},
		{
			Name:     "number of retail areas in this store-traffic-monitoring",
			XPath:    "count($this/../retail-area)",
			Expected: float64(2),
		},
		{
			Name:     "number of retail areas in this store-traffic-monitoring",
			XPath:    "count(/retail-area[@area-id = $this/../retail-area/@area-ref]/@area-id)",
			Expected: float64(2),
		},
	}

	for _, test := range tests {
		expr, testErr := xpath.Compile(test.XPath)
		assert.NoError(t, testErr, test.Name)
		assert.NotNil(t, expr, test.Name)

		ynn.MoveToRoot()
		assert.True(t, ynn.MoveToChild()) // retail-area[1]
		assert.True(t, ynn.MoveToNext())  // retail-area[2]
		assert.True(t, ynn.MoveToNext())  // retail-area[3]
		assert.True(t, ynn.MoveToNext())  // store-traffic-monitoring
		assert.True(t, ynn.MoveToChild()) // default

		result := expr.Evaluate(ynn)
		assert.Equal(t, test.Expected, result, test.Name)
	}
}

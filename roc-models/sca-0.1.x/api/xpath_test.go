// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package api

import (
	"fmt"
	"github.com/SeanCondon/xpath"
	"github.com/onosproject/config-models/pkg/xpath/navigator"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

// Test_XPathSelectRelativeStart - start each test from collision-detection/default
func Test_XPathSelectRelativeStart(t *testing.T) {
	sampleConfig, err := os.ReadFile("testdata/full-config-example.json")
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
	nn := navigator.NewYangNodeNavigator(schema.RootSchema(), device, true)
	assert.NotNil(t, nn)
	ynn, ok := nn.(*navigator.YangNodeNavigator)
	assert.True(t, ok)

	tests := []navigator.XpathSelect{
		{
			Name:  "test name",
			XPath: ".",
			Expected: []string{
				"Iter Value: default: source1-2",
			},
		},
		{
			Name:  "districts of this collision-detection",
			XPath: "$this/../district/@district-ref",
			Expected: []string{
				"Iter Value: district-ref: district1",
				"Iter Value: district-ref: district3",
			},
		},
		{
			Name:  "districts referenced by this collision-detection",
			XPath: "/district[@district-id=$this/../district/@district-ref]/@district-id",
			Expected: []string{
				"Iter Value: district-id: district1",
				"Iter Value: district-id: district3",
			},
		},
		{
			Name:  "sources of retail areas referenced by this shopper-monitoring",
			XPath: "/district[@district-id=$this/../district/@district-ref]/source/@source-id",
			Expected: []string{
				"Iter Value: source-id: source1-1",
				"Iter Value: source-id: source1-2",
				"Iter Value: source-id: source3-1",
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

		ynn.NavigateTo("/collision-detection/default")

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
	sampleConfig, err := os.ReadFile("testdata/full-config-example.json")
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
	nn := navigator.NewYangNodeNavigator(schema.RootSchema(), device, true)
	assert.NotNil(t, nn)
	ynn, ok := nn.(*navigator.YangNodeNavigator)
	assert.True(t, ok)

	tests := []navigator.XpathEvaluate{
		{
			Name:     "test this",
			XPath:    "string(.)",
			Expected: "source1-2",
		},
		{
			Name:     "number of retail areas in this store-traffic-monitoring",
			XPath:    "count($this/../district)",
			Expected: float64(2),
		},
		{
			Name:     "number of retail areas in this store-traffic-monitoring",
			XPath:    "count(/district[@district-id = $this/../district/@district-ref]/@district-id)",
			Expected: float64(2),
		},
		{
			Name:     "number of sources of retail areas in this store-traffic-monitoring",
			XPath:    "count(/district[@district-id = $this/../district/@district-ref]/source/@source-id)",
			Expected: float64(3),
		},
	}

	for _, test := range tests {
		expr, testErr := xpath.Compile(test.XPath)
		assert.NoError(t, testErr, test.Name)
		assert.NotNil(t, expr, test.Name)

		ynn.NavigateTo("/collision-detection/default")

		result := expr.Evaluate(ynn)
		assert.Equal(t, test.Expected, result, test.Name)
	}
}

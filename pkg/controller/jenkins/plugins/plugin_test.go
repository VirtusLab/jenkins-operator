package plugins

import (
	"fmt"
	"github.com/VirtusLab/jenkins-operator/pkg/log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifyDependencies(t *testing.T) {
	data := []struct {
		basePlugins    map[string][]Plugin
		extraPlugins   map[string][]Plugin
		expectedResult bool
	}{
		{
			basePlugins: map[string][]Plugin{
				"first-root-plugin:1.0.0": {
					Must(New("first-plugin:0.0.1")),
				},
			},
			expectedResult: true,
		},
		{
			basePlugins: map[string][]Plugin{
				"first-root-plugin:1.0.0": {
					Must(New("first-plugin:0.0.1")),
				},
				"second-root-plugin:1.0.0": {
					Must(New("first-plugin:0.0.1")),
				},
			},
			expectedResult: true,
		},
		{
			basePlugins: map[string][]Plugin{
				"first-root-plugin:1.0.0": {
					Must(New("first-plugin:0.0.1")),
				},
			},
			extraPlugins: map[string][]Plugin{
				"second-root-plugin:2.0.0": {
					Must(New("first-plugin:0.0.1")),
				},
			},
			expectedResult: true,
		},
		{
			basePlugins: map[string][]Plugin{
				"first-root-plugin:1.0.0": {
					Must(New("first-plugin:0.0.1")),
				},
				"first-root-plugin:2.0.0": {
					Must(New("first-plugin:0.0.2")),
				},
			},
			expectedResult: false,
		},
		{
			basePlugins: map[string][]Plugin{
				"first-root-plugin:1.0.0": {
					Must(New("first-plugin:0.0.1")),
				},
			},
			extraPlugins: map[string][]Plugin{
				"first-root-plugin:2.0.0": {
					Must(New("first-plugin:0.0.2")),
				},
			},
			expectedResult: false,
		},
		{
			basePlugins: map[string][]Plugin{
				"invalid-plugin-name": {},
			},
			expectedResult: false,
		},
	}

	debug := false
	log.SetupLogger(&debug)

	for index, testingData := range data {
		t.Run(fmt.Sprintf("Testing %d data", index), func(t *testing.T) {
			result := VerifyDependencies(testingData.basePlugins, testingData.extraPlugins)
			assert.Equal(t, testingData.expectedResult, result)
		})
	}
}

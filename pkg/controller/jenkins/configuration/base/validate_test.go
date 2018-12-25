package base

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

func TestValidatePlugins(t *testing.T) {
	data := []struct {
		plugins        map[string][]string
		expectedResult bool
	}{
		{
			plugins: map[string][]string{
				"valid-plugin-name:1.0": {
					"valid-plugin-name:1.0",
				},
			},
			expectedResult: true,
		},
		{
			plugins: map[string][]string{
				"invalid-plugin-name": {
					"invalid-plugin-name",
				},
			},
			expectedResult: false,
		},
		{
			plugins: map[string][]string{
				"valid-plugin-name:1.0": {
					"valid-plugin-name:1.0",
					"valid-plugin-name2:1.0",
				},
			},
			expectedResult: true,
		},
		{
			plugins: map[string][]string{
				"valid-plugin-name:1.0": {},
			},
			expectedResult: true,
		},
	}

	baseReconcileLoop := New(nil, nil, logf.ZapLogger(false),
		nil, false, false)

	for index, testingData := range data {
		t.Run(fmt.Sprintf("Testing %d plugins set", index), func(t *testing.T) {
			result := baseReconcileLoop.validatePlugins(testingData.plugins)
			assert.Equal(t, testingData.expectedResult, result)
		})
	}
}

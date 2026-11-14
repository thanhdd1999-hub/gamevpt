// Feather step bonuses are exported from the read-only Flash GamePredef.FEATHER_STEP table.
package pet

import (
	_ "embed"
	"encoding/json"
	"strconv"
)

//go:embed evolution_steps.json
var evolutionStepsJSON []byte

var evolutionSteps = func() map[string]map[string][][]int {
	var data map[string]map[string][][]int
	if err := json.Unmarshal(evolutionStepsJSON, &data); err != nil {
		panic(err)
	}
	return data
}()

var featherPointCosts = [3][9]int{{10, 10, 10, 20, 20, 20, 30, 30, 30}, {15, 15, 15, 30, 30, 30, 45, 45, 45}, {20, 20, 20, 40, 40, 40, 60, 60, 60}}
var featherGoldCosts = [3][9]int64{{600, 500, 400, 1900, 1800, 1500, 3900, 3700, 3100}, {1100, 1000, 800, 3500, 3400, 2800, 7700, 7500, 6400}, {1600, 1500, 1300, 5700, 5600, 4700, 12000, 11500, 10000}}
var featherProtectionCosts = [3][9]int{{0, 1, 2, 0, 1, 4, 0, 2, 9}, {0, 1, 3, 0, 1, 8, 0, 3, 16}, {0, 1, 4, 0, 2, 12, 0, 3, 18}}
var featherSuccessRates = [3][9]int{{35, 29, 24, 30, 24, 20, 25, 20, 20}, {29, 24, 23, 24, 21, 18, 20, 18, 16}, {30, 24, 19, 24, 20, 16, 24, 18, 12}}

func featherStepValues(template, round, stage int) ([5]int, bool) {
	var result [5]int
	data, ok := evolutionSteps[strconv.Itoa(template)]
	if !ok || round < 0 || round > 2 || stage < 0 || stage > 9 {
		return result, false
	}
	for i := range result {
		values := data[strconv.Itoa(i)]
		if len(values) != 3 || len(values[round]) != 10 {
			return result, false
		}
		result[i] = values[round][stage]
	}
	return result, true
}

package processors

import "github.com/fynxlabs/rwr/internal/types"

// SelectRun is shared by command dispatch and the execution engine.
func SelectRun(c *types.InitConfig, requested []string) (types.RunSelection, error) {
	order, err := GetBlueprintRunOrder(c)
	if err != nil {
		return types.RunSelection{}, err
	}
	return types.SelectRun(c, requested, order)
}

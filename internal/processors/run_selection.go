package processors

import "github.com/freehold-digital/rwr/internal/types"

// SelectRun is shared by command dispatch and the execution engine.
func SelectRun(c *types.InitConfig, requested []string) (types.RunSelection, error) {
	order, err := GetBlueprintRunOrder(c)
	if err != nil {
		return types.RunSelection{}, err
	}
	return types.SelectRun(c, requested, order)
}

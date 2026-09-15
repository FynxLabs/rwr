package processors

import (
	"encoding/json"

	"github.com/fynxlabs/rwr/internal/omarchy"
	"github.com/fynxlabs/rwr/internal/types"
)

func omarchyOperations(files []types.ResolvedFile, cfg *types.InitConfig) ([]omarchy.Operation, error) {
	var operations []omarchy.Operation
	for _, file := range files {
		current, err := omarchy.Load(file.Resolved, file.Format, file.Path, cfg)
		if err != nil {
			return nil, err
		}
		operations = append(operations, current...)
	}
	return omarchy.Merge(operations)
}

func omarchyResources(files []types.ResolvedFile, cfg *types.InitConfig) []types.Resource {
	operations, err := omarchyOperations(files, cfg)
	if err != nil {
		return nil
	}
	var resources []types.Resource
	for _, operation := range operations {
		raw, err := json.Marshal(operation)
		if err != nil {
			continue
		}
		resources = append(resources, types.Resource{
			Processor:    types.BlueprintTypeConfiguration,
			Provider:     "omarchy",
			Name:         operation.ID,
			Action:       "reconcile",
			Status:       types.StatusPlanned,
			DesiredState: raw,
		})
	}
	return resources
}

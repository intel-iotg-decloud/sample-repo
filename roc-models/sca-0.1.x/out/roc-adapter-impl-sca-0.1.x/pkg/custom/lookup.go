// SPDX-FileCopyrightText: (C) 2022 Intel Corporation
// SPDX-License-Identifier: LicenseRef-Intel

package custom

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/intel-innersource/frameworks.edge.one-intel-edge.maestro-app.roc.config-adapter/pkg/synchronizer"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	onoserrors "github.com/onosproject/onos-lib-go/pkg/errors"
)

// LookupEnable given a pipeline name, returns the enable status
func (ss *SCASyncStep) LookupEnable(pipeline *string) (bool, error) {
	if pipeline == nil {
		return false, fmt.Errorf("nil pipeline")
	}
	if *pipeline == PipeLineCollisionDetection {
		return (ss.City != nil) &&
			(ss.City.CollisionDetection != nil) &&
			(ss.City.CollisionDetection.Enable != nil) && *ss.City.CollisionDetection.Enable, nil
	} else if *pipeline == PipelineTrafficClassification {
		return (ss.City != nil) &&
			(ss.City.TrafficClassification != nil) &&
			(ss.City.TrafficClassification.Enable != nil) && *ss.City.TrafficClassification.Enable, nil
	} else if *pipeline == PipelineTrafficMonitoring {
		return (ss.City != nil) &&
			(ss.City.TrafficMonitoring != nil) &&
			(ss.City.TrafficMonitoring.Enable != nil) && *ss.City.TrafficMonitoring.Enable, nil
	} else {
		return false, fmt.Errorf("unknown pipeline %s", *pipeline)
	}
}

// LookupDestrict given a district name returns that District
func (ss *SCASyncStep) LookupDistrict(ref *string) (*District, error) {
	if ref == nil {
		return nil, errors.New("district ref is nil")
	}

	ra, okay := ss.City.District[*ref]
	if !okay {
		return nil, fmt.Errorf("failed to find district %s", *ref)
	}

	return ra, nil
}

// LookupDistricts given a pipeline name returns the set of area references for that pipeline
func (ss *SCASyncStep) LookupDistricts(pipeline *string) ([]*DistrictRef, error) {
	districtrefs := []*DistrictRef{}
	if pipeline == nil {
		return districtrefs, fmt.Errorf("nil pipeline")
	}
	if *pipeline == PipeLineCollisionDetection {
		if ss.City.CollisionDetection == nil {
			return districtrefs, nil
		}
		// be deterministic
		areaKeys := []string{}
		for k := range ss.City.CollisionDetection.District {
			areaKeys = append(areaKeys, k)
		}
		sort.Strings(areaKeys)

		for _, areaKey := range areaKeys {
			area := ss.City.CollisionDetection.District[areaKey]
			if (area.Enabled != nil) && (*area.Enabled) {
				if area.DistrictRef != nil && area.StreamCount != nil {
					districtrefs = append(districtrefs,
						&DistrictRef{DistrictID: *area.DistrictRef, StreamCount: *area.StreamCount})
				}
			}
		}
		return districtrefs, nil

	} else if *pipeline == PipelineTrafficClassification {
		if ss.City.TrafficClassification == nil {
			return districtrefs, nil
		}
		// be deterministic
		areaKeys := []string{}
		for k := range ss.City.TrafficClassification.District {
			areaKeys = append(areaKeys, k)
		}
		sort.Strings(areaKeys)

		for _, areaKey := range areaKeys {
			area := ss.City.TrafficClassification.District[areaKey]
			if (area.Enabled != nil) && (*area.Enabled) {
				if area.DistrictRef != nil && area.StreamCount != nil {
					districtrefs = append(districtrefs,
						&DistrictRef{DistrictID: *area.DistrictRef, StreamCount: *area.StreamCount})
				}
			}
		}
		return districtrefs, nil

	} else if *pipeline == PipelineTrafficMonitoring {
		if ss.City.TrafficMonitoring == nil {
			return districtrefs, nil
		}
		// be deterministic
		areaKeys := []string{}
		for k := range ss.City.TrafficMonitoring.District {
			areaKeys = append(areaKeys, k)
		}
		sort.Strings(areaKeys)

		for _, areaKey := range areaKeys {
			area := ss.City.TrafficMonitoring.District[areaKey]
			if (area.Enabled != nil) && (*area.Enabled) {
				if area.DistrictRef != nil && area.StreamCount != nil {
					districtrefs = append(districtrefs,
						&DistrictRef{DistrictID: *area.DistrictRef, StreamCount: *area.StreamCount})
				}
			}
		}
		return districtrefs, nil
	}

	return districtrefs, fmt.Errorf("unknown pipeline %s", *pipeline)
}

func lookupSRAControllerInfo(ctx context.Context, sync synchronizer.SynchronizerInterface, storeID string) (*topoapi.ControllerInfo, error) {
	topoClient, err := sync.GetTopoClient(ctx)
	if err != nil {
		return nil, onoserrors.FromGRPC(err)
	}

	getResponse, err := topoClient.Get(ctx, &topoapi.GetRequest{
		ID: topoapi.ID(storeID),
	})
	if err != nil {
		return nil, onoserrors.FromGRPC(err)
	}
	log.Debug("topo response object: %v", getResponse.Object)

	fabricObject := getResponse.Object
	controllerInfo := &topoapi.ControllerInfo{}
	err = fabricObject.GetAspect(controllerInfo)
	if err != nil {
		return nil, onoserrors.FromGRPC(err)
	}
	log.Debug("controller address %v port %v", controllerInfo.ControlEndpoint.Address, controllerInfo.ControlEndpoint.Port)

	return controllerInfo, nil
}

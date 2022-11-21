// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package sca_0_1_x

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/onosproject/aether-roc-api/pkg/utils"
	"github.com/openconfig/gnmi/proto/gnmi"
	"net/http"
	"reflect"
	"regexp"
)

// Elements defines model for Elements.
type Elements struct {
    
    
    ElementCollisionDetection *CollisionDetection `json:"collision-detection,omitempty"`
    
    
    
    ElementTrafficMonitoring *TrafficMonitoring `json:"traffic-monitoring,omitempty"`
    
    
    
    ElementDistrict *DistrictList `json:"district,omitempty"`
    
    
    
    ElementTrafficClassification *TrafficClassification `json:"traffic-classification,omitempty"`
    
    
}

// PatchBody defines model for PatchBody.
type PatchBody struct {
	Deletes *Elements `json:"Deletes,omitempty"`

	// Model type and version of 'target' on first creation [link](https://docs.onosproject.org/onos-config/docs/gnmi_extensions/#use-of-extension-101-device-version-in-setrequest)
	Extensions *struct {
		ChangeName100   *string `json:"change-name-100,omitempty"`
		ModelType102    *string `json:"model-type-102,omitempty"`
		ModelVersion101 *string `json:"model-version-101,omitempty"`

		// Used in the responses, carries inforamtion about the transaction.
		TransactionInfo110 *struct {
			ID    *string `json:"ID,omitempty"`
			Index *int    `json:"index,omitempty"`
		} `json:"transaction-info-110,omitempty"`

		// Identifies whether a request needs to be handles Asynchronously (val: 0) or Synchronously (val: 1)
		TransactionStrategy111 *int `json:"transaction-strategy-111,omitempty"`
	} `json:"Extensions,omitempty"`
	Updates *Elements `json:"Updates,omitempty"`

	// Target (device name) to use by default if not specified on indivdual updates/deletes as an additional property
	DefaultTarget string `json:"default-target"`
}


// GnmiPatchBody contains all the info required to build a gNMI requests
type GnmiPatchBody struct {
	Updates       []*gnmi.Update
	Deletes       []*gnmi.Path
	DefaultTarget string
	Ext100Name    *string
	Ext101Version *string
	Ext102Type    *string
	Ext110Info    *struct {
		ID    *string `json:"ID,omitempty"`
		Index *int    `json:"index,omitempty"`
	}
	Ext111Strategy *int
}


// PatchAetherRocAPI impl of gNMI access at /roc-api
func (i *ServerImpl) PatchRocAPI(ctx echo.Context) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response patched
	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	response, err = i.gnmiPatchRocAPI(gnmiCtx, body, "/roc-api")
	if err != nil {
		return utils.ConvertGrpcError(err)
	}

	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	log.Infof("PatchRocAPI")
	return ctx.JSON(http.StatusOK, response)
}


// gnmiPatchRocAPI patches an existing configuration with PatchBody.
func (i *ServerImpl) gnmiPatchRocAPI(ctx context.Context, body []byte, dummy string) (*string, error) {

	var jsonObj PatchBody
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields() // Force errors

	if err := dec.Decode(&jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as PatchBody: %s", err.Error())
	}

	patchBody, err := encodeToGnmiPatchBody(&jsonObj)
	if err != nil {
		return nil, err
	}
	gnmiSet, err := utils.NewGnmiSetRequest(patchBody.Updates, patchBody.Deletes,
		patchBody.Ext100Name, patchBody.Ext101Version, patchBody.Ext102Type, patchBody.Ext111Strategy)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, fmt.Errorf(" %v", err)
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

//Ignoring AdditionalPropertyUnchanged
var patchRe = regexp.MustCompile(`[0-9a-z\-\._]+`)

const undefined = "undefined"


func encodeToGnmiPatchBody(jsonObj *PatchBody) (*GnmiPatchBody, error) {
	updates := make([]*gnmi.Update, 0)
	deletes := make([]*gnmi.Path, 0)

	pb := &GnmiPatchBody{}

	if jsonObj.Extensions != nil {
		pb.Ext100Name = jsonObj.Extensions.ChangeName100
		pb.Ext101Version = jsonObj.Extensions.ModelVersion101
		pb.Ext102Type = jsonObj.Extensions.ModelType102
		//pb.Ext110Info = jsonObj.Extensions.TransactionInfo110
		pb.Ext111Strategy = jsonObj.Extensions.TransactionStrategy111
	}

	if !patchRe.MatchString(jsonObj.DefaultTarget) {
		return nil, fmt.Errorf("default-target cannot be blank")
	}
	pb.DefaultTarget = jsonObj.DefaultTarget

	gnmiUpdates, err := encodeToGnmiElements(jsonObj.Updates, jsonObj.DefaultTarget, false)
	if err != nil {
		return nil, err
	}
	updates = append(updates, gnmiUpdates...)
	pb.Updates = updates

	gnmiDeletes, err := encodeToGnmiElements(jsonObj.Deletes, jsonObj.DefaultTarget, true)
	if err != nil {
		return nil, err
	}
	for _, gd := range gnmiDeletes {
		deletes = append(deletes, gd.Path)
	}
	pb.Deletes = deletes

	return pb, nil
}




func encodeToGnmiElements(elements *Elements, target string, forDelete bool) ([]*gnmi.Update, error) {
    if elements == nil {
        return nil, nil
    }
    updates := make([]*gnmi.Update, 0)
    
    
    
    if elements.ElementCollisionDetection != nil {
        
        
        
        ModelUpdates, err := EncodeToGnmiCollisionDetection(elements.ElementCollisionDetection, false, forDelete,
        			CityId(target), "/collision-detection")
        if err != nil {
            return nil, fmt.Errorf("EncodeToGnmiCollisionDetection %s", err)
        }
        updates = append(updates, ModelUpdates...)
    }
    
    
    
    
    
    if elements.ElementTrafficMonitoring != nil {
        
        
        
        ModelUpdates, err := EncodeToGnmiTrafficMonitoring(elements.ElementTrafficMonitoring, false, forDelete,
        			CityId(target), "/traffic-monitoring")
        if err != nil {
            return nil, fmt.Errorf("EncodeToGnmiTrafficMonitoring %s", err)
        }
        updates = append(updates, ModelUpdates...)
    }
    
    
    
    
    
    if elements.ElementDistrict != nil && len(*elements.ElementDistrict) > 0 {
        for _, e := range *elements.ElementDistrict {
            
            if e.DistrictId == undefined {
                log.Warnw("district-id is undefined", "district-id", e)
                return nil, echo.NewHTTPError(http.StatusUnprocessableEntity, "district-id cannot be undefined")
            }
            
            ModelUpdates, err := EncodeToGnmiDistrictList(elements.ElementDistrict, false,
            			forDelete, CityId(target), "/district")
            if err != nil {
                return nil, fmt.Errorf("EncodeToGnmiDistrictList() %s", err)
            }
            updates = append(updates, ModelUpdates...)
        }
    }
    
    
    
    
    if elements.ElementTrafficClassification != nil {
        
        
        
        ModelUpdates, err := EncodeToGnmiTrafficClassification(elements.ElementTrafficClassification, false, forDelete,
        			CityId(target), "/traffic-classification")
        if err != nil {
            return nil, fmt.Errorf("EncodeToGnmiTrafficClassification %s", err)
        }
        updates = append(updates, ModelUpdates...)
    }
    
    
    

    return updates, nil
}
// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

// imports template override
package sra_0_2_x

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ghodss/yaml"
	externalRef0 "github.com/intel-innersource/frameworks.edge.one-intel-edge.springboard.reference-implementation.roc-models/models/sra-0.2.x/api"
	"github.com/labstack/echo/v4"
	"github.com/onosproject/aether-roc-api/pkg/southbound"
	"github.com/onosproject/aether-roc-api/pkg/utils"
	"github.com/onosproject/onos-api/go/onos/config/admin"
	topoapi "github.com/onosproject/onos-api/go/onos/topo"
	"github.com/onosproject/onos-config/pkg/store/topo"
	"github.com/onosproject/onos-lib-go/pkg/certs"
	"github.com/onosproject/onos-lib-go/pkg/logging"
	"github.com/openconfig/gnmi/proto/gnmi"
)

// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

// HTMLData -
type HTMLData struct {
	File        string
	Description string
}

// GetSpec -
func (i *ServerImpl) GetSpec(ctx echo.Context) error {
	response, err := GetSwagger()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}
	return acceptTypes(ctx, response)
}

// GetTargets -
func (i *ServerImpl) GetTargets(ctx echo.Context, typeId string) error {
	var err error

	opts, err := certs.HandleCertPaths("", "", "", true)
	if err != nil {
		return err
	}

	topoClient, err := topo.NewStore(i.TopoEndpoint, opts...)
	if err != nil {
		return utils.ConvertGrpcError(err)
	}

	topoCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	kindFilter := &topoapi.Filters{}
	if typeId != "" {
		kindFilter = &topoapi.Filters{
			KindFilter: &topoapi.Filter{
				Filter: &topoapi.Filter_Equal_{
					Equal_: &topoapi.EqualFilter{
						Value: typeId,
					},
				},
			},
		}

	} else {
		kindFilter = &topoapi.Filters{
			ObjectTypes: []topoapi.Object_Type{topoapi.Object_ENTITY},
			WithAspects: []string{"onos.topo.Configurable"},
		}
	}

	objects, err := topoClient.List(topoCtx, kindFilter)
	if err != nil {
		return utils.ConvertGrpcError(err)
	}

	targetsNames := make(TargetsNames, 0)
	for _, object := range objects {
		targetName := string(object.ID)
		targetsNames = append(targetsNames, TargetName{
			Name: &targetName,
		})
	}
	log.Infof("GetTargets")
	return ctx.JSON(http.StatusOK, targetsNames)
}

//Ignoring AdditionalPropertiesUnchTarget

//Ignoring AdditionalPropertyStoreId

//Ignoring AdditionalPropertyUnchanged

//Ignoring LeafRefOption

//Ignoring LeafRefOptions

// GnmiDeleteRetailArea deletes an instance of Retail-area.
func (i *ServerImpl) GnmiDeleteRetailArea(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetRetailArea(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetRetailArea returns an instance of Retail-area.
func (i *ServerImpl) GnmiGetRetailArea(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*RetailArea, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToRetailArea(args...)
}

// GnmiPostRetailArea adds an instance of Retail-area.
func (i *ServerImpl) GnmiPostRetailArea(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(RetailArea)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Retail-area %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiRetailArea(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert RetailArea to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteRetailAreaList deletes an instance of Retail-area_List.
func (i *ServerImpl) GnmiDeleteRetailAreaList(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetRetailAreaList(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetRetailAreaList returns an instance of Retail-area_List.
func (i *ServerImpl) GnmiGetRetailAreaList(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*RetailAreaList, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToRetailAreaList(args...)
}

// GnmiPostRetailAreaList adds an instance of Retail-area_List.
func (i *ServerImpl) GnmiPostRetailAreaList(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(RetailAreaList)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Retail-area_List %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiRetailAreaList(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert RetailAreaList to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteRetailAreaLocation deletes an instance of Retail-area_Location.
func (i *ServerImpl) GnmiDeleteRetailAreaLocation(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetRetailAreaLocation(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetRetailAreaLocation returns an instance of Retail-area_Location.
func (i *ServerImpl) GnmiGetRetailAreaLocation(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*RetailAreaLocation, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToRetailAreaLocation(args...)
}

// GnmiPostRetailAreaLocation adds an instance of Retail-area_Location.
func (i *ServerImpl) GnmiPostRetailAreaLocation(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(RetailAreaLocation)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Retail-area_Location %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiRetailAreaLocation(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert RetailAreaLocation to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteRetailAreaLocationCoordinateSystem deletes an instance of RetailAreaLocation.CoordinateSystem.
func (i *ServerImpl) GnmiDeleteRetailAreaLocationCoordinateSystem(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetRetailAreaLocationCoordinateSystem(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetRetailAreaLocationCoordinateSystem returns an instance of RetailAreaLocation.CoordinateSystem.
func (i *ServerImpl) GnmiGetRetailAreaLocationCoordinateSystem(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*RetailAreaLocationCoordinateSystem, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToRetailAreaLocationCoordinateSystem(args...)
}

// GnmiPostRetailAreaLocationCoordinateSystem adds an instance of RetailAreaLocation.CoordinateSystem.
func (i *ServerImpl) GnmiPostRetailAreaLocationCoordinateSystem(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(RetailAreaLocationCoordinateSystem)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as RetailAreaLocation.CoordinateSystem %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiRetailAreaLocationCoordinateSystem(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert RetailAreaLocationCoordinateSystem to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteRetailAreaSource deletes an instance of Retail-area_Source.
func (i *ServerImpl) GnmiDeleteRetailAreaSource(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetRetailAreaSource(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetRetailAreaSource returns an instance of Retail-area_Source.
func (i *ServerImpl) GnmiGetRetailAreaSource(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*RetailAreaSource, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToRetailAreaSource(args...)
}

// GnmiPostRetailAreaSource adds an instance of Retail-area_Source.
func (i *ServerImpl) GnmiPostRetailAreaSource(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(RetailAreaSource)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Retail-area_Source %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiRetailAreaSource(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert RetailAreaSource to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteRetailAreaSourceList deletes an instance of Retail-area_Source_List.
func (i *ServerImpl) GnmiDeleteRetailAreaSourceList(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetRetailAreaSourceList(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetRetailAreaSourceList returns an instance of Retail-area_Source_List.
func (i *ServerImpl) GnmiGetRetailAreaSourceList(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*RetailAreaSourceList, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToRetailAreaSourceList(args...)
}

// GnmiPostRetailAreaSourceList adds an instance of Retail-area_Source_List.
func (i *ServerImpl) GnmiPostRetailAreaSourceList(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(RetailAreaSourceList)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Retail-area_Source_List %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiRetailAreaSourceList(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert RetailAreaSourceList to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteRetailAreaSourceLocation deletes an instance of Retail-area_Source_Location.
func (i *ServerImpl) GnmiDeleteRetailAreaSourceLocation(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetRetailAreaSourceLocation(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetRetailAreaSourceLocation returns an instance of Retail-area_Source_Location.
func (i *ServerImpl) GnmiGetRetailAreaSourceLocation(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*RetailAreaSourceLocation, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToRetailAreaSourceLocation(args...)
}

// GnmiPostRetailAreaSourceLocation adds an instance of Retail-area_Source_Location.
func (i *ServerImpl) GnmiPostRetailAreaSourceLocation(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(RetailAreaSourceLocation)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Retail-area_Source_Location %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiRetailAreaSourceLocation(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert RetailAreaSourceLocation to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteRetailAreaSourceLocationCoordinateSystem deletes an instance of RetailAreaSourceLocation.CoordinateSystem.
func (i *ServerImpl) GnmiDeleteRetailAreaSourceLocationCoordinateSystem(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetRetailAreaSourceLocationCoordinateSystem(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetRetailAreaSourceLocationCoordinateSystem returns an instance of RetailAreaSourceLocation.CoordinateSystem.
func (i *ServerImpl) GnmiGetRetailAreaSourceLocationCoordinateSystem(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*RetailAreaSourceLocationCoordinateSystem, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToRetailAreaSourceLocationCoordinateSystem(args...)
}

// GnmiPostRetailAreaSourceLocationCoordinateSystem adds an instance of RetailAreaSourceLocation.CoordinateSystem.
func (i *ServerImpl) GnmiPostRetailAreaSourceLocationCoordinateSystem(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(RetailAreaSourceLocationCoordinateSystem)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as RetailAreaSourceLocation.CoordinateSystem %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiRetailAreaSourceLocationCoordinateSystem(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert RetailAreaSourceLocationCoordinateSystem to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteRetailAreaSourceState deletes an instance of Retail-area_Source_State.
func (i *ServerImpl) GnmiDeleteRetailAreaSourceState(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetRetailAreaSourceState(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetRetailAreaSourceState returns an instance of Retail-area_Source_State.
func (i *ServerImpl) GnmiGetRetailAreaSourceState(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*RetailAreaSourceState, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToRetailAreaSourceState(args...)
}

// GnmiPostRetailAreaSourceState adds an instance of Retail-area_Source_State.
func (i *ServerImpl) GnmiPostRetailAreaSourceState(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(RetailAreaSourceState)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Retail-area_Source_State %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiRetailAreaSourceState(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert RetailAreaSourceState to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteRetailAreaSourceVideo deletes an instance of Retail-area_Source_Video.
func (i *ServerImpl) GnmiDeleteRetailAreaSourceVideo(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetRetailAreaSourceVideo(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetRetailAreaSourceVideo returns an instance of Retail-area_Source_Video.
func (i *ServerImpl) GnmiGetRetailAreaSourceVideo(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*RetailAreaSourceVideo, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToRetailAreaSourceVideo(args...)
}

// GnmiPostRetailAreaSourceVideo adds an instance of Retail-area_Source_Video.
func (i *ServerImpl) GnmiPostRetailAreaSourceVideo(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(RetailAreaSourceVideo)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Retail-area_Source_Video %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiRetailAreaSourceVideo(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert RetailAreaSourceVideo to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteRetailAreaSourceVideoSourceType deletes an instance of RetailAreaSourceVideo.SourceType.
func (i *ServerImpl) GnmiDeleteRetailAreaSourceVideoSourceType(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetRetailAreaSourceVideoSourceType(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetRetailAreaSourceVideoSourceType returns an instance of RetailAreaSourceVideo.SourceType.
func (i *ServerImpl) GnmiGetRetailAreaSourceVideoSourceType(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*RetailAreaSourceVideoSourceType, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToRetailAreaSourceVideoSourceType(args...)
}

// GnmiPostRetailAreaSourceVideoSourceType adds an instance of RetailAreaSourceVideo.SourceType.
func (i *ServerImpl) GnmiPostRetailAreaSourceVideoSourceType(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(RetailAreaSourceVideoSourceType)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as RetailAreaSourceVideo.SourceType %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiRetailAreaSourceVideoSourceType(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert RetailAreaSourceVideoSourceType to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShelfMonitoring deletes an instance of Shelf-monitoring.
func (i *ServerImpl) GnmiDeleteShelfMonitoring(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShelfMonitoring(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShelfMonitoring returns an instance of Shelf-monitoring.
func (i *ServerImpl) GnmiGetShelfMonitoring(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShelfMonitoring, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShelfMonitoring(args...)
}

// GnmiPostShelfMonitoring adds an instance of Shelf-monitoring.
func (i *ServerImpl) GnmiPostShelfMonitoring(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShelfMonitoring)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shelf-monitoring %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShelfMonitoring(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShelfMonitoring to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShelfMonitoringObjectDetectionApplication deletes an instance of Shelf-monitoring_Object-detection-application.
func (i *ServerImpl) GnmiDeleteShelfMonitoringObjectDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShelfMonitoringObjectDetectionApplication(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShelfMonitoringObjectDetectionApplication returns an instance of Shelf-monitoring_Object-detection-application.
func (i *ServerImpl) GnmiGetShelfMonitoringObjectDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShelfMonitoringObjectDetectionApplication, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShelfMonitoringObjectDetectionApplication(args...)
}

// GnmiPostShelfMonitoringObjectDetectionApplication adds an instance of Shelf-monitoring_Object-detection-application.
func (i *ServerImpl) GnmiPostShelfMonitoringObjectDetectionApplication(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShelfMonitoringObjectDetectionApplication)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shelf-monitoring_Object-detection-application %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShelfMonitoringObjectDetectionApplication(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShelfMonitoringObjectDetectionApplication to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShelfMonitoringObjectDetectionApplicationDevice deletes an instance of ShelfMonitoringObjectDetectionApplication.Device.
func (i *ServerImpl) GnmiDeleteShelfMonitoringObjectDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShelfMonitoringObjectDetectionApplicationDevice(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShelfMonitoringObjectDetectionApplicationDevice returns an instance of ShelfMonitoringObjectDetectionApplication.Device.
func (i *ServerImpl) GnmiGetShelfMonitoringObjectDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShelfMonitoringObjectDetectionApplicationDevice, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShelfMonitoringObjectDetectionApplicationDevice(args...)
}

// GnmiPostShelfMonitoringObjectDetectionApplicationDevice adds an instance of ShelfMonitoringObjectDetectionApplication.Device.
func (i *ServerImpl) GnmiPostShelfMonitoringObjectDetectionApplicationDevice(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShelfMonitoringObjectDetectionApplicationDevice)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as ShelfMonitoringObjectDetectionApplication.Device %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShelfMonitoringObjectDetectionApplicationDevice(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShelfMonitoringObjectDetectionApplicationDevice to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShelfMonitoringObjectDetectionApplicationPrecision deletes an instance of ShelfMonitoringObjectDetectionApplication.Precision.
func (i *ServerImpl) GnmiDeleteShelfMonitoringObjectDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShelfMonitoringObjectDetectionApplicationPrecision(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShelfMonitoringObjectDetectionApplicationPrecision returns an instance of ShelfMonitoringObjectDetectionApplication.Precision.
func (i *ServerImpl) GnmiGetShelfMonitoringObjectDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShelfMonitoringObjectDetectionApplicationPrecision, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShelfMonitoringObjectDetectionApplicationPrecision(args...)
}

// GnmiPostShelfMonitoringObjectDetectionApplicationPrecision adds an instance of ShelfMonitoringObjectDetectionApplication.Precision.
func (i *ServerImpl) GnmiPostShelfMonitoringObjectDetectionApplicationPrecision(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShelfMonitoringObjectDetectionApplicationPrecision)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as ShelfMonitoringObjectDetectionApplication.Precision %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShelfMonitoringObjectDetectionApplicationPrecision(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShelfMonitoringObjectDetectionApplicationPrecision to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShelfMonitoringObjectDetectionApplicationModelState deletes an instance of Shelf-monitoring_Object-detection-application_Model-state.
func (i *ServerImpl) GnmiDeleteShelfMonitoringObjectDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShelfMonitoringObjectDetectionApplicationModelState(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShelfMonitoringObjectDetectionApplicationModelState returns an instance of Shelf-monitoring_Object-detection-application_Model-state.
func (i *ServerImpl) GnmiGetShelfMonitoringObjectDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShelfMonitoringObjectDetectionApplicationModelState, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShelfMonitoringObjectDetectionApplicationModelState(args...)
}

// GnmiPostShelfMonitoringObjectDetectionApplicationModelState adds an instance of Shelf-monitoring_Object-detection-application_Model-state.
func (i *ServerImpl) GnmiPostShelfMonitoringObjectDetectionApplicationModelState(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShelfMonitoringObjectDetectionApplicationModelState)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shelf-monitoring_Object-detection-application_Model-state %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShelfMonitoringObjectDetectionApplicationModelState(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShelfMonitoringObjectDetectionApplicationModelState to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShelfMonitoringRetailArea deletes an instance of Shelf-monitoring_Retail-area.
func (i *ServerImpl) GnmiDeleteShelfMonitoringRetailArea(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShelfMonitoringRetailArea(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShelfMonitoringRetailArea returns an instance of Shelf-monitoring_Retail-area.
func (i *ServerImpl) GnmiGetShelfMonitoringRetailArea(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShelfMonitoringRetailArea, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShelfMonitoringRetailArea(args...)
}

// GnmiPostShelfMonitoringRetailArea adds an instance of Shelf-monitoring_Retail-area.
func (i *ServerImpl) GnmiPostShelfMonitoringRetailArea(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShelfMonitoringRetailArea)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shelf-monitoring_Retail-area %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShelfMonitoringRetailArea(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShelfMonitoringRetailArea to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShelfMonitoringRetailAreaList deletes an instance of Shelf-monitoring_Retail-area_List.
func (i *ServerImpl) GnmiDeleteShelfMonitoringRetailAreaList(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShelfMonitoringRetailAreaList(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShelfMonitoringRetailAreaList returns an instance of Shelf-monitoring_Retail-area_List.
func (i *ServerImpl) GnmiGetShelfMonitoringRetailAreaList(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShelfMonitoringRetailAreaList, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShelfMonitoringRetailAreaList(args...)
}

// GnmiPostShelfMonitoringRetailAreaList adds an instance of Shelf-monitoring_Retail-area_List.
func (i *ServerImpl) GnmiPostShelfMonitoringRetailAreaList(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShelfMonitoringRetailAreaList)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shelf-monitoring_Retail-area_List %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShelfMonitoringRetailAreaList(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShelfMonitoringRetailAreaList to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoring deletes an instance of Shopper-monitoring.
func (i *ServerImpl) GnmiDeleteShopperMonitoring(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoring(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoring returns an instance of Shopper-monitoring.
func (i *ServerImpl) GnmiGetShopperMonitoring(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoring, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoring(args...)
}

// GnmiPostShopperMonitoring adds an instance of Shopper-monitoring.
func (i *ServerImpl) GnmiPostShopperMonitoring(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoring)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shopper-monitoring %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoring(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoring to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringEmotionRecognitionApplication deletes an instance of Shopper-monitoring_Emotion-recognition-application.
func (i *ServerImpl) GnmiDeleteShopperMonitoringEmotionRecognitionApplication(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringEmotionRecognitionApplication(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringEmotionRecognitionApplication returns an instance of Shopper-monitoring_Emotion-recognition-application.
func (i *ServerImpl) GnmiGetShopperMonitoringEmotionRecognitionApplication(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringEmotionRecognitionApplication, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringEmotionRecognitionApplication(args...)
}

// GnmiPostShopperMonitoringEmotionRecognitionApplication adds an instance of Shopper-monitoring_Emotion-recognition-application.
func (i *ServerImpl) GnmiPostShopperMonitoringEmotionRecognitionApplication(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringEmotionRecognitionApplication)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shopper-monitoring_Emotion-recognition-application %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringEmotionRecognitionApplication(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringEmotionRecognitionApplication to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringEmotionRecognitionApplicationDevice deletes an instance of ShopperMonitoringEmotionRecognitionApplication.Device.
func (i *ServerImpl) GnmiDeleteShopperMonitoringEmotionRecognitionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringEmotionRecognitionApplicationDevice(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringEmotionRecognitionApplicationDevice returns an instance of ShopperMonitoringEmotionRecognitionApplication.Device.
func (i *ServerImpl) GnmiGetShopperMonitoringEmotionRecognitionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringEmotionRecognitionApplicationDevice, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringEmotionRecognitionApplicationDevice(args...)
}

// GnmiPostShopperMonitoringEmotionRecognitionApplicationDevice adds an instance of ShopperMonitoringEmotionRecognitionApplication.Device.
func (i *ServerImpl) GnmiPostShopperMonitoringEmotionRecognitionApplicationDevice(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringEmotionRecognitionApplicationDevice)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as ShopperMonitoringEmotionRecognitionApplication.Device %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringEmotionRecognitionApplicationDevice(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringEmotionRecognitionApplicationDevice to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringEmotionRecognitionApplicationPrecision deletes an instance of ShopperMonitoringEmotionRecognitionApplication.Precision.
func (i *ServerImpl) GnmiDeleteShopperMonitoringEmotionRecognitionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringEmotionRecognitionApplicationPrecision(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringEmotionRecognitionApplicationPrecision returns an instance of ShopperMonitoringEmotionRecognitionApplication.Precision.
func (i *ServerImpl) GnmiGetShopperMonitoringEmotionRecognitionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringEmotionRecognitionApplicationPrecision, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringEmotionRecognitionApplicationPrecision(args...)
}

// GnmiPostShopperMonitoringEmotionRecognitionApplicationPrecision adds an instance of ShopperMonitoringEmotionRecognitionApplication.Precision.
func (i *ServerImpl) GnmiPostShopperMonitoringEmotionRecognitionApplicationPrecision(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringEmotionRecognitionApplicationPrecision)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as ShopperMonitoringEmotionRecognitionApplication.Precision %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringEmotionRecognitionApplicationPrecision(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringEmotionRecognitionApplicationPrecision to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringEmotionRecognitionApplicationModelState deletes an instance of Shopper-monitoring_Emotion-recognition-application_Model-state.
func (i *ServerImpl) GnmiDeleteShopperMonitoringEmotionRecognitionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringEmotionRecognitionApplicationModelState(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringEmotionRecognitionApplicationModelState returns an instance of Shopper-monitoring_Emotion-recognition-application_Model-state.
func (i *ServerImpl) GnmiGetShopperMonitoringEmotionRecognitionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringEmotionRecognitionApplicationModelState, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringEmotionRecognitionApplicationModelState(args...)
}

// GnmiPostShopperMonitoringEmotionRecognitionApplicationModelState adds an instance of Shopper-monitoring_Emotion-recognition-application_Model-state.
func (i *ServerImpl) GnmiPostShopperMonitoringEmotionRecognitionApplicationModelState(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringEmotionRecognitionApplicationModelState)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shopper-monitoring_Emotion-recognition-application_Model-state %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringEmotionRecognitionApplicationModelState(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringEmotionRecognitionApplicationModelState to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringFaceDetectionApplication deletes an instance of Shopper-monitoring_Face-detection-application.
func (i *ServerImpl) GnmiDeleteShopperMonitoringFaceDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringFaceDetectionApplication(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringFaceDetectionApplication returns an instance of Shopper-monitoring_Face-detection-application.
func (i *ServerImpl) GnmiGetShopperMonitoringFaceDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringFaceDetectionApplication, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringFaceDetectionApplication(args...)
}

// GnmiPostShopperMonitoringFaceDetectionApplication adds an instance of Shopper-monitoring_Face-detection-application.
func (i *ServerImpl) GnmiPostShopperMonitoringFaceDetectionApplication(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringFaceDetectionApplication)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shopper-monitoring_Face-detection-application %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringFaceDetectionApplication(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringFaceDetectionApplication to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringFaceDetectionApplicationDevice deletes an instance of ShopperMonitoringFaceDetectionApplication.Device.
func (i *ServerImpl) GnmiDeleteShopperMonitoringFaceDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringFaceDetectionApplicationDevice(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringFaceDetectionApplicationDevice returns an instance of ShopperMonitoringFaceDetectionApplication.Device.
func (i *ServerImpl) GnmiGetShopperMonitoringFaceDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringFaceDetectionApplicationDevice, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringFaceDetectionApplicationDevice(args...)
}

// GnmiPostShopperMonitoringFaceDetectionApplicationDevice adds an instance of ShopperMonitoringFaceDetectionApplication.Device.
func (i *ServerImpl) GnmiPostShopperMonitoringFaceDetectionApplicationDevice(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringFaceDetectionApplicationDevice)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as ShopperMonitoringFaceDetectionApplication.Device %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringFaceDetectionApplicationDevice(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringFaceDetectionApplicationDevice to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringFaceDetectionApplicationPrecision deletes an instance of ShopperMonitoringFaceDetectionApplication.Precision.
func (i *ServerImpl) GnmiDeleteShopperMonitoringFaceDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringFaceDetectionApplicationPrecision(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringFaceDetectionApplicationPrecision returns an instance of ShopperMonitoringFaceDetectionApplication.Precision.
func (i *ServerImpl) GnmiGetShopperMonitoringFaceDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringFaceDetectionApplicationPrecision, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringFaceDetectionApplicationPrecision(args...)
}

// GnmiPostShopperMonitoringFaceDetectionApplicationPrecision adds an instance of ShopperMonitoringFaceDetectionApplication.Precision.
func (i *ServerImpl) GnmiPostShopperMonitoringFaceDetectionApplicationPrecision(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringFaceDetectionApplicationPrecision)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as ShopperMonitoringFaceDetectionApplication.Precision %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringFaceDetectionApplicationPrecision(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringFaceDetectionApplicationPrecision to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringFaceDetectionApplicationModelState deletes an instance of Shopper-monitoring_Face-detection-application_Model-state.
func (i *ServerImpl) GnmiDeleteShopperMonitoringFaceDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringFaceDetectionApplicationModelState(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringFaceDetectionApplicationModelState returns an instance of Shopper-monitoring_Face-detection-application_Model-state.
func (i *ServerImpl) GnmiGetShopperMonitoringFaceDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringFaceDetectionApplicationModelState, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringFaceDetectionApplicationModelState(args...)
}

// GnmiPostShopperMonitoringFaceDetectionApplicationModelState adds an instance of Shopper-monitoring_Face-detection-application_Model-state.
func (i *ServerImpl) GnmiPostShopperMonitoringFaceDetectionApplicationModelState(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringFaceDetectionApplicationModelState)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shopper-monitoring_Face-detection-application_Model-state %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringFaceDetectionApplicationModelState(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringFaceDetectionApplicationModelState to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringHeadPoseDetectionApplication deletes an instance of Shopper-monitoring_Head-pose-detection-application.
func (i *ServerImpl) GnmiDeleteShopperMonitoringHeadPoseDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringHeadPoseDetectionApplication(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringHeadPoseDetectionApplication returns an instance of Shopper-monitoring_Head-pose-detection-application.
func (i *ServerImpl) GnmiGetShopperMonitoringHeadPoseDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringHeadPoseDetectionApplication, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringHeadPoseDetectionApplication(args...)
}

// GnmiPostShopperMonitoringHeadPoseDetectionApplication adds an instance of Shopper-monitoring_Head-pose-detection-application.
func (i *ServerImpl) GnmiPostShopperMonitoringHeadPoseDetectionApplication(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringHeadPoseDetectionApplication)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shopper-monitoring_Head-pose-detection-application %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringHeadPoseDetectionApplication(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringHeadPoseDetectionApplication to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringHeadPoseDetectionApplicationDevice deletes an instance of ShopperMonitoringHeadPoseDetectionApplication.Device.
func (i *ServerImpl) GnmiDeleteShopperMonitoringHeadPoseDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringHeadPoseDetectionApplicationDevice(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringHeadPoseDetectionApplicationDevice returns an instance of ShopperMonitoringHeadPoseDetectionApplication.Device.
func (i *ServerImpl) GnmiGetShopperMonitoringHeadPoseDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringHeadPoseDetectionApplicationDevice, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringHeadPoseDetectionApplicationDevice(args...)
}

// GnmiPostShopperMonitoringHeadPoseDetectionApplicationDevice adds an instance of ShopperMonitoringHeadPoseDetectionApplication.Device.
func (i *ServerImpl) GnmiPostShopperMonitoringHeadPoseDetectionApplicationDevice(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringHeadPoseDetectionApplicationDevice)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as ShopperMonitoringHeadPoseDetectionApplication.Device %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringHeadPoseDetectionApplicationDevice(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringHeadPoseDetectionApplicationDevice to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringHeadPoseDetectionApplicationPrecision deletes an instance of ShopperMonitoringHeadPoseDetectionApplication.Precision.
func (i *ServerImpl) GnmiDeleteShopperMonitoringHeadPoseDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringHeadPoseDetectionApplicationPrecision(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringHeadPoseDetectionApplicationPrecision returns an instance of ShopperMonitoringHeadPoseDetectionApplication.Precision.
func (i *ServerImpl) GnmiGetShopperMonitoringHeadPoseDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringHeadPoseDetectionApplicationPrecision, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringHeadPoseDetectionApplicationPrecision(args...)
}

// GnmiPostShopperMonitoringHeadPoseDetectionApplicationPrecision adds an instance of ShopperMonitoringHeadPoseDetectionApplication.Precision.
func (i *ServerImpl) GnmiPostShopperMonitoringHeadPoseDetectionApplicationPrecision(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringHeadPoseDetectionApplicationPrecision)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as ShopperMonitoringHeadPoseDetectionApplication.Precision %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringHeadPoseDetectionApplicationPrecision(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringHeadPoseDetectionApplicationPrecision to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringHeadPoseDetectionApplicationModelState deletes an instance of Shopper-monitoring_Head-pose-detection-application_Model-state.
func (i *ServerImpl) GnmiDeleteShopperMonitoringHeadPoseDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringHeadPoseDetectionApplicationModelState(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringHeadPoseDetectionApplicationModelState returns an instance of Shopper-monitoring_Head-pose-detection-application_Model-state.
func (i *ServerImpl) GnmiGetShopperMonitoringHeadPoseDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringHeadPoseDetectionApplicationModelState, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringHeadPoseDetectionApplicationModelState(args...)
}

// GnmiPostShopperMonitoringHeadPoseDetectionApplicationModelState adds an instance of Shopper-monitoring_Head-pose-detection-application_Model-state.
func (i *ServerImpl) GnmiPostShopperMonitoringHeadPoseDetectionApplicationModelState(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringHeadPoseDetectionApplicationModelState)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shopper-monitoring_Head-pose-detection-application_Model-state %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringHeadPoseDetectionApplicationModelState(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringHeadPoseDetectionApplicationModelState to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringRetailArea deletes an instance of Shopper-monitoring_Retail-area.
func (i *ServerImpl) GnmiDeleteShopperMonitoringRetailArea(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringRetailArea(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringRetailArea returns an instance of Shopper-monitoring_Retail-area.
func (i *ServerImpl) GnmiGetShopperMonitoringRetailArea(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringRetailArea, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringRetailArea(args...)
}

// GnmiPostShopperMonitoringRetailArea adds an instance of Shopper-monitoring_Retail-area.
func (i *ServerImpl) GnmiPostShopperMonitoringRetailArea(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringRetailArea)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shopper-monitoring_Retail-area %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringRetailArea(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringRetailArea to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteShopperMonitoringRetailAreaList deletes an instance of Shopper-monitoring_Retail-area_List.
func (i *ServerImpl) GnmiDeleteShopperMonitoringRetailAreaList(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetShopperMonitoringRetailAreaList(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetShopperMonitoringRetailAreaList returns an instance of Shopper-monitoring_Retail-area_List.
func (i *ServerImpl) GnmiGetShopperMonitoringRetailAreaList(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*ShopperMonitoringRetailAreaList, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToShopperMonitoringRetailAreaList(args...)
}

// GnmiPostShopperMonitoringRetailAreaList adds an instance of Shopper-monitoring_Retail-area_List.
func (i *ServerImpl) GnmiPostShopperMonitoringRetailAreaList(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(ShopperMonitoringRetailAreaList)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Shopper-monitoring_Retail-area_List %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiShopperMonitoringRetailAreaList(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert ShopperMonitoringRetailAreaList to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteStoreTrafficMonitoring deletes an instance of Store-traffic-monitoring.
func (i *ServerImpl) GnmiDeleteStoreTrafficMonitoring(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetStoreTrafficMonitoring(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetStoreTrafficMonitoring returns an instance of Store-traffic-monitoring.
func (i *ServerImpl) GnmiGetStoreTrafficMonitoring(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*StoreTrafficMonitoring, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToStoreTrafficMonitoring(args...)
}

// GnmiPostStoreTrafficMonitoring adds an instance of Store-traffic-monitoring.
func (i *ServerImpl) GnmiPostStoreTrafficMonitoring(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(StoreTrafficMonitoring)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Store-traffic-monitoring %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiStoreTrafficMonitoring(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert StoreTrafficMonitoring to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteStoreTrafficMonitoringPersonDetectionApplication deletes an instance of Store-traffic-monitoring_Person-detection-application.
func (i *ServerImpl) GnmiDeleteStoreTrafficMonitoringPersonDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetStoreTrafficMonitoringPersonDetectionApplication(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetStoreTrafficMonitoringPersonDetectionApplication returns an instance of Store-traffic-monitoring_Person-detection-application.
func (i *ServerImpl) GnmiGetStoreTrafficMonitoringPersonDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*StoreTrafficMonitoringPersonDetectionApplication, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToStoreTrafficMonitoringPersonDetectionApplication(args...)
}

// GnmiPostStoreTrafficMonitoringPersonDetectionApplication adds an instance of Store-traffic-monitoring_Person-detection-application.
func (i *ServerImpl) GnmiPostStoreTrafficMonitoringPersonDetectionApplication(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(StoreTrafficMonitoringPersonDetectionApplication)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Store-traffic-monitoring_Person-detection-application %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiStoreTrafficMonitoringPersonDetectionApplication(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert StoreTrafficMonitoringPersonDetectionApplication to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteStoreTrafficMonitoringPersonDetectionApplicationDevice deletes an instance of StoreTrafficMonitoringPersonDetectionApplication.Device.
func (i *ServerImpl) GnmiDeleteStoreTrafficMonitoringPersonDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetStoreTrafficMonitoringPersonDetectionApplicationDevice(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetStoreTrafficMonitoringPersonDetectionApplicationDevice returns an instance of StoreTrafficMonitoringPersonDetectionApplication.Device.
func (i *ServerImpl) GnmiGetStoreTrafficMonitoringPersonDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*StoreTrafficMonitoringPersonDetectionApplicationDevice, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToStoreTrafficMonitoringPersonDetectionApplicationDevice(args...)
}

// GnmiPostStoreTrafficMonitoringPersonDetectionApplicationDevice adds an instance of StoreTrafficMonitoringPersonDetectionApplication.Device.
func (i *ServerImpl) GnmiPostStoreTrafficMonitoringPersonDetectionApplicationDevice(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(StoreTrafficMonitoringPersonDetectionApplicationDevice)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as StoreTrafficMonitoringPersonDetectionApplication.Device %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiStoreTrafficMonitoringPersonDetectionApplicationDevice(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert StoreTrafficMonitoringPersonDetectionApplicationDevice to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteStoreTrafficMonitoringPersonDetectionApplicationPrecision deletes an instance of StoreTrafficMonitoringPersonDetectionApplication.Precision.
func (i *ServerImpl) GnmiDeleteStoreTrafficMonitoringPersonDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetStoreTrafficMonitoringPersonDetectionApplicationPrecision(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetStoreTrafficMonitoringPersonDetectionApplicationPrecision returns an instance of StoreTrafficMonitoringPersonDetectionApplication.Precision.
func (i *ServerImpl) GnmiGetStoreTrafficMonitoringPersonDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*StoreTrafficMonitoringPersonDetectionApplicationPrecision, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToStoreTrafficMonitoringPersonDetectionApplicationPrecision(args...)
}

// GnmiPostStoreTrafficMonitoringPersonDetectionApplicationPrecision adds an instance of StoreTrafficMonitoringPersonDetectionApplication.Precision.
func (i *ServerImpl) GnmiPostStoreTrafficMonitoringPersonDetectionApplicationPrecision(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(StoreTrafficMonitoringPersonDetectionApplicationPrecision)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as StoreTrafficMonitoringPersonDetectionApplication.Precision %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiStoreTrafficMonitoringPersonDetectionApplicationPrecision(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert StoreTrafficMonitoringPersonDetectionApplicationPrecision to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteStoreTrafficMonitoringPersonDetectionApplicationModelState deletes an instance of Store-traffic-monitoring_Person-detection-application_Model-state.
func (i *ServerImpl) GnmiDeleteStoreTrafficMonitoringPersonDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetStoreTrafficMonitoringPersonDetectionApplicationModelState(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetStoreTrafficMonitoringPersonDetectionApplicationModelState returns an instance of Store-traffic-monitoring_Person-detection-application_Model-state.
func (i *ServerImpl) GnmiGetStoreTrafficMonitoringPersonDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*StoreTrafficMonitoringPersonDetectionApplicationModelState, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToStoreTrafficMonitoringPersonDetectionApplicationModelState(args...)
}

// GnmiPostStoreTrafficMonitoringPersonDetectionApplicationModelState adds an instance of Store-traffic-monitoring_Person-detection-application_Model-state.
func (i *ServerImpl) GnmiPostStoreTrafficMonitoringPersonDetectionApplicationModelState(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(StoreTrafficMonitoringPersonDetectionApplicationModelState)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Store-traffic-monitoring_Person-detection-application_Model-state %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiStoreTrafficMonitoringPersonDetectionApplicationModelState(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert StoreTrafficMonitoringPersonDetectionApplicationModelState to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteStoreTrafficMonitoringRetailArea deletes an instance of Store-traffic-monitoring_Retail-area.
func (i *ServerImpl) GnmiDeleteStoreTrafficMonitoringRetailArea(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetStoreTrafficMonitoringRetailArea(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetStoreTrafficMonitoringRetailArea returns an instance of Store-traffic-monitoring_Retail-area.
func (i *ServerImpl) GnmiGetStoreTrafficMonitoringRetailArea(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*StoreTrafficMonitoringRetailArea, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToStoreTrafficMonitoringRetailArea(args...)
}

// GnmiPostStoreTrafficMonitoringRetailArea adds an instance of Store-traffic-monitoring_Retail-area.
func (i *ServerImpl) GnmiPostStoreTrafficMonitoringRetailArea(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(StoreTrafficMonitoringRetailArea)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Store-traffic-monitoring_Retail-area %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiStoreTrafficMonitoringRetailArea(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert StoreTrafficMonitoringRetailArea to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteStoreTrafficMonitoringRetailAreaList deletes an instance of Store-traffic-monitoring_Retail-area_List.
func (i *ServerImpl) GnmiDeleteStoreTrafficMonitoringRetailAreaList(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetStoreTrafficMonitoringRetailAreaList(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetStoreTrafficMonitoringRetailAreaList returns an instance of Store-traffic-monitoring_Retail-area_List.
func (i *ServerImpl) GnmiGetStoreTrafficMonitoringRetailAreaList(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*StoreTrafficMonitoringRetailAreaList, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToStoreTrafficMonitoringRetailAreaList(args...)
}

// GnmiPostStoreTrafficMonitoringRetailAreaList adds an instance of Store-traffic-monitoring_Retail-area_List.
func (i *ServerImpl) GnmiPostStoreTrafficMonitoringRetailAreaList(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(StoreTrafficMonitoringRetailAreaList)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Store-traffic-monitoring_Retail-area_List %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiStoreTrafficMonitoringRetailAreaList(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert StoreTrafficMonitoringRetailAreaList to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiDeleteStoreId deletes an instance of store-id.
func (i *ServerImpl) GnmiDeleteStoreId(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetStoreId(ctx, openApiPath, enterpriseId, args...)
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		log.Infof("Item at path %s with args %v not found", openApiPath, args)
		return nil, echo.NewHTTPError(http.StatusNotFound, fmt.Sprintf("item at path %s with args %v does not exists", openApiPath, args))
	}

	gnmiSet, err := utils.NewGnmiSetDeleteRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}

	return utils.ExtractResponseID(gnmiSetResponse)
}

// GnmiGetStoreId returns an instance of store-id.
func (i *ServerImpl) GnmiGetStoreId(ctx context.Context,
	openApiPath string, enterpriseId StoreId, args ...string) (*StoreId, error) {

	gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiGetRequest %s", gnmiGet.String())
	gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
	if err != nil {
		return nil, err
	}
	if gnmiVal == nil {
		return nil, nil
	}
	gnmiJsonVal, ok := gnmiVal.Value.(*gnmi.TypedValue_JsonVal)
	if !ok {
		return nil, fmt.Errorf("unexpected type of reply from server %v", gnmiVal.Value)
	}

	log.Debugf("gNMI Json %s", string(gnmiJsonVal.JsonVal))
	var gnmiResponse externalRef0.Device
	if err = externalRef0.Unmarshal(gnmiJsonVal.JsonVal, &gnmiResponse); err != nil {
		return nil, fmt.Errorf("error unmarshalling gnmiResponse %v", err)
	}
	mpd := ModelPluginDevice{
		device: gnmiResponse,
	}

	return mpd.ToStoreId(args...)
}

// GnmiPostStoreId adds an instance of store-id.
func (i *ServerImpl) GnmiPostStoreId(ctx context.Context, body []byte,
	openApiPath string, enterpriseId StoreId, args ...string) (*string, error) {

	jsonObj := new(StoreId)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as store-id %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiStoreId(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert StoreId to gNMI %v", err)
	}
	gnmiSet, err := utils.NewGnmiSetUpdateRequestUpdatesDeletes(openApiPath, string(enterpriseId), gnmiUpdates, gnmiDeletes, args...)
	if err != nil {
		return nil, err
	}
	log.Infof("gnmiSetRequest %s", gnmiSet.String())
	gnmiSetResponse, err := i.GnmiClient.Set(ctx, gnmiSet)
	if err != nil {
		return nil, err
	}
	return utils.ExtractResponseID(gnmiSetResponse)
}

//Ignoring RequestBodyRetailArea

//Ignoring RequestBodyRetailAreaLocation

//Ignoring RequestBodyRetailAreaSource

//Ignoring RequestBodyRetailAreaSourceLocation

//Ignoring RequestBodyRetailAreaSourceVideo

//Ignoring RequestBodyShelfMonitoring

//Ignoring RequestBodyShelfMonitoringObjectDetectionApplication

//Ignoring RequestBodyShelfMonitoringRetailArea

//Ignoring RequestBodyShopperMonitoring

//Ignoring RequestBodyShopperMonitoringEmotionRecognitionApplication

//Ignoring RequestBodyShopperMonitoringFaceDetectionApplication

//Ignoring RequestBodyShopperMonitoringHeadPoseDetectionApplication

//Ignoring RequestBodyShopperMonitoringRetailArea

//Ignoring RequestBodyStoreTrafficMonitoring

//Ignoring RequestBodyStoreTrafficMonitoringPersonDetectionApplication

//Ignoring RequestBodyStoreTrafficMonitoringRetailArea

type Translator interface {
	toAdditionalPropertiesUnchTarget(args ...string) (*AdditionalPropertiesUnchTarget, error)
	toAdditionalPropertyStoreId(args ...string) (*AdditionalPropertyStoreId, error)
	toAdditionalPropertyUnchanged(args ...string) (*AdditionalPropertyUnchanged, error) //Ignoring LeafRefOption//Ignoring LeafRefOptions
	toRetailArea(args ...string) (*RetailArea, error)
	toRetailAreaList(args ...string) (*RetailAreaList, error)
	toRetailAreaLocation(args ...string) (*RetailAreaLocation, error)
	toRetailAreaLocationCoordinateSystem(args ...string) (*RetailAreaLocationCoordinateSystem, error)
	toRetailAreaSource(args ...string) (*RetailAreaSource, error)
	toRetailAreaSourceList(args ...string) (*RetailAreaSourceList, error)
	toRetailAreaSourceLocation(args ...string) (*RetailAreaSourceLocation, error)
	toRetailAreaSourceLocationCoordinateSystem(args ...string) (*RetailAreaSourceLocationCoordinateSystem, error)
	toRetailAreaSourceState(args ...string) (*RetailAreaSourceState, error)
	toRetailAreaSourceVideo(args ...string) (*RetailAreaSourceVideo, error)
	toRetailAreaSourceVideoSourceType(args ...string) (*RetailAreaSourceVideoSourceType, error)
	toShelfMonitoring(args ...string) (*ShelfMonitoring, error)
	toShelfMonitoringObjectDetectionApplication(args ...string) (*ShelfMonitoringObjectDetectionApplication, error)
	toShelfMonitoringObjectDetectionApplicationDevice(args ...string) (*ShelfMonitoringObjectDetectionApplicationDevice, error)
	toShelfMonitoringObjectDetectionApplicationPrecision(args ...string) (*ShelfMonitoringObjectDetectionApplicationPrecision, error)
	toShelfMonitoringObjectDetectionApplicationModelState(args ...string) (*ShelfMonitoringObjectDetectionApplicationModelState, error)
	toShelfMonitoringRetailArea(args ...string) (*ShelfMonitoringRetailArea, error)
	toShelfMonitoringRetailAreaList(args ...string) (*ShelfMonitoringRetailAreaList, error)
	toShopperMonitoring(args ...string) (*ShopperMonitoring, error)
	toShopperMonitoringEmotionRecognitionApplication(args ...string) (*ShopperMonitoringEmotionRecognitionApplication, error)
	toShopperMonitoringEmotionRecognitionApplicationDevice(args ...string) (*ShopperMonitoringEmotionRecognitionApplicationDevice, error)
	toShopperMonitoringEmotionRecognitionApplicationPrecision(args ...string) (*ShopperMonitoringEmotionRecognitionApplicationPrecision, error)
	toShopperMonitoringEmotionRecognitionApplicationModelState(args ...string) (*ShopperMonitoringEmotionRecognitionApplicationModelState, error)
	toShopperMonitoringFaceDetectionApplication(args ...string) (*ShopperMonitoringFaceDetectionApplication, error)
	toShopperMonitoringFaceDetectionApplicationDevice(args ...string) (*ShopperMonitoringFaceDetectionApplicationDevice, error)
	toShopperMonitoringFaceDetectionApplicationPrecision(args ...string) (*ShopperMonitoringFaceDetectionApplicationPrecision, error)
	toShopperMonitoringFaceDetectionApplicationModelState(args ...string) (*ShopperMonitoringFaceDetectionApplicationModelState, error)
	toShopperMonitoringHeadPoseDetectionApplication(args ...string) (*ShopperMonitoringHeadPoseDetectionApplication, error)
	toShopperMonitoringHeadPoseDetectionApplicationDevice(args ...string) (*ShopperMonitoringHeadPoseDetectionApplicationDevice, error)
	toShopperMonitoringHeadPoseDetectionApplicationPrecision(args ...string) (*ShopperMonitoringHeadPoseDetectionApplicationPrecision, error)
	toShopperMonitoringHeadPoseDetectionApplicationModelState(args ...string) (*ShopperMonitoringHeadPoseDetectionApplicationModelState, error)
	toShopperMonitoringRetailArea(args ...string) (*ShopperMonitoringRetailArea, error)
	toShopperMonitoringRetailAreaList(args ...string) (*ShopperMonitoringRetailAreaList, error)
	toStoreTrafficMonitoring(args ...string) (*StoreTrafficMonitoring, error)
	toStoreTrafficMonitoringPersonDetectionApplication(args ...string) (*StoreTrafficMonitoringPersonDetectionApplication, error)
	toStoreTrafficMonitoringPersonDetectionApplicationDevice(args ...string) (*StoreTrafficMonitoringPersonDetectionApplicationDevice, error)
	toStoreTrafficMonitoringPersonDetectionApplicationPrecision(args ...string) (*StoreTrafficMonitoringPersonDetectionApplicationPrecision, error)
	toStoreTrafficMonitoringPersonDetectionApplicationModelState(args ...string) (*StoreTrafficMonitoringPersonDetectionApplicationModelState, error)
	toStoreTrafficMonitoringRetailArea(args ...string) (*StoreTrafficMonitoringRetailArea, error)
	toStoreTrafficMonitoringRetailAreaList(args ...string) (*StoreTrafficMonitoringRetailAreaList, error)
	toStoreId(args ...string) (*StoreId, error)
}

func acceptTypes(ctx echo.Context, response *openapi3.T) error {
	acceptType := ctx.Request().Header.Get("Accept")

	if strings.Contains(acceptType, "application/json") {
		return ctx.JSONPretty(http.StatusOK, response, "  ")
	} else if strings.Contains(acceptType, "text/html") {
		templateText, err := ioutil.ReadFile("assets/html-page.tpl")
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "unable to load template %s", err)
		}
		specTemplate, err := htmltemplate.New("spectemplate").Parse(string(templateText))
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "error parsing template %s", err)
		}
		var b bytes.Buffer
		_ = specTemplate.Execute(&b, HTMLData{
			File:        ctx.Request().RequestURI[1:],
			Description: "ROC API",
		})
		ctx.Response().Header().Set("Content-Type", "text/html")
		return ctx.HTMLBlob(http.StatusOK, b.Bytes())
	} else if strings.Contains(acceptType, "application/yaml") || strings.Contains(acceptType, "*/*") {
		jsonFirst, err := json.Marshal(response)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		yamlResp, err := yaml.JSONToYAML(jsonFirst)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err)
		}
		ctx.Response().Header().Set("Content-Type", "application/yaml")
		return ctx.HTMLBlob(http.StatusOK, yamlResp)
	}
	return echo.NewHTTPError(http.StatusNotImplemented,
		fmt.Sprintf("only application/yaml, application/json and text/html encoding supported. "+
			"No match for %s", acceptType))
}

// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

// Not generating param-types

// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

// Not generating request-bodies

// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

// Not generating additional-properties
// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

// Not generating additional-properties
// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

// server-interface template override

// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

const authorization = "Authorization"

// Implement the Server Interface for access to gNMI
var log = logging.GetLogger("model_0_0_0")

// ServerImpl -
type ServerImpl struct {
	GnmiClient               southbound.GnmiClient
	GnmiTimeout              time.Duration
	TopoEndpoint             string
	ConfigAdminServiceClient admin.ConfigAdminServiceClient
}

// GetRetailAreaList impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area
func (i *ServerImpl) GetRetailAreaList(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetRetailAreaList(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetRetailAreaList")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteRetailArea impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}
func (i *ServerImpl) DeleteRetailArea(ctx echo.Context, storeId StoreId, areaId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteRetailArea(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area/{area-id}", storeId, areaId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteRetailArea")
	return ctx.JSON(http.StatusOK, response)
}

// GetRetailArea impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}
func (i *ServerImpl) GetRetailArea(ctx echo.Context, storeId StoreId, areaId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetRetailArea(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area/{area-id}", storeId, areaId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetRetailArea")
	return ctx.JSON(http.StatusOK, response)
}

// PostRetailArea impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}
func (i *ServerImpl) PostRetailArea(ctx echo.Context, storeId StoreId, areaId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostRetailArea(gnmiCtx, body, "/sra/v0.2.x/{store-id}/retail-area/{area-id}", storeId, areaId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostRetailArea")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteRetailAreaLocation impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/location
func (i *ServerImpl) DeleteRetailAreaLocation(ctx echo.Context, storeId StoreId, areaId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteRetailAreaLocation(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/location", storeId, areaId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteRetailAreaLocation")
	return ctx.JSON(http.StatusOK, response)
}

// GetRetailAreaLocation impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/location
func (i *ServerImpl) GetRetailAreaLocation(ctx echo.Context, storeId StoreId, areaId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetRetailAreaLocation(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/location", storeId, areaId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetRetailAreaLocation")
	return ctx.JSON(http.StatusOK, response)
}

// PostRetailAreaLocation impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/location
func (i *ServerImpl) PostRetailAreaLocation(ctx echo.Context, storeId StoreId, areaId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostRetailAreaLocation(gnmiCtx, body, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/location", storeId, areaId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostRetailAreaLocation")
	return ctx.JSON(http.StatusOK, response)
}

// GetRetailAreaSourceList impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/source
func (i *ServerImpl) GetRetailAreaSourceList(ctx echo.Context, storeId StoreId, areaId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetRetailAreaSourceList(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/source", storeId, areaId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetRetailAreaSourceList")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteRetailAreaSource impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}
func (i *ServerImpl) DeleteRetailAreaSource(ctx echo.Context, storeId StoreId, areaId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteRetailAreaSource(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}", storeId, areaId, sourceId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteRetailAreaSource")
	return ctx.JSON(http.StatusOK, response)
}

// GetRetailAreaSource impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}
func (i *ServerImpl) GetRetailAreaSource(ctx echo.Context, storeId StoreId, areaId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetRetailAreaSource(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}", storeId, areaId, sourceId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetRetailAreaSource")
	return ctx.JSON(http.StatusOK, response)
}

// PostRetailAreaSource impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}
func (i *ServerImpl) PostRetailAreaSource(ctx echo.Context, storeId StoreId, areaId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostRetailAreaSource(gnmiCtx, body, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}", storeId, areaId, sourceId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostRetailAreaSource")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteRetailAreaSourceLocation impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/location
func (i *ServerImpl) DeleteRetailAreaSourceLocation(ctx echo.Context, storeId StoreId, areaId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteRetailAreaSourceLocation(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/location", storeId, areaId, sourceId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteRetailAreaSourceLocation")
	return ctx.JSON(http.StatusOK, response)
}

// GetRetailAreaSourceLocation impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/location
func (i *ServerImpl) GetRetailAreaSourceLocation(ctx echo.Context, storeId StoreId, areaId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetRetailAreaSourceLocation(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/location", storeId, areaId, sourceId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetRetailAreaSourceLocation")
	return ctx.JSON(http.StatusOK, response)
}

// PostRetailAreaSourceLocation impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/location
func (i *ServerImpl) PostRetailAreaSourceLocation(ctx echo.Context, storeId StoreId, areaId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostRetailAreaSourceLocation(gnmiCtx, body, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/location", storeId, areaId, sourceId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostRetailAreaSourceLocation")
	return ctx.JSON(http.StatusOK, response)
}

// GetRetailAreaSourceState impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/state
func (i *ServerImpl) GetRetailAreaSourceState(ctx echo.Context, storeId StoreId, areaId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetRetailAreaSourceState(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/state", storeId, areaId, sourceId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetRetailAreaSourceState")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteRetailAreaSourceVideo impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/video
func (i *ServerImpl) DeleteRetailAreaSourceVideo(ctx echo.Context, storeId StoreId, areaId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteRetailAreaSourceVideo(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/video", storeId, areaId, sourceId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteRetailAreaSourceVideo")
	return ctx.JSON(http.StatusOK, response)
}

// GetRetailAreaSourceVideo impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/video
func (i *ServerImpl) GetRetailAreaSourceVideo(ctx echo.Context, storeId StoreId, areaId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetRetailAreaSourceVideo(gnmiCtx, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/video", storeId, areaId, sourceId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetRetailAreaSourceVideo")
	return ctx.JSON(http.StatusOK, response)
}

// PostRetailAreaSourceVideo impl of gNMI access at /sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/video
func (i *ServerImpl) PostRetailAreaSourceVideo(ctx echo.Context, storeId StoreId, areaId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostRetailAreaSourceVideo(gnmiCtx, body, "/sra/v0.2.x/{store-id}/retail-area/{area-id}/source/{source-id}/video", storeId, areaId, sourceId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostRetailAreaSourceVideo")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteShelfMonitoring impl of gNMI access at /sra/v0.2.x/{store-id}/shelf-monitoring
func (i *ServerImpl) DeleteShelfMonitoring(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteShelfMonitoring(gnmiCtx, "/sra/v0.2.x/{store-id}/shelf-monitoring", storeId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteShelfMonitoring")
	return ctx.JSON(http.StatusOK, response)
}

// GetShelfMonitoring impl of gNMI access at /sra/v0.2.x/{store-id}/shelf-monitoring
func (i *ServerImpl) GetShelfMonitoring(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShelfMonitoring(gnmiCtx, "/sra/v0.2.x/{store-id}/shelf-monitoring", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShelfMonitoring")
	return ctx.JSON(http.StatusOK, response)
}

// PostShelfMonitoring impl of gNMI access at /sra/v0.2.x/{store-id}/shelf-monitoring
func (i *ServerImpl) PostShelfMonitoring(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostShelfMonitoring(gnmiCtx, body, "/sra/v0.2.x/{store-id}/shelf-monitoring", storeId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostShelfMonitoring")
	return ctx.JSON(http.StatusOK, response)
}

//Ignoring leafref endpoints

// DeleteShelfMonitoringObjectDetectionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/shelf-monitoring/object-detection-application
func (i *ServerImpl) DeleteShelfMonitoringObjectDetectionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteShelfMonitoringObjectDetectionApplication(gnmiCtx, "/sra/v0.2.x/{store-id}/shelf-monitoring/object-detection-application", storeId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteShelfMonitoringObjectDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetShelfMonitoringObjectDetectionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/shelf-monitoring/object-detection-application
func (i *ServerImpl) GetShelfMonitoringObjectDetectionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShelfMonitoringObjectDetectionApplication(gnmiCtx, "/sra/v0.2.x/{store-id}/shelf-monitoring/object-detection-application", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShelfMonitoringObjectDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// PostShelfMonitoringObjectDetectionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/shelf-monitoring/object-detection-application
func (i *ServerImpl) PostShelfMonitoringObjectDetectionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostShelfMonitoringObjectDetectionApplication(gnmiCtx, body, "/sra/v0.2.x/{store-id}/shelf-monitoring/object-detection-application", storeId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostShelfMonitoringObjectDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetShelfMonitoringObjectDetectionApplicationModelState impl of gNMI access at /sra/v0.2.x/{store-id}/shelf-monitoring/object-detection-application/model-state
func (i *ServerImpl) GetShelfMonitoringObjectDetectionApplicationModelState(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShelfMonitoringObjectDetectionApplicationModelState(gnmiCtx, "/sra/v0.2.x/{store-id}/shelf-monitoring/object-detection-application/model-state", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShelfMonitoringObjectDetectionApplicationModelState")
	return ctx.JSON(http.StatusOK, response)
}

// GetShelfMonitoringRetailAreaList impl of gNMI access at /sra/v0.2.x/{store-id}/shelf-monitoring/retail-area
func (i *ServerImpl) GetShelfMonitoringRetailAreaList(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShelfMonitoringRetailAreaList(gnmiCtx, "/sra/v0.2.x/{store-id}/shelf-monitoring/retail-area", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShelfMonitoringRetailAreaList")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteShelfMonitoringRetailArea impl of gNMI access at /sra/v0.2.x/{store-id}/shelf-monitoring/retail-area/{area-ref}
func (i *ServerImpl) DeleteShelfMonitoringRetailArea(ctx echo.Context, storeId StoreId, areaRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteShelfMonitoringRetailArea(gnmiCtx, "/sra/v0.2.x/{store-id}/shelf-monitoring/retail-area/{area-ref}", storeId, areaRef)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteShelfMonitoringRetailArea")
	return ctx.JSON(http.StatusOK, response)
}

// GetShelfMonitoringRetailArea impl of gNMI access at /sra/v0.2.x/{store-id}/shelf-monitoring/retail-area/{area-ref}
func (i *ServerImpl) GetShelfMonitoringRetailArea(ctx echo.Context, storeId StoreId, areaRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShelfMonitoringRetailArea(gnmiCtx, "/sra/v0.2.x/{store-id}/shelf-monitoring/retail-area/{area-ref}", storeId, areaRef)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShelfMonitoringRetailArea")
	return ctx.JSON(http.StatusOK, response)
}

// PostShelfMonitoringRetailArea impl of gNMI access at /sra/v0.2.x/{store-id}/shelf-monitoring/retail-area/{area-ref}
func (i *ServerImpl) PostShelfMonitoringRetailArea(ctx echo.Context, storeId StoreId, areaRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostShelfMonitoringRetailArea(gnmiCtx, body, "/sra/v0.2.x/{store-id}/shelf-monitoring/retail-area/{area-ref}", storeId, areaRef)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostShelfMonitoringRetailArea")
	return ctx.JSON(http.StatusOK, response)
}

//Ignoring leafref endpoints

// DeleteShopperMonitoring impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring
func (i *ServerImpl) DeleteShopperMonitoring(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteShopperMonitoring(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring", storeId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteShopperMonitoring")
	return ctx.JSON(http.StatusOK, response)
}

// GetShopperMonitoring impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring
func (i *ServerImpl) GetShopperMonitoring(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShopperMonitoring(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShopperMonitoring")
	return ctx.JSON(http.StatusOK, response)
}

// PostShopperMonitoring impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring
func (i *ServerImpl) PostShopperMonitoring(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostShopperMonitoring(gnmiCtx, body, "/sra/v0.2.x/{store-id}/shopper-monitoring", storeId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostShopperMonitoring")
	return ctx.JSON(http.StatusOK, response)
}

//Ignoring leafref endpoints

// DeleteShopperMonitoringEmotionRecognitionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/emotion-recognition-application
func (i *ServerImpl) DeleteShopperMonitoringEmotionRecognitionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteShopperMonitoringEmotionRecognitionApplication(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring/emotion-recognition-application", storeId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteShopperMonitoringEmotionRecognitionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetShopperMonitoringEmotionRecognitionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/emotion-recognition-application
func (i *ServerImpl) GetShopperMonitoringEmotionRecognitionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShopperMonitoringEmotionRecognitionApplication(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring/emotion-recognition-application", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShopperMonitoringEmotionRecognitionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// PostShopperMonitoringEmotionRecognitionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/emotion-recognition-application
func (i *ServerImpl) PostShopperMonitoringEmotionRecognitionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostShopperMonitoringEmotionRecognitionApplication(gnmiCtx, body, "/sra/v0.2.x/{store-id}/shopper-monitoring/emotion-recognition-application", storeId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostShopperMonitoringEmotionRecognitionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetShopperMonitoringEmotionRecognitionApplicationModelState impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/emotion-recognition-application/model-state
func (i *ServerImpl) GetShopperMonitoringEmotionRecognitionApplicationModelState(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShopperMonitoringEmotionRecognitionApplicationModelState(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring/emotion-recognition-application/model-state", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShopperMonitoringEmotionRecognitionApplicationModelState")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteShopperMonitoringFaceDetectionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/face-detection-application
func (i *ServerImpl) DeleteShopperMonitoringFaceDetectionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteShopperMonitoringFaceDetectionApplication(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring/face-detection-application", storeId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteShopperMonitoringFaceDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetShopperMonitoringFaceDetectionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/face-detection-application
func (i *ServerImpl) GetShopperMonitoringFaceDetectionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShopperMonitoringFaceDetectionApplication(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring/face-detection-application", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShopperMonitoringFaceDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// PostShopperMonitoringFaceDetectionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/face-detection-application
func (i *ServerImpl) PostShopperMonitoringFaceDetectionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostShopperMonitoringFaceDetectionApplication(gnmiCtx, body, "/sra/v0.2.x/{store-id}/shopper-monitoring/face-detection-application", storeId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostShopperMonitoringFaceDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetShopperMonitoringFaceDetectionApplicationModelState impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/face-detection-application/model-state
func (i *ServerImpl) GetShopperMonitoringFaceDetectionApplicationModelState(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShopperMonitoringFaceDetectionApplicationModelState(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring/face-detection-application/model-state", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShopperMonitoringFaceDetectionApplicationModelState")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteShopperMonitoringHeadPoseDetectionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/head-pose-detection-application
func (i *ServerImpl) DeleteShopperMonitoringHeadPoseDetectionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteShopperMonitoringHeadPoseDetectionApplication(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring/head-pose-detection-application", storeId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteShopperMonitoringHeadPoseDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetShopperMonitoringHeadPoseDetectionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/head-pose-detection-application
func (i *ServerImpl) GetShopperMonitoringHeadPoseDetectionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShopperMonitoringHeadPoseDetectionApplication(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring/head-pose-detection-application", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShopperMonitoringHeadPoseDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// PostShopperMonitoringHeadPoseDetectionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/head-pose-detection-application
func (i *ServerImpl) PostShopperMonitoringHeadPoseDetectionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostShopperMonitoringHeadPoseDetectionApplication(gnmiCtx, body, "/sra/v0.2.x/{store-id}/shopper-monitoring/head-pose-detection-application", storeId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostShopperMonitoringHeadPoseDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetShopperMonitoringHeadPoseDetectionApplicationModelState impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/head-pose-detection-application/model-state
func (i *ServerImpl) GetShopperMonitoringHeadPoseDetectionApplicationModelState(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShopperMonitoringHeadPoseDetectionApplicationModelState(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring/head-pose-detection-application/model-state", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShopperMonitoringHeadPoseDetectionApplicationModelState")
	return ctx.JSON(http.StatusOK, response)
}

// GetShopperMonitoringRetailAreaList impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/retail-area
func (i *ServerImpl) GetShopperMonitoringRetailAreaList(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShopperMonitoringRetailAreaList(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring/retail-area", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShopperMonitoringRetailAreaList")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteShopperMonitoringRetailArea impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/retail-area/{area-ref}
func (i *ServerImpl) DeleteShopperMonitoringRetailArea(ctx echo.Context, storeId StoreId, areaRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteShopperMonitoringRetailArea(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring/retail-area/{area-ref}", storeId, areaRef)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteShopperMonitoringRetailArea")
	return ctx.JSON(http.StatusOK, response)
}

// GetShopperMonitoringRetailArea impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/retail-area/{area-ref}
func (i *ServerImpl) GetShopperMonitoringRetailArea(ctx echo.Context, storeId StoreId, areaRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetShopperMonitoringRetailArea(gnmiCtx, "/sra/v0.2.x/{store-id}/shopper-monitoring/retail-area/{area-ref}", storeId, areaRef)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetShopperMonitoringRetailArea")
	return ctx.JSON(http.StatusOK, response)
}

// PostShopperMonitoringRetailArea impl of gNMI access at /sra/v0.2.x/{store-id}/shopper-monitoring/retail-area/{area-ref}
func (i *ServerImpl) PostShopperMonitoringRetailArea(ctx echo.Context, storeId StoreId, areaRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostShopperMonitoringRetailArea(gnmiCtx, body, "/sra/v0.2.x/{store-id}/shopper-monitoring/retail-area/{area-ref}", storeId, areaRef)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostShopperMonitoringRetailArea")
	return ctx.JSON(http.StatusOK, response)
}

//Ignoring leafref endpoints

// DeleteStoreTrafficMonitoring impl of gNMI access at /sra/v0.2.x/{store-id}/store-traffic-monitoring
func (i *ServerImpl) DeleteStoreTrafficMonitoring(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteStoreTrafficMonitoring(gnmiCtx, "/sra/v0.2.x/{store-id}/store-traffic-monitoring", storeId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteStoreTrafficMonitoring")
	return ctx.JSON(http.StatusOK, response)
}

// GetStoreTrafficMonitoring impl of gNMI access at /sra/v0.2.x/{store-id}/store-traffic-monitoring
func (i *ServerImpl) GetStoreTrafficMonitoring(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetStoreTrafficMonitoring(gnmiCtx, "/sra/v0.2.x/{store-id}/store-traffic-monitoring", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetStoreTrafficMonitoring")
	return ctx.JSON(http.StatusOK, response)
}

// PostStoreTrafficMonitoring impl of gNMI access at /sra/v0.2.x/{store-id}/store-traffic-monitoring
func (i *ServerImpl) PostStoreTrafficMonitoring(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostStoreTrafficMonitoring(gnmiCtx, body, "/sra/v0.2.x/{store-id}/store-traffic-monitoring", storeId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostStoreTrafficMonitoring")
	return ctx.JSON(http.StatusOK, response)
}

//Ignoring leafref endpoints

// DeleteStoreTrafficMonitoringPersonDetectionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/store-traffic-monitoring/person-detection-application
func (i *ServerImpl) DeleteStoreTrafficMonitoringPersonDetectionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteStoreTrafficMonitoringPersonDetectionApplication(gnmiCtx, "/sra/v0.2.x/{store-id}/store-traffic-monitoring/person-detection-application", storeId)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteStoreTrafficMonitoringPersonDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetStoreTrafficMonitoringPersonDetectionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/store-traffic-monitoring/person-detection-application
func (i *ServerImpl) GetStoreTrafficMonitoringPersonDetectionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetStoreTrafficMonitoringPersonDetectionApplication(gnmiCtx, "/sra/v0.2.x/{store-id}/store-traffic-monitoring/person-detection-application", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetStoreTrafficMonitoringPersonDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// PostStoreTrafficMonitoringPersonDetectionApplication impl of gNMI access at /sra/v0.2.x/{store-id}/store-traffic-monitoring/person-detection-application
func (i *ServerImpl) PostStoreTrafficMonitoringPersonDetectionApplication(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostStoreTrafficMonitoringPersonDetectionApplication(gnmiCtx, body, "/sra/v0.2.x/{store-id}/store-traffic-monitoring/person-detection-application", storeId)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostStoreTrafficMonitoringPersonDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetStoreTrafficMonitoringPersonDetectionApplicationModelState impl of gNMI access at /sra/v0.2.x/{store-id}/store-traffic-monitoring/person-detection-application/model-state
func (i *ServerImpl) GetStoreTrafficMonitoringPersonDetectionApplicationModelState(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetStoreTrafficMonitoringPersonDetectionApplicationModelState(gnmiCtx, "/sra/v0.2.x/{store-id}/store-traffic-monitoring/person-detection-application/model-state", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetStoreTrafficMonitoringPersonDetectionApplicationModelState")
	return ctx.JSON(http.StatusOK, response)
}

// GetStoreTrafficMonitoringRetailAreaList impl of gNMI access at /sra/v0.2.x/{store-id}/store-traffic-monitoring/retail-area
func (i *ServerImpl) GetStoreTrafficMonitoringRetailAreaList(ctx echo.Context, storeId StoreId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetStoreTrafficMonitoringRetailAreaList(gnmiCtx, "/sra/v0.2.x/{store-id}/store-traffic-monitoring/retail-area", storeId)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetStoreTrafficMonitoringRetailAreaList")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteStoreTrafficMonitoringRetailArea impl of gNMI access at /sra/v0.2.x/{store-id}/store-traffic-monitoring/retail-area/{area-ref}
func (i *ServerImpl) DeleteStoreTrafficMonitoringRetailArea(ctx echo.Context, storeId StoreId, areaRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteStoreTrafficMonitoringRetailArea(gnmiCtx, "/sra/v0.2.x/{store-id}/store-traffic-monitoring/retail-area/{area-ref}", storeId, areaRef)
	if err == nil {
		log.Infof("Delete succeded %s", *extension100)
		return ctx.JSON(http.StatusOK, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("DeleteStoreTrafficMonitoringRetailArea")
	return ctx.JSON(http.StatusOK, response)
}

// GetStoreTrafficMonitoringRetailArea impl of gNMI access at /sra/v0.2.x/{store-id}/store-traffic-monitoring/retail-area/{area-ref}
func (i *ServerImpl) GetStoreTrafficMonitoringRetailArea(ctx echo.Context, storeId StoreId, areaRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetStoreTrafficMonitoringRetailArea(gnmiCtx, "/sra/v0.2.x/{store-id}/store-traffic-monitoring/retail-area/{area-ref}", storeId, areaRef)

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("GetStoreTrafficMonitoringRetailArea")
	return ctx.JSON(http.StatusOK, response)
}

// PostStoreTrafficMonitoringRetailArea impl of gNMI access at /sra/v0.2.x/{store-id}/store-traffic-monitoring/retail-area/{area-ref}
func (i *ServerImpl) PostStoreTrafficMonitoringRetailArea(ctx echo.Context, storeId StoreId, areaRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostStoreTrafficMonitoringRetailArea(gnmiCtx, body, "/sra/v0.2.x/{store-id}/store-traffic-monitoring/retail-area/{area-ref}", storeId, areaRef)
	if err == nil {
		log.Infof("Post succeded %s", *extension100)
		return ctx.JSON(http.StatusCreated, extension100)
	}

	if err != nil {
		httpErr := utils.ConvertGrpcError(err)
		if httpErr == echo.ErrNotFound {
			return ctx.NoContent(http.StatusNotFound)
		}
		return httpErr
	}
	// It's not enough to check if response==nil - see https://medium.com/@glucn/golang-an-interface-holding-a-nil-value-is-not-nil-bb151f472cc7
	if reflect.ValueOf(response).Kind() == reflect.Ptr && reflect.ValueOf(response).IsNil() {
		return ctx.NoContent(http.StatusNotFound)
	}

	log.Infof("PostStoreTrafficMonitoringRetailArea")
	return ctx.JSON(http.StatusOK, response)
}

//Ignoring leafref endpoints

// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

// register template override

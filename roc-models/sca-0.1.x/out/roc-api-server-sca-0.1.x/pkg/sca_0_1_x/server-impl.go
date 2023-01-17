// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

// imports template override
package sca_0_1_x

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
	externalRef0 "github.com/intel-innersource/frameworks.edge.one-intel-edge.springboard.reference-implementation.roc-models/models/sca-0.1.x/api"
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

//Ignoring AdditionalPropertyCityId

//Ignoring AdditionalPropertyUnchanged

// GnmiDeleteCollisionDetection deletes an instance of Collision-detection.
func (i *ServerImpl) GnmiDeleteCollisionDetection(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetCollisionDetection(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetCollisionDetection returns an instance of Collision-detection.
func (i *ServerImpl) GnmiGetCollisionDetection(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*CollisionDetection, error) {

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

	return mpd.ToCollisionDetection(args...)
}

// GnmiPostCollisionDetection adds an instance of Collision-detection.
func (i *ServerImpl) GnmiPostCollisionDetection(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(CollisionDetection)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Collision-detection %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiCollisionDetection(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert CollisionDetection to gNMI %v", err)
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

// GnmiDeleteCollisionDetectionDetectionApplication deletes an instance of Collision-detection_Detection-application.
func (i *ServerImpl) GnmiDeleteCollisionDetectionDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetCollisionDetectionDetectionApplication(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetCollisionDetectionDetectionApplication returns an instance of Collision-detection_Detection-application.
func (i *ServerImpl) GnmiGetCollisionDetectionDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*CollisionDetectionDetectionApplication, error) {

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

	return mpd.ToCollisionDetectionDetectionApplication(args...)
}

// GnmiPostCollisionDetectionDetectionApplication adds an instance of Collision-detection_Detection-application.
func (i *ServerImpl) GnmiPostCollisionDetectionDetectionApplication(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(CollisionDetectionDetectionApplication)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Collision-detection_Detection-application %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiCollisionDetectionDetectionApplication(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert CollisionDetectionDetectionApplication to gNMI %v", err)
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

// GnmiDeleteCollisionDetectionDetectionApplicationDevice deletes an instance of CollisionDetectionDetectionApplication.Device.
func (i *ServerImpl) GnmiDeleteCollisionDetectionDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetCollisionDetectionDetectionApplicationDevice(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetCollisionDetectionDetectionApplicationDevice returns an instance of CollisionDetectionDetectionApplication.Device.
func (i *ServerImpl) GnmiGetCollisionDetectionDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*CollisionDetectionDetectionApplicationDevice, error) {

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

	return mpd.ToCollisionDetectionDetectionApplicationDevice(args...)
}

// GnmiPostCollisionDetectionDetectionApplicationDevice adds an instance of CollisionDetectionDetectionApplication.Device.
func (i *ServerImpl) GnmiPostCollisionDetectionDetectionApplicationDevice(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(CollisionDetectionDetectionApplicationDevice)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as CollisionDetectionDetectionApplication.Device %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiCollisionDetectionDetectionApplicationDevice(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert CollisionDetectionDetectionApplicationDevice to gNMI %v", err)
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

// GnmiDeleteCollisionDetectionDetectionApplicationPrecision deletes an instance of CollisionDetectionDetectionApplication.Precision.
func (i *ServerImpl) GnmiDeleteCollisionDetectionDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetCollisionDetectionDetectionApplicationPrecision(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetCollisionDetectionDetectionApplicationPrecision returns an instance of CollisionDetectionDetectionApplication.Precision.
func (i *ServerImpl) GnmiGetCollisionDetectionDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*CollisionDetectionDetectionApplicationPrecision, error) {

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

	return mpd.ToCollisionDetectionDetectionApplicationPrecision(args...)
}

// GnmiPostCollisionDetectionDetectionApplicationPrecision adds an instance of CollisionDetectionDetectionApplication.Precision.
func (i *ServerImpl) GnmiPostCollisionDetectionDetectionApplicationPrecision(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(CollisionDetectionDetectionApplicationPrecision)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as CollisionDetectionDetectionApplication.Precision %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiCollisionDetectionDetectionApplicationPrecision(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert CollisionDetectionDetectionApplicationPrecision to gNMI %v", err)
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

// GnmiDeleteCollisionDetectionDetectionApplicationModelState deletes an instance of Collision-detection_Detection-application_Model-state.
func (i *ServerImpl) GnmiDeleteCollisionDetectionDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetCollisionDetectionDetectionApplicationModelState(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetCollisionDetectionDetectionApplicationModelState returns an instance of Collision-detection_Detection-application_Model-state.
func (i *ServerImpl) GnmiGetCollisionDetectionDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*CollisionDetectionDetectionApplicationModelState, error) {

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

	return mpd.ToCollisionDetectionDetectionApplicationModelState(args...)
}

// GnmiPostCollisionDetectionDetectionApplicationModelState adds an instance of Collision-detection_Detection-application_Model-state.
func (i *ServerImpl) GnmiPostCollisionDetectionDetectionApplicationModelState(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(CollisionDetectionDetectionApplicationModelState)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Collision-detection_Detection-application_Model-state %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiCollisionDetectionDetectionApplicationModelState(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert CollisionDetectionDetectionApplicationModelState to gNMI %v", err)
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

// GnmiDeleteCollisionDetectionDistrict deletes an instance of Collision-detection_District.
func (i *ServerImpl) GnmiDeleteCollisionDetectionDistrict(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetCollisionDetectionDistrict(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetCollisionDetectionDistrict returns an instance of Collision-detection_District.
func (i *ServerImpl) GnmiGetCollisionDetectionDistrict(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*CollisionDetectionDistrict, error) {

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

	return mpd.ToCollisionDetectionDistrict(args...)
}

// GnmiPostCollisionDetectionDistrict adds an instance of Collision-detection_District.
func (i *ServerImpl) GnmiPostCollisionDetectionDistrict(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(CollisionDetectionDistrict)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Collision-detection_District %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiCollisionDetectionDistrict(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert CollisionDetectionDistrict to gNMI %v", err)
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

// GnmiDeleteCollisionDetectionDistrictList deletes an instance of Collision-detection_District_List.
func (i *ServerImpl) GnmiDeleteCollisionDetectionDistrictList(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetCollisionDetectionDistrictList(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetCollisionDetectionDistrictList returns an instance of Collision-detection_District_List.
func (i *ServerImpl) GnmiGetCollisionDetectionDistrictList(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*CollisionDetectionDistrictList, error) {

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

	return mpd.ToCollisionDetectionDistrictList(args...)
}

// GnmiPostCollisionDetectionDistrictList adds an instance of Collision-detection_District_List.
func (i *ServerImpl) GnmiPostCollisionDetectionDistrictList(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(CollisionDetectionDistrictList)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Collision-detection_District_List %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiCollisionDetectionDistrictList(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert CollisionDetectionDistrictList to gNMI %v", err)
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

// GnmiDeleteDistrict deletes an instance of District.
func (i *ServerImpl) GnmiDeleteDistrict(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetDistrict(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetDistrict returns an instance of District.
func (i *ServerImpl) GnmiGetDistrict(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*District, error) {

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

	return mpd.ToDistrict(args...)
}

// GnmiPostDistrict adds an instance of District.
func (i *ServerImpl) GnmiPostDistrict(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(District)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as District %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiDistrict(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert District to gNMI %v", err)
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

// GnmiDeleteDistrictList deletes an instance of District_List.
func (i *ServerImpl) GnmiDeleteDistrictList(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetDistrictList(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetDistrictList returns an instance of District_List.
func (i *ServerImpl) GnmiGetDistrictList(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*DistrictList, error) {

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

	return mpd.ToDistrictList(args...)
}

// GnmiPostDistrictList adds an instance of District_List.
func (i *ServerImpl) GnmiPostDistrictList(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(DistrictList)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as District_List %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiDistrictList(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert DistrictList to gNMI %v", err)
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

// GnmiDeleteDistrictLocation deletes an instance of District_Location.
func (i *ServerImpl) GnmiDeleteDistrictLocation(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetDistrictLocation(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetDistrictLocation returns an instance of District_Location.
func (i *ServerImpl) GnmiGetDistrictLocation(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*DistrictLocation, error) {

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

	return mpd.ToDistrictLocation(args...)
}

// GnmiPostDistrictLocation adds an instance of District_Location.
func (i *ServerImpl) GnmiPostDistrictLocation(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(DistrictLocation)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as District_Location %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiDistrictLocation(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert DistrictLocation to gNMI %v", err)
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

// GnmiDeleteDistrictLocationCoordinateSystem deletes an instance of DistrictLocation.CoordinateSystem.
func (i *ServerImpl) GnmiDeleteDistrictLocationCoordinateSystem(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetDistrictLocationCoordinateSystem(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetDistrictLocationCoordinateSystem returns an instance of DistrictLocation.CoordinateSystem.
func (i *ServerImpl) GnmiGetDistrictLocationCoordinateSystem(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*DistrictLocationCoordinateSystem, error) {

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

	return mpd.ToDistrictLocationCoordinateSystem(args...)
}

// GnmiPostDistrictLocationCoordinateSystem adds an instance of DistrictLocation.CoordinateSystem.
func (i *ServerImpl) GnmiPostDistrictLocationCoordinateSystem(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(DistrictLocationCoordinateSystem)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as DistrictLocation.CoordinateSystem %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiDistrictLocationCoordinateSystem(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert DistrictLocationCoordinateSystem to gNMI %v", err)
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

// GnmiDeleteDistrictSource deletes an instance of District_Source.
func (i *ServerImpl) GnmiDeleteDistrictSource(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetDistrictSource(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetDistrictSource returns an instance of District_Source.
func (i *ServerImpl) GnmiGetDistrictSource(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*DistrictSource, error) {

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

	return mpd.ToDistrictSource(args...)
}

// GnmiPostDistrictSource adds an instance of District_Source.
func (i *ServerImpl) GnmiPostDistrictSource(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(DistrictSource)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as District_Source %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiDistrictSource(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert DistrictSource to gNMI %v", err)
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

// GnmiDeleteDistrictSourceList deletes an instance of District_Source_List.
func (i *ServerImpl) GnmiDeleteDistrictSourceList(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetDistrictSourceList(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetDistrictSourceList returns an instance of District_Source_List.
func (i *ServerImpl) GnmiGetDistrictSourceList(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*DistrictSourceList, error) {

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

	return mpd.ToDistrictSourceList(args...)
}

// GnmiPostDistrictSourceList adds an instance of District_Source_List.
func (i *ServerImpl) GnmiPostDistrictSourceList(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(DistrictSourceList)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as District_Source_List %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiDistrictSourceList(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert DistrictSourceList to gNMI %v", err)
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

// GnmiDeleteDistrictSourceLocation deletes an instance of District_Source_Location.
func (i *ServerImpl) GnmiDeleteDistrictSourceLocation(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetDistrictSourceLocation(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetDistrictSourceLocation returns an instance of District_Source_Location.
func (i *ServerImpl) GnmiGetDistrictSourceLocation(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*DistrictSourceLocation, error) {

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

	return mpd.ToDistrictSourceLocation(args...)
}

// GnmiPostDistrictSourceLocation adds an instance of District_Source_Location.
func (i *ServerImpl) GnmiPostDistrictSourceLocation(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(DistrictSourceLocation)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as District_Source_Location %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiDistrictSourceLocation(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert DistrictSourceLocation to gNMI %v", err)
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

// GnmiDeleteDistrictSourceLocationCoordinateSystem deletes an instance of DistrictSourceLocation.CoordinateSystem.
func (i *ServerImpl) GnmiDeleteDistrictSourceLocationCoordinateSystem(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetDistrictSourceLocationCoordinateSystem(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetDistrictSourceLocationCoordinateSystem returns an instance of DistrictSourceLocation.CoordinateSystem.
func (i *ServerImpl) GnmiGetDistrictSourceLocationCoordinateSystem(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*DistrictSourceLocationCoordinateSystem, error) {

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

	return mpd.ToDistrictSourceLocationCoordinateSystem(args...)
}

// GnmiPostDistrictSourceLocationCoordinateSystem adds an instance of DistrictSourceLocation.CoordinateSystem.
func (i *ServerImpl) GnmiPostDistrictSourceLocationCoordinateSystem(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(DistrictSourceLocationCoordinateSystem)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as DistrictSourceLocation.CoordinateSystem %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiDistrictSourceLocationCoordinateSystem(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert DistrictSourceLocationCoordinateSystem to gNMI %v", err)
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

// GnmiDeleteDistrictSourceState deletes an instance of District_Source_State.
func (i *ServerImpl) GnmiDeleteDistrictSourceState(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetDistrictSourceState(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetDistrictSourceState returns an instance of District_Source_State.
func (i *ServerImpl) GnmiGetDistrictSourceState(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*DistrictSourceState, error) {

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

	return mpd.ToDistrictSourceState(args...)
}

// GnmiPostDistrictSourceState adds an instance of District_Source_State.
func (i *ServerImpl) GnmiPostDistrictSourceState(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(DistrictSourceState)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as District_Source_State %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiDistrictSourceState(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert DistrictSourceState to gNMI %v", err)
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

// GnmiDeleteDistrictSourceVideo deletes an instance of District_Source_Video.
func (i *ServerImpl) GnmiDeleteDistrictSourceVideo(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetDistrictSourceVideo(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetDistrictSourceVideo returns an instance of District_Source_Video.
func (i *ServerImpl) GnmiGetDistrictSourceVideo(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*DistrictSourceVideo, error) {

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

	return mpd.ToDistrictSourceVideo(args...)
}

// GnmiPostDistrictSourceVideo adds an instance of District_Source_Video.
func (i *ServerImpl) GnmiPostDistrictSourceVideo(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(DistrictSourceVideo)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as District_Source_Video %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiDistrictSourceVideo(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert DistrictSourceVideo to gNMI %v", err)
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

// GnmiDeleteDistrictSourceVideoSourceType deletes an instance of DistrictSourceVideo.SourceType.
func (i *ServerImpl) GnmiDeleteDistrictSourceVideoSourceType(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetDistrictSourceVideoSourceType(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetDistrictSourceVideoSourceType returns an instance of DistrictSourceVideo.SourceType.
func (i *ServerImpl) GnmiGetDistrictSourceVideoSourceType(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*DistrictSourceVideoSourceType, error) {

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

	return mpd.ToDistrictSourceVideoSourceType(args...)
}

// GnmiPostDistrictSourceVideoSourceType adds an instance of DistrictSourceVideo.SourceType.
func (i *ServerImpl) GnmiPostDistrictSourceVideoSourceType(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(DistrictSourceVideoSourceType)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as DistrictSourceVideo.SourceType %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiDistrictSourceVideoSourceType(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert DistrictSourceVideoSourceType to gNMI %v", err)
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

//Ignoring LeafRefOption

//Ignoring LeafRefOptions

// GnmiDeleteTrafficClassification deletes an instance of Traffic-classification.
func (i *ServerImpl) GnmiDeleteTrafficClassification(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficClassification(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficClassification returns an instance of Traffic-classification.
func (i *ServerImpl) GnmiGetTrafficClassification(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficClassification, error) {

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

	return mpd.ToTrafficClassification(args...)
}

// GnmiPostTrafficClassification adds an instance of Traffic-classification.
func (i *ServerImpl) GnmiPostTrafficClassification(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficClassification)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Traffic-classification %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficClassification(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficClassification to gNMI %v", err)
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

// GnmiDeleteTrafficClassificationClassificationApplication deletes an instance of Traffic-classification_Classification-application.
func (i *ServerImpl) GnmiDeleteTrafficClassificationClassificationApplication(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficClassificationClassificationApplication(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficClassificationClassificationApplication returns an instance of Traffic-classification_Classification-application.
func (i *ServerImpl) GnmiGetTrafficClassificationClassificationApplication(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficClassificationClassificationApplication, error) {

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

	return mpd.ToTrafficClassificationClassificationApplication(args...)
}

// GnmiPostTrafficClassificationClassificationApplication adds an instance of Traffic-classification_Classification-application.
func (i *ServerImpl) GnmiPostTrafficClassificationClassificationApplication(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficClassificationClassificationApplication)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Traffic-classification_Classification-application %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficClassificationClassificationApplication(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficClassificationClassificationApplication to gNMI %v", err)
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

// GnmiDeleteTrafficClassificationClassificationApplicationDevice deletes an instance of TrafficClassificationClassificationApplication.Device.
func (i *ServerImpl) GnmiDeleteTrafficClassificationClassificationApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficClassificationClassificationApplicationDevice(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficClassificationClassificationApplicationDevice returns an instance of TrafficClassificationClassificationApplication.Device.
func (i *ServerImpl) GnmiGetTrafficClassificationClassificationApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficClassificationClassificationApplicationDevice, error) {

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

	return mpd.ToTrafficClassificationClassificationApplicationDevice(args...)
}

// GnmiPostTrafficClassificationClassificationApplicationDevice adds an instance of TrafficClassificationClassificationApplication.Device.
func (i *ServerImpl) GnmiPostTrafficClassificationClassificationApplicationDevice(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficClassificationClassificationApplicationDevice)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as TrafficClassificationClassificationApplication.Device %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficClassificationClassificationApplicationDevice(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficClassificationClassificationApplicationDevice to gNMI %v", err)
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

// GnmiDeleteTrafficClassificationClassificationApplicationPrecision deletes an instance of TrafficClassificationClassificationApplication.Precision.
func (i *ServerImpl) GnmiDeleteTrafficClassificationClassificationApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficClassificationClassificationApplicationPrecision(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficClassificationClassificationApplicationPrecision returns an instance of TrafficClassificationClassificationApplication.Precision.
func (i *ServerImpl) GnmiGetTrafficClassificationClassificationApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficClassificationClassificationApplicationPrecision, error) {

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

	return mpd.ToTrafficClassificationClassificationApplicationPrecision(args...)
}

// GnmiPostTrafficClassificationClassificationApplicationPrecision adds an instance of TrafficClassificationClassificationApplication.Precision.
func (i *ServerImpl) GnmiPostTrafficClassificationClassificationApplicationPrecision(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficClassificationClassificationApplicationPrecision)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as TrafficClassificationClassificationApplication.Precision %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficClassificationClassificationApplicationPrecision(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficClassificationClassificationApplicationPrecision to gNMI %v", err)
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

// GnmiDeleteTrafficClassificationClassificationApplicationModelState deletes an instance of Traffic-classification_Classification-application_Model-state.
func (i *ServerImpl) GnmiDeleteTrafficClassificationClassificationApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficClassificationClassificationApplicationModelState(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficClassificationClassificationApplicationModelState returns an instance of Traffic-classification_Classification-application_Model-state.
func (i *ServerImpl) GnmiGetTrafficClassificationClassificationApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficClassificationClassificationApplicationModelState, error) {

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

	return mpd.ToTrafficClassificationClassificationApplicationModelState(args...)
}

// GnmiPostTrafficClassificationClassificationApplicationModelState adds an instance of Traffic-classification_Classification-application_Model-state.
func (i *ServerImpl) GnmiPostTrafficClassificationClassificationApplicationModelState(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficClassificationClassificationApplicationModelState)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Traffic-classification_Classification-application_Model-state %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficClassificationClassificationApplicationModelState(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficClassificationClassificationApplicationModelState to gNMI %v", err)
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

// GnmiDeleteTrafficClassificationDetectionApplication deletes an instance of Traffic-classification_Detection-application.
func (i *ServerImpl) GnmiDeleteTrafficClassificationDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficClassificationDetectionApplication(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficClassificationDetectionApplication returns an instance of Traffic-classification_Detection-application.
func (i *ServerImpl) GnmiGetTrafficClassificationDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficClassificationDetectionApplication, error) {

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

	return mpd.ToTrafficClassificationDetectionApplication(args...)
}

// GnmiPostTrafficClassificationDetectionApplication adds an instance of Traffic-classification_Detection-application.
func (i *ServerImpl) GnmiPostTrafficClassificationDetectionApplication(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficClassificationDetectionApplication)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Traffic-classification_Detection-application %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficClassificationDetectionApplication(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficClassificationDetectionApplication to gNMI %v", err)
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

// GnmiDeleteTrafficClassificationDetectionApplicationDevice deletes an instance of TrafficClassificationDetectionApplication.Device.
func (i *ServerImpl) GnmiDeleteTrafficClassificationDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficClassificationDetectionApplicationDevice(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficClassificationDetectionApplicationDevice returns an instance of TrafficClassificationDetectionApplication.Device.
func (i *ServerImpl) GnmiGetTrafficClassificationDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficClassificationDetectionApplicationDevice, error) {

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

	return mpd.ToTrafficClassificationDetectionApplicationDevice(args...)
}

// GnmiPostTrafficClassificationDetectionApplicationDevice adds an instance of TrafficClassificationDetectionApplication.Device.
func (i *ServerImpl) GnmiPostTrafficClassificationDetectionApplicationDevice(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficClassificationDetectionApplicationDevice)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as TrafficClassificationDetectionApplication.Device %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficClassificationDetectionApplicationDevice(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficClassificationDetectionApplicationDevice to gNMI %v", err)
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

// GnmiDeleteTrafficClassificationDetectionApplicationPrecision deletes an instance of TrafficClassificationDetectionApplication.Precision.
func (i *ServerImpl) GnmiDeleteTrafficClassificationDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficClassificationDetectionApplicationPrecision(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficClassificationDetectionApplicationPrecision returns an instance of TrafficClassificationDetectionApplication.Precision.
func (i *ServerImpl) GnmiGetTrafficClassificationDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficClassificationDetectionApplicationPrecision, error) {

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

	return mpd.ToTrafficClassificationDetectionApplicationPrecision(args...)
}

// GnmiPostTrafficClassificationDetectionApplicationPrecision adds an instance of TrafficClassificationDetectionApplication.Precision.
func (i *ServerImpl) GnmiPostTrafficClassificationDetectionApplicationPrecision(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficClassificationDetectionApplicationPrecision)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as TrafficClassificationDetectionApplication.Precision %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficClassificationDetectionApplicationPrecision(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficClassificationDetectionApplicationPrecision to gNMI %v", err)
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

// GnmiDeleteTrafficClassificationDetectionApplicationModelState deletes an instance of Traffic-classification_Detection-application_Model-state.
func (i *ServerImpl) GnmiDeleteTrafficClassificationDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficClassificationDetectionApplicationModelState(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficClassificationDetectionApplicationModelState returns an instance of Traffic-classification_Detection-application_Model-state.
func (i *ServerImpl) GnmiGetTrafficClassificationDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficClassificationDetectionApplicationModelState, error) {

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

	return mpd.ToTrafficClassificationDetectionApplicationModelState(args...)
}

// GnmiPostTrafficClassificationDetectionApplicationModelState adds an instance of Traffic-classification_Detection-application_Model-state.
func (i *ServerImpl) GnmiPostTrafficClassificationDetectionApplicationModelState(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficClassificationDetectionApplicationModelState)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Traffic-classification_Detection-application_Model-state %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficClassificationDetectionApplicationModelState(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficClassificationDetectionApplicationModelState to gNMI %v", err)
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

// GnmiDeleteTrafficClassificationDistrict deletes an instance of Traffic-classification_District.
func (i *ServerImpl) GnmiDeleteTrafficClassificationDistrict(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficClassificationDistrict(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficClassificationDistrict returns an instance of Traffic-classification_District.
func (i *ServerImpl) GnmiGetTrafficClassificationDistrict(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficClassificationDistrict, error) {

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

	return mpd.ToTrafficClassificationDistrict(args...)
}

// GnmiPostTrafficClassificationDistrict adds an instance of Traffic-classification_District.
func (i *ServerImpl) GnmiPostTrafficClassificationDistrict(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficClassificationDistrict)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Traffic-classification_District %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficClassificationDistrict(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficClassificationDistrict to gNMI %v", err)
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

// GnmiDeleteTrafficClassificationDistrictList deletes an instance of Traffic-classification_District_List.
func (i *ServerImpl) GnmiDeleteTrafficClassificationDistrictList(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficClassificationDistrictList(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficClassificationDistrictList returns an instance of Traffic-classification_District_List.
func (i *ServerImpl) GnmiGetTrafficClassificationDistrictList(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficClassificationDistrictList, error) {

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

	return mpd.ToTrafficClassificationDistrictList(args...)
}

// GnmiPostTrafficClassificationDistrictList adds an instance of Traffic-classification_District_List.
func (i *ServerImpl) GnmiPostTrafficClassificationDistrictList(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficClassificationDistrictList)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Traffic-classification_District_List %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficClassificationDistrictList(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficClassificationDistrictList to gNMI %v", err)
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

// GnmiDeleteTrafficMonitoring deletes an instance of Traffic-monitoring.
func (i *ServerImpl) GnmiDeleteTrafficMonitoring(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficMonitoring(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficMonitoring returns an instance of Traffic-monitoring.
func (i *ServerImpl) GnmiGetTrafficMonitoring(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficMonitoring, error) {

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

	return mpd.ToTrafficMonitoring(args...)
}

// GnmiPostTrafficMonitoring adds an instance of Traffic-monitoring.
func (i *ServerImpl) GnmiPostTrafficMonitoring(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficMonitoring)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Traffic-monitoring %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficMonitoring(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficMonitoring to gNMI %v", err)
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

// GnmiDeleteTrafficMonitoringDistrict deletes an instance of Traffic-monitoring_District.
func (i *ServerImpl) GnmiDeleteTrafficMonitoringDistrict(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficMonitoringDistrict(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficMonitoringDistrict returns an instance of Traffic-monitoring_District.
func (i *ServerImpl) GnmiGetTrafficMonitoringDistrict(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficMonitoringDistrict, error) {

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

	return mpd.ToTrafficMonitoringDistrict(args...)
}

// GnmiPostTrafficMonitoringDistrict adds an instance of Traffic-monitoring_District.
func (i *ServerImpl) GnmiPostTrafficMonitoringDistrict(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficMonitoringDistrict)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Traffic-monitoring_District %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficMonitoringDistrict(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficMonitoringDistrict to gNMI %v", err)
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

// GnmiDeleteTrafficMonitoringDistrictList deletes an instance of Traffic-monitoring_District_List.
func (i *ServerImpl) GnmiDeleteTrafficMonitoringDistrictList(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficMonitoringDistrictList(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficMonitoringDistrictList returns an instance of Traffic-monitoring_District_List.
func (i *ServerImpl) GnmiGetTrafficMonitoringDistrictList(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficMonitoringDistrictList, error) {

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

	return mpd.ToTrafficMonitoringDistrictList(args...)
}

// GnmiPostTrafficMonitoringDistrictList adds an instance of Traffic-monitoring_District_List.
func (i *ServerImpl) GnmiPostTrafficMonitoringDistrictList(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficMonitoringDistrictList)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Traffic-monitoring_District_List %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficMonitoringDistrictList(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficMonitoringDistrictList to gNMI %v", err)
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

// GnmiDeleteTrafficMonitoringPersonVehicleBikeDetectionApplication deletes an instance of Traffic-monitoring_Person-vehicle-bike-detection-application.
func (i *ServerImpl) GnmiDeleteTrafficMonitoringPersonVehicleBikeDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplication(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplication returns an instance of Traffic-monitoring_Person-vehicle-bike-detection-application.
func (i *ServerImpl) GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplication(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficMonitoringPersonVehicleBikeDetectionApplication, error) {

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

	return mpd.ToTrafficMonitoringPersonVehicleBikeDetectionApplication(args...)
}

// GnmiPostTrafficMonitoringPersonVehicleBikeDetectionApplication adds an instance of Traffic-monitoring_Person-vehicle-bike-detection-application.
func (i *ServerImpl) GnmiPostTrafficMonitoringPersonVehicleBikeDetectionApplication(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficMonitoringPersonVehicleBikeDetectionApplication)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Traffic-monitoring_Person-vehicle-bike-detection-application %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficMonitoringPersonVehicleBikeDetectionApplication(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficMonitoringPersonVehicleBikeDetectionApplication to gNMI %v", err)
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

// GnmiDeleteTrafficMonitoringPersonVehicleBikeDetectionApplicationDevice deletes an instance of TrafficMonitoringPersonVehicleBikeDetectionApplication.Device.
func (i *ServerImpl) GnmiDeleteTrafficMonitoringPersonVehicleBikeDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplicationDevice(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplicationDevice returns an instance of TrafficMonitoringPersonVehicleBikeDetectionApplication.Device.
func (i *ServerImpl) GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplicationDevice(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficMonitoringPersonVehicleBikeDetectionApplicationDevice, error) {

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

	return mpd.ToTrafficMonitoringPersonVehicleBikeDetectionApplicationDevice(args...)
}

// GnmiPostTrafficMonitoringPersonVehicleBikeDetectionApplicationDevice adds an instance of TrafficMonitoringPersonVehicleBikeDetectionApplication.Device.
func (i *ServerImpl) GnmiPostTrafficMonitoringPersonVehicleBikeDetectionApplicationDevice(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficMonitoringPersonVehicleBikeDetectionApplicationDevice)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as TrafficMonitoringPersonVehicleBikeDetectionApplication.Device %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficMonitoringPersonVehicleBikeDetectionApplicationDevice(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficMonitoringPersonVehicleBikeDetectionApplicationDevice to gNMI %v", err)
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

// GnmiDeleteTrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision deletes an instance of TrafficMonitoringPersonVehicleBikeDetectionApplication.Precision.
func (i *ServerImpl) GnmiDeleteTrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision returns an instance of TrafficMonitoringPersonVehicleBikeDetectionApplication.Precision.
func (i *ServerImpl) GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision, error) {

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

	return mpd.ToTrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision(args...)
}

// GnmiPostTrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision adds an instance of TrafficMonitoringPersonVehicleBikeDetectionApplication.Precision.
func (i *ServerImpl) GnmiPostTrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as TrafficMonitoringPersonVehicleBikeDetectionApplication.Precision %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision to gNMI %v", err)
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

// GnmiDeleteTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState deletes an instance of Traffic-monitoring_Person-vehicle-bike-detection-application_Model-state.
func (i *ServerImpl) GnmiDeleteTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState returns an instance of Traffic-monitoring_Person-vehicle-bike-detection-application_Model-state.
func (i *ServerImpl) GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*TrafficMonitoringPersonVehicleBikeDetectionApplicationModelState, error) {

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

	return mpd.ToTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState(args...)
}

// GnmiPostTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState adds an instance of Traffic-monitoring_Person-vehicle-bike-detection-application_Model-state.
func (i *ServerImpl) GnmiPostTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(TrafficMonitoringPersonVehicleBikeDetectionApplicationModelState)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as Traffic-monitoring_Person-vehicle-bike-detection-application_Model-state %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert TrafficMonitoringPersonVehicleBikeDetectionApplicationModelState to gNMI %v", err)
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

// GnmiDeleteCityId deletes an instance of city-id.
func (i *ServerImpl) GnmiDeleteCityId(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	// check to see if the item exists before deleting it
	response, err := i.GnmiGetCityId(ctx, openApiPath, enterpriseId, args...)
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

// GnmiGetCityId returns an instance of city-id.
func (i *ServerImpl) GnmiGetCityId(ctx context.Context,
	openApiPath string, enterpriseId CityId, args ...string) (*CityId, error) {

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

	return mpd.ToCityId(args...)
}

// GnmiPostCityId adds an instance of city-id.
func (i *ServerImpl) GnmiPostCityId(ctx context.Context, body []byte,
	openApiPath string, enterpriseId CityId, args ...string) (*string, error) {

	jsonObj := new(CityId)
	if err := json.Unmarshal(body, jsonObj); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON as city-id %v", err)
	}
	gnmiUpdates, gnmiDeletes, err := EncodeToGnmiCityId(jsonObj, false, false, enterpriseId, "", args...)
	if err != nil {
		return nil, fmt.Errorf("unable to convert CityId to gNMI %v", err)
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

//Ignoring RequestBodyCollisionDetection

//Ignoring RequestBodyCollisionDetectionDetectionApplication

//Ignoring RequestBodyCollisionDetectionDistrict

//Ignoring RequestBodyDistrict

//Ignoring RequestBodyDistrictLocation

//Ignoring RequestBodyDistrictSource

//Ignoring RequestBodyDistrictSourceLocation

//Ignoring RequestBodyDistrictSourceVideo

//Ignoring RequestBodyTrafficClassification

//Ignoring RequestBodyTrafficClassificationClassificationApplication

//Ignoring RequestBodyTrafficClassificationDetectionApplication

//Ignoring RequestBodyTrafficClassificationDistrict

//Ignoring RequestBodyTrafficMonitoring

//Ignoring RequestBodyTrafficMonitoringDistrict

//Ignoring RequestBodyTrafficMonitoringPersonVehicleBikeDetectionApplication

type Translator interface {
	toAdditionalPropertiesUnchTarget(args ...string) (*AdditionalPropertiesUnchTarget, error)
	toAdditionalPropertyCityId(args ...string) (*AdditionalPropertyCityId, error)
	toAdditionalPropertyUnchanged(args ...string) (*AdditionalPropertyUnchanged, error)
	toCollisionDetection(args ...string) (*CollisionDetection, error)
	toCollisionDetectionDetectionApplication(args ...string) (*CollisionDetectionDetectionApplication, error)
	toCollisionDetectionDetectionApplicationDevice(args ...string) (*CollisionDetectionDetectionApplicationDevice, error)
	toCollisionDetectionDetectionApplicationPrecision(args ...string) (*CollisionDetectionDetectionApplicationPrecision, error)
	toCollisionDetectionDetectionApplicationModelState(args ...string) (*CollisionDetectionDetectionApplicationModelState, error)
	toCollisionDetectionDistrict(args ...string) (*CollisionDetectionDistrict, error)
	toCollisionDetectionDistrictList(args ...string) (*CollisionDetectionDistrictList, error)
	toDistrict(args ...string) (*District, error)
	toDistrictList(args ...string) (*DistrictList, error)
	toDistrictLocation(args ...string) (*DistrictLocation, error)
	toDistrictLocationCoordinateSystem(args ...string) (*DistrictLocationCoordinateSystem, error)
	toDistrictSource(args ...string) (*DistrictSource, error)
	toDistrictSourceList(args ...string) (*DistrictSourceList, error)
	toDistrictSourceLocation(args ...string) (*DistrictSourceLocation, error)
	toDistrictSourceLocationCoordinateSystem(args ...string) (*DistrictSourceLocationCoordinateSystem, error)
	toDistrictSourceState(args ...string) (*DistrictSourceState, error)
	toDistrictSourceVideo(args ...string) (*DistrictSourceVideo, error)
	toDistrictSourceVideoSourceType(args ...string) (*DistrictSourceVideoSourceType, error) //Ignoring LeafRefOption//Ignoring LeafRefOptions
	toTrafficClassification(args ...string) (*TrafficClassification, error)
	toTrafficClassificationClassificationApplication(args ...string) (*TrafficClassificationClassificationApplication, error)
	toTrafficClassificationClassificationApplicationDevice(args ...string) (*TrafficClassificationClassificationApplicationDevice, error)
	toTrafficClassificationClassificationApplicationPrecision(args ...string) (*TrafficClassificationClassificationApplicationPrecision, error)
	toTrafficClassificationClassificationApplicationModelState(args ...string) (*TrafficClassificationClassificationApplicationModelState, error)
	toTrafficClassificationDetectionApplication(args ...string) (*TrafficClassificationDetectionApplication, error)
	toTrafficClassificationDetectionApplicationDevice(args ...string) (*TrafficClassificationDetectionApplicationDevice, error)
	toTrafficClassificationDetectionApplicationPrecision(args ...string) (*TrafficClassificationDetectionApplicationPrecision, error)
	toTrafficClassificationDetectionApplicationModelState(args ...string) (*TrafficClassificationDetectionApplicationModelState, error)
	toTrafficClassificationDistrict(args ...string) (*TrafficClassificationDistrict, error)
	toTrafficClassificationDistrictList(args ...string) (*TrafficClassificationDistrictList, error)
	toTrafficMonitoring(args ...string) (*TrafficMonitoring, error)
	toTrafficMonitoringDistrict(args ...string) (*TrafficMonitoringDistrict, error)
	toTrafficMonitoringDistrictList(args ...string) (*TrafficMonitoringDistrictList, error)
	toTrafficMonitoringPersonVehicleBikeDetectionApplication(args ...string) (*TrafficMonitoringPersonVehicleBikeDetectionApplication, error)
	toTrafficMonitoringPersonVehicleBikeDetectionApplicationDevice(args ...string) (*TrafficMonitoringPersonVehicleBikeDetectionApplicationDevice, error)
	toTrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision(args ...string) (*TrafficMonitoringPersonVehicleBikeDetectionApplicationPrecision, error)
	toTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState(args ...string) (*TrafficMonitoringPersonVehicleBikeDetectionApplicationModelState, error)
	toCityId(args ...string) (*CityId, error)
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

// DeleteCollisionDetection impl of gNMI access at /sca/v0.1.x/{city-id}/collision-detection
func (i *ServerImpl) DeleteCollisionDetection(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteCollisionDetection(gnmiCtx, "/sca/v0.1.x/{city-id}/collision-detection", cityId)
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

	log.Infof("DeleteCollisionDetection")
	return ctx.JSON(http.StatusOK, response)
}

// GetCollisionDetection impl of gNMI access at /sca/v0.1.x/{city-id}/collision-detection
func (i *ServerImpl) GetCollisionDetection(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetCollisionDetection(gnmiCtx, "/sca/v0.1.x/{city-id}/collision-detection", cityId)

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

	log.Infof("GetCollisionDetection")
	return ctx.JSON(http.StatusOK, response)
}

// PostCollisionDetection impl of gNMI access at /sca/v0.1.x/{city-id}/collision-detection
func (i *ServerImpl) PostCollisionDetection(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostCollisionDetection(gnmiCtx, body, "/sca/v0.1.x/{city-id}/collision-detection", cityId)
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

	log.Infof("PostCollisionDetection")
	return ctx.JSON(http.StatusOK, response)
}

//Ignoring leafref endpoints

// DeleteCollisionDetectionDetectionApplication impl of gNMI access at /sca/v0.1.x/{city-id}/collision-detection/detection-application
func (i *ServerImpl) DeleteCollisionDetectionDetectionApplication(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteCollisionDetectionDetectionApplication(gnmiCtx, "/sca/v0.1.x/{city-id}/collision-detection/detection-application", cityId)
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

	log.Infof("DeleteCollisionDetectionDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetCollisionDetectionDetectionApplication impl of gNMI access at /sca/v0.1.x/{city-id}/collision-detection/detection-application
func (i *ServerImpl) GetCollisionDetectionDetectionApplication(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetCollisionDetectionDetectionApplication(gnmiCtx, "/sca/v0.1.x/{city-id}/collision-detection/detection-application", cityId)

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

	log.Infof("GetCollisionDetectionDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// PostCollisionDetectionDetectionApplication impl of gNMI access at /sca/v0.1.x/{city-id}/collision-detection/detection-application
func (i *ServerImpl) PostCollisionDetectionDetectionApplication(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostCollisionDetectionDetectionApplication(gnmiCtx, body, "/sca/v0.1.x/{city-id}/collision-detection/detection-application", cityId)
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

	log.Infof("PostCollisionDetectionDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetCollisionDetectionDetectionApplicationModelState impl of gNMI access at /sca/v0.1.x/{city-id}/collision-detection/detection-application/model-state
func (i *ServerImpl) GetCollisionDetectionDetectionApplicationModelState(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetCollisionDetectionDetectionApplicationModelState(gnmiCtx, "/sca/v0.1.x/{city-id}/collision-detection/detection-application/model-state", cityId)

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

	log.Infof("GetCollisionDetectionDetectionApplicationModelState")
	return ctx.JSON(http.StatusOK, response)
}

// GetCollisionDetectionDistrictList impl of gNMI access at /sca/v0.1.x/{city-id}/collision-detection/district
func (i *ServerImpl) GetCollisionDetectionDistrictList(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetCollisionDetectionDistrictList(gnmiCtx, "/sca/v0.1.x/{city-id}/collision-detection/district", cityId)

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

	log.Infof("GetCollisionDetectionDistrictList")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteCollisionDetectionDistrict impl of gNMI access at /sca/v0.1.x/{city-id}/collision-detection/district/{district-ref}
func (i *ServerImpl) DeleteCollisionDetectionDistrict(ctx echo.Context, cityId CityId, districtRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteCollisionDetectionDistrict(gnmiCtx, "/sca/v0.1.x/{city-id}/collision-detection/district/{district-ref}", cityId, districtRef)
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

	log.Infof("DeleteCollisionDetectionDistrict")
	return ctx.JSON(http.StatusOK, response)
}

// GetCollisionDetectionDistrict impl of gNMI access at /sca/v0.1.x/{city-id}/collision-detection/district/{district-ref}
func (i *ServerImpl) GetCollisionDetectionDistrict(ctx echo.Context, cityId CityId, districtRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetCollisionDetectionDistrict(gnmiCtx, "/sca/v0.1.x/{city-id}/collision-detection/district/{district-ref}", cityId, districtRef)

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

	log.Infof("GetCollisionDetectionDistrict")
	return ctx.JSON(http.StatusOK, response)
}

// PostCollisionDetectionDistrict impl of gNMI access at /sca/v0.1.x/{city-id}/collision-detection/district/{district-ref}
func (i *ServerImpl) PostCollisionDetectionDistrict(ctx echo.Context, cityId CityId, districtRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostCollisionDetectionDistrict(gnmiCtx, body, "/sca/v0.1.x/{city-id}/collision-detection/district/{district-ref}", cityId, districtRef)
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

	log.Infof("PostCollisionDetectionDistrict")
	return ctx.JSON(http.StatusOK, response)
}

//Ignoring leafref endpoints

// GetDistrictList impl of gNMI access at /sca/v0.1.x/{city-id}/district
func (i *ServerImpl) GetDistrictList(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetDistrictList(gnmiCtx, "/sca/v0.1.x/{city-id}/district", cityId)

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

	log.Infof("GetDistrictList")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteDistrict impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}
func (i *ServerImpl) DeleteDistrict(ctx echo.Context, cityId CityId, districtId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteDistrict(gnmiCtx, "/sca/v0.1.x/{city-id}/district/{district-id}", cityId, districtId)
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

	log.Infof("DeleteDistrict")
	return ctx.JSON(http.StatusOK, response)
}

// GetDistrict impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}
func (i *ServerImpl) GetDistrict(ctx echo.Context, cityId CityId, districtId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetDistrict(gnmiCtx, "/sca/v0.1.x/{city-id}/district/{district-id}", cityId, districtId)

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

	log.Infof("GetDistrict")
	return ctx.JSON(http.StatusOK, response)
}

// PostDistrict impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}
func (i *ServerImpl) PostDistrict(ctx echo.Context, cityId CityId, districtId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostDistrict(gnmiCtx, body, "/sca/v0.1.x/{city-id}/district/{district-id}", cityId, districtId)
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

	log.Infof("PostDistrict")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteDistrictLocation impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/location
func (i *ServerImpl) DeleteDistrictLocation(ctx echo.Context, cityId CityId, districtId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteDistrictLocation(gnmiCtx, "/sca/v0.1.x/{city-id}/district/{district-id}/location", cityId, districtId)
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

	log.Infof("DeleteDistrictLocation")
	return ctx.JSON(http.StatusOK, response)
}

// GetDistrictLocation impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/location
func (i *ServerImpl) GetDistrictLocation(ctx echo.Context, cityId CityId, districtId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetDistrictLocation(gnmiCtx, "/sca/v0.1.x/{city-id}/district/{district-id}/location", cityId, districtId)

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

	log.Infof("GetDistrictLocation")
	return ctx.JSON(http.StatusOK, response)
}

// PostDistrictLocation impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/location
func (i *ServerImpl) PostDistrictLocation(ctx echo.Context, cityId CityId, districtId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostDistrictLocation(gnmiCtx, body, "/sca/v0.1.x/{city-id}/district/{district-id}/location", cityId, districtId)
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

	log.Infof("PostDistrictLocation")
	return ctx.JSON(http.StatusOK, response)
}

// GetDistrictSourceList impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/source
func (i *ServerImpl) GetDistrictSourceList(ctx echo.Context, cityId CityId, districtId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetDistrictSourceList(gnmiCtx, "/sca/v0.1.x/{city-id}/district/{district-id}/source", cityId, districtId)

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

	log.Infof("GetDistrictSourceList")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteDistrictSource impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}
func (i *ServerImpl) DeleteDistrictSource(ctx echo.Context, cityId CityId, districtId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteDistrictSource(gnmiCtx, "/sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}", cityId, districtId, sourceId)
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

	log.Infof("DeleteDistrictSource")
	return ctx.JSON(http.StatusOK, response)
}

// GetDistrictSource impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}
func (i *ServerImpl) GetDistrictSource(ctx echo.Context, cityId CityId, districtId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetDistrictSource(gnmiCtx, "/sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}", cityId, districtId, sourceId)

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

	log.Infof("GetDistrictSource")
	return ctx.JSON(http.StatusOK, response)
}

// PostDistrictSource impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}
func (i *ServerImpl) PostDistrictSource(ctx echo.Context, cityId CityId, districtId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostDistrictSource(gnmiCtx, body, "/sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}", cityId, districtId, sourceId)
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

	log.Infof("PostDistrictSource")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteDistrictSourceLocation impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/location
func (i *ServerImpl) DeleteDistrictSourceLocation(ctx echo.Context, cityId CityId, districtId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteDistrictSourceLocation(gnmiCtx, "/sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/location", cityId, districtId, sourceId)
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

	log.Infof("DeleteDistrictSourceLocation")
	return ctx.JSON(http.StatusOK, response)
}

// GetDistrictSourceLocation impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/location
func (i *ServerImpl) GetDistrictSourceLocation(ctx echo.Context, cityId CityId, districtId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetDistrictSourceLocation(gnmiCtx, "/sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/location", cityId, districtId, sourceId)

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

	log.Infof("GetDistrictSourceLocation")
	return ctx.JSON(http.StatusOK, response)
}

// PostDistrictSourceLocation impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/location
func (i *ServerImpl) PostDistrictSourceLocation(ctx echo.Context, cityId CityId, districtId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostDistrictSourceLocation(gnmiCtx, body, "/sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/location", cityId, districtId, sourceId)
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

	log.Infof("PostDistrictSourceLocation")
	return ctx.JSON(http.StatusOK, response)
}

// GetDistrictSourceState impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/state
func (i *ServerImpl) GetDistrictSourceState(ctx echo.Context, cityId CityId, districtId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetDistrictSourceState(gnmiCtx, "/sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/state", cityId, districtId, sourceId)

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

	log.Infof("GetDistrictSourceState")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteDistrictSourceVideo impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/video
func (i *ServerImpl) DeleteDistrictSourceVideo(ctx echo.Context, cityId CityId, districtId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteDistrictSourceVideo(gnmiCtx, "/sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/video", cityId, districtId, sourceId)
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

	log.Infof("DeleteDistrictSourceVideo")
	return ctx.JSON(http.StatusOK, response)
}

// GetDistrictSourceVideo impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/video
func (i *ServerImpl) GetDistrictSourceVideo(ctx echo.Context, cityId CityId, districtId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetDistrictSourceVideo(gnmiCtx, "/sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/video", cityId, districtId, sourceId)

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

	log.Infof("GetDistrictSourceVideo")
	return ctx.JSON(http.StatusOK, response)
}

// PostDistrictSourceVideo impl of gNMI access at /sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/video
func (i *ServerImpl) PostDistrictSourceVideo(ctx echo.Context, cityId CityId, districtId string, sourceId string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostDistrictSourceVideo(gnmiCtx, body, "/sca/v0.1.x/{city-id}/district/{district-id}/source/{source-id}/video", cityId, districtId, sourceId)
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

	log.Infof("PostDistrictSourceVideo")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteTrafficClassification impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification
func (i *ServerImpl) DeleteTrafficClassification(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteTrafficClassification(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-classification", cityId)
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

	log.Infof("DeleteTrafficClassification")
	return ctx.JSON(http.StatusOK, response)
}

// GetTrafficClassification impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification
func (i *ServerImpl) GetTrafficClassification(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetTrafficClassification(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-classification", cityId)

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

	log.Infof("GetTrafficClassification")
	return ctx.JSON(http.StatusOK, response)
}

// PostTrafficClassification impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification
func (i *ServerImpl) PostTrafficClassification(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostTrafficClassification(gnmiCtx, body, "/sca/v0.1.x/{city-id}/traffic-classification", cityId)
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

	log.Infof("PostTrafficClassification")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteTrafficClassificationClassificationApplication impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification/classification-application
func (i *ServerImpl) DeleteTrafficClassificationClassificationApplication(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteTrafficClassificationClassificationApplication(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-classification/classification-application", cityId)
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

	log.Infof("DeleteTrafficClassificationClassificationApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetTrafficClassificationClassificationApplication impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification/classification-application
func (i *ServerImpl) GetTrafficClassificationClassificationApplication(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetTrafficClassificationClassificationApplication(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-classification/classification-application", cityId)

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

	log.Infof("GetTrafficClassificationClassificationApplication")
	return ctx.JSON(http.StatusOK, response)
}

// PostTrafficClassificationClassificationApplication impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification/classification-application
func (i *ServerImpl) PostTrafficClassificationClassificationApplication(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostTrafficClassificationClassificationApplication(gnmiCtx, body, "/sca/v0.1.x/{city-id}/traffic-classification/classification-application", cityId)
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

	log.Infof("PostTrafficClassificationClassificationApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetTrafficClassificationClassificationApplicationModelState impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification/classification-application/model-state
func (i *ServerImpl) GetTrafficClassificationClassificationApplicationModelState(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetTrafficClassificationClassificationApplicationModelState(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-classification/classification-application/model-state", cityId)

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

	log.Infof("GetTrafficClassificationClassificationApplicationModelState")
	return ctx.JSON(http.StatusOK, response)
}

//Ignoring leafref endpoints

// DeleteTrafficClassificationDetectionApplication impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification/detection-application
func (i *ServerImpl) DeleteTrafficClassificationDetectionApplication(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteTrafficClassificationDetectionApplication(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-classification/detection-application", cityId)
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

	log.Infof("DeleteTrafficClassificationDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetTrafficClassificationDetectionApplication impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification/detection-application
func (i *ServerImpl) GetTrafficClassificationDetectionApplication(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetTrafficClassificationDetectionApplication(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-classification/detection-application", cityId)

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

	log.Infof("GetTrafficClassificationDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// PostTrafficClassificationDetectionApplication impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification/detection-application
func (i *ServerImpl) PostTrafficClassificationDetectionApplication(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostTrafficClassificationDetectionApplication(gnmiCtx, body, "/sca/v0.1.x/{city-id}/traffic-classification/detection-application", cityId)
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

	log.Infof("PostTrafficClassificationDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetTrafficClassificationDetectionApplicationModelState impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification/detection-application/model-state
func (i *ServerImpl) GetTrafficClassificationDetectionApplicationModelState(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetTrafficClassificationDetectionApplicationModelState(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-classification/detection-application/model-state", cityId)

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

	log.Infof("GetTrafficClassificationDetectionApplicationModelState")
	return ctx.JSON(http.StatusOK, response)
}

// GetTrafficClassificationDistrictList impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification/district
func (i *ServerImpl) GetTrafficClassificationDistrictList(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetTrafficClassificationDistrictList(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-classification/district", cityId)

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

	log.Infof("GetTrafficClassificationDistrictList")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteTrafficClassificationDistrict impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification/district/{district-ref}
func (i *ServerImpl) DeleteTrafficClassificationDistrict(ctx echo.Context, cityId CityId, districtRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteTrafficClassificationDistrict(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-classification/district/{district-ref}", cityId, districtRef)
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

	log.Infof("DeleteTrafficClassificationDistrict")
	return ctx.JSON(http.StatusOK, response)
}

// GetTrafficClassificationDistrict impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification/district/{district-ref}
func (i *ServerImpl) GetTrafficClassificationDistrict(ctx echo.Context, cityId CityId, districtRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetTrafficClassificationDistrict(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-classification/district/{district-ref}", cityId, districtRef)

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

	log.Infof("GetTrafficClassificationDistrict")
	return ctx.JSON(http.StatusOK, response)
}

// PostTrafficClassificationDistrict impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-classification/district/{district-ref}
func (i *ServerImpl) PostTrafficClassificationDistrict(ctx echo.Context, cityId CityId, districtRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostTrafficClassificationDistrict(gnmiCtx, body, "/sca/v0.1.x/{city-id}/traffic-classification/district/{district-ref}", cityId, districtRef)
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

	log.Infof("PostTrafficClassificationDistrict")
	return ctx.JSON(http.StatusOK, response)
}

//Ignoring leafref endpoints

// DeleteTrafficMonitoring impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-monitoring
func (i *ServerImpl) DeleteTrafficMonitoring(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteTrafficMonitoring(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-monitoring", cityId)
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

	log.Infof("DeleteTrafficMonitoring")
	return ctx.JSON(http.StatusOK, response)
}

// GetTrafficMonitoring impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-monitoring
func (i *ServerImpl) GetTrafficMonitoring(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetTrafficMonitoring(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-monitoring", cityId)

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

	log.Infof("GetTrafficMonitoring")
	return ctx.JSON(http.StatusOK, response)
}

// PostTrafficMonitoring impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-monitoring
func (i *ServerImpl) PostTrafficMonitoring(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostTrafficMonitoring(gnmiCtx, body, "/sca/v0.1.x/{city-id}/traffic-monitoring", cityId)
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

	log.Infof("PostTrafficMonitoring")
	return ctx.JSON(http.StatusOK, response)
}

//Ignoring leafref endpoints

// GetTrafficMonitoringDistrictList impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-monitoring/district
func (i *ServerImpl) GetTrafficMonitoringDistrictList(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetTrafficMonitoringDistrictList(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-monitoring/district", cityId)

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

	log.Infof("GetTrafficMonitoringDistrictList")
	return ctx.JSON(http.StatusOK, response)
}

// DeleteTrafficMonitoringDistrict impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-monitoring/district/{district-ref}
func (i *ServerImpl) DeleteTrafficMonitoringDistrict(ctx echo.Context, cityId CityId, districtRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteTrafficMonitoringDistrict(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-monitoring/district/{district-ref}", cityId, districtRef)
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

	log.Infof("DeleteTrafficMonitoringDistrict")
	return ctx.JSON(http.StatusOK, response)
}

// GetTrafficMonitoringDistrict impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-monitoring/district/{district-ref}
func (i *ServerImpl) GetTrafficMonitoringDistrict(ctx echo.Context, cityId CityId, districtRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetTrafficMonitoringDistrict(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-monitoring/district/{district-ref}", cityId, districtRef)

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

	log.Infof("GetTrafficMonitoringDistrict")
	return ctx.JSON(http.StatusOK, response)
}

// PostTrafficMonitoringDistrict impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-monitoring/district/{district-ref}
func (i *ServerImpl) PostTrafficMonitoringDistrict(ctx echo.Context, cityId CityId, districtRef string) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostTrafficMonitoringDistrict(gnmiCtx, body, "/sca/v0.1.x/{city-id}/traffic-monitoring/district/{district-ref}", cityId, districtRef)
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

	log.Infof("PostTrafficMonitoringDistrict")
	return ctx.JSON(http.StatusOK, response)
}

//Ignoring leafref endpoints

// DeleteTrafficMonitoringPersonVehicleBikeDetectionApplication impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-monitoring/person-vehicle-bike-detection-application
func (i *ServerImpl) DeleteTrafficMonitoringPersonVehicleBikeDetectionApplication(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response DELETE 200 OK
	extension100, err := i.GnmiDeleteTrafficMonitoringPersonVehicleBikeDetectionApplication(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-monitoring/person-vehicle-bike-detection-application", cityId)
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

	log.Infof("DeleteTrafficMonitoringPersonVehicleBikeDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetTrafficMonitoringPersonVehicleBikeDetectionApplication impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-monitoring/person-vehicle-bike-detection-application
func (i *ServerImpl) GetTrafficMonitoringPersonVehicleBikeDetectionApplication(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplication(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-monitoring/person-vehicle-bike-detection-application", cityId)

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

	log.Infof("GetTrafficMonitoringPersonVehicleBikeDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// PostTrafficMonitoringPersonVehicleBikeDetectionApplication impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-monitoring/person-vehicle-bike-detection-application
func (i *ServerImpl) PostTrafficMonitoringPersonVehicleBikeDetectionApplication(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response created

	body, err := utils.ReadRequestBody(ctx.Request().Body)
	if err != nil {
		return err
	}
	extension100, err := i.GnmiPostTrafficMonitoringPersonVehicleBikeDetectionApplication(gnmiCtx, body, "/sca/v0.1.x/{city-id}/traffic-monitoring/person-vehicle-bike-detection-application", cityId)
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

	log.Infof("PostTrafficMonitoringPersonVehicleBikeDetectionApplication")
	return ctx.JSON(http.StatusOK, response)
}

// GetTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState impl of gNMI access at /sca/v0.1.x/{city-id}/traffic-monitoring/person-vehicle-bike-detection-application/model-state
func (i *ServerImpl) GetTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState(ctx echo.Context, cityId CityId) error {

	var response interface{}
	var err error

	gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
	defer cancel()

	// Response GET OK 200
	response, err = i.GnmiGetTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState(gnmiCtx, "/sca/v0.1.x/{city-id}/traffic-monitoring/person-vehicle-bike-detection-application/model-state", cityId)

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

	log.Infof("GetTrafficMonitoringPersonVehicleBikeDetectionApplicationModelState")
	return ctx.JSON(http.StatusOK, response)
}

// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

// register template override

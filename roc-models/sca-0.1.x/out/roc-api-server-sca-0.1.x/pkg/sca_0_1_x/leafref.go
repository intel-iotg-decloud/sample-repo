// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package sca_0_1_x

import (
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/onosproject/aether-roc-api/pkg/utils"
	externalRef0 "github.com/intel-innersource/frameworks.edge.one-intel-edge.springboard.reference-implementation.roc-models/models/sca-0.1.x/api"
	"github.com/openconfig/gnmi/proto/gnmi"
	"golang.org/x/exp/maps"
	"net/http"
	"reflect"
	"strings"
)

func iteratePath(refValue reflect.Value, path []string) map[string]string {
	resList := make(map[string]string)
    if refValue == (reflect.Value{}) {
        return resList
    }
	switch refValue.Kind() {
	case reflect.Ptr:
		refValue = refValue.Elem()
		maps.Copy(resList, iteratePath(refValue, path))
	case reflect.Slice:
		for i := 0; i < refValue.Len(); i++ {
			val := refValue.Index(i)
			maps.Copy(resList, iteratePath(val, path))
		}
	case reflect.Struct:
		dir := utils.ToCamelCase(path[0])
		refValue = refValue.FieldByName(dir)
		maps.Copy(resList, iteratePath(refValue, path[1:]))
	default:
		leafIdValue := fmt.Sprint(refValue)
		resList[leafIdValue] = leafIdValue
	}

	return resList
}



func DistrictListToLeafRefOptions (districtListVar  *DistrictList, targetPath string, args ...string) (*LeafRefOptions, error) {
    response := new(LeafRefOptions)
    if targetPath == "" {
    	log.Error("No target path is defined for the leafref")
    	return nil, fmt.Errorf("no target path is defined for the leafref")
    }
    splitTargetPath := strings.Split(targetPath, "/")
    v := reflect.Indirect(reflect.ValueOf(districtListVar))
    resList := iteratePath(v, splitTargetPath)
    for _, val := range resList {
        leafVal := new(string)
        *leafVal = val
        *response = append(*response, LeafRefOption{
            Label: leafVal,
            Value: leafVal,
        })
    }
    return response, nil
}




func (i *ServerImpl) GnmiGetCollisionDetectionDistrictDistrictRefValuesLeafref (ctx context.Context, openApiPath string, enterpriseId CityId , args ...string) (*LeafRefOptions, error) {
    gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
    if err != nil {
        return nil, err
    }
    log.Infof("gnmiGetRequest %s", gnmiGet.String())
    gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
    if err != nil {
        log.Info("getresponse update error: ", err)
        return nil, err
    }
    if gnmiVal == nil {
        log.Info("gnmiVal is empty")
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

    response, err := mpd.ToDistrictList(args[:0]...)
    if err != nil {
        log.Info("error unmarshaling to DistrictList: ", err)
        return nil, err
    }

    return DistrictListToLeafRefOptions(response, "district-id", args...)

}


func (i *ServerImpl) GetCollisionDetectionDistrictDistrictRefValuesLeafref (ctx echo.Context, cityId CityId, districtRef string) error {
    var response interface{}
    var err error

    gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
    defer cancel()

    // Response GET OK 200
    response, err = i.GnmiGetCollisionDetectionDistrictDistrictRefValuesLeafref(gnmiCtx, "/sca/v0.1.x/{city-id}/district", cityId, districtRef)
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

    return ctx.JSON(http.StatusOK, response)
}


func (i *ServerImpl) GnmiGetCollisionDetectionDefaultValuesLeafref (ctx context.Context, openApiPath string, enterpriseId CityId , args ...string) (*LeafRefOptions, error) {
    gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
    if err != nil {
        return nil, err
    }
    log.Infof("gnmiGetRequest %s", gnmiGet.String())
    gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
    if err != nil {
        log.Info("getresponse update error: ", err)
        return nil, err
    }
    if gnmiVal == nil {
        log.Info("gnmiVal is empty")
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

    response, err := mpd.ToDistrictList(args[:0]...)
    if err != nil {
        log.Info("error unmarshaling to DistrictList: ", err)
        return nil, err
    }

    return DistrictListToLeafRefOptions(response, "source/source-id", args...)

}


func (i *ServerImpl) GetCollisionDetectionDefaultValuesLeafref (ctx echo.Context, cityId CityId) error {
    var response interface{}
    var err error

    gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
    defer cancel()

    // Response GET OK 200
    response, err = i.GnmiGetCollisionDetectionDefaultValuesLeafref(gnmiCtx, "/sca/v0.1.x/{city-id}/district", cityId)
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

    return ctx.JSON(http.StatusOK, response)
}


func (i *ServerImpl) GnmiGetTrafficMonitoringDefaultValuesLeafref (ctx context.Context, openApiPath string, enterpriseId CityId , args ...string) (*LeafRefOptions, error) {
    gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
    if err != nil {
        return nil, err
    }
    log.Infof("gnmiGetRequest %s", gnmiGet.String())
    gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
    if err != nil {
        log.Info("getresponse update error: ", err)
        return nil, err
    }
    if gnmiVal == nil {
        log.Info("gnmiVal is empty")
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

    response, err := mpd.ToDistrictList(args[:0]...)
    if err != nil {
        log.Info("error unmarshaling to DistrictList: ", err)
        return nil, err
    }

    return DistrictListToLeafRefOptions(response, "source/source-id", args...)

}


func (i *ServerImpl) GetTrafficMonitoringDefaultValuesLeafref (ctx echo.Context, cityId CityId) error {
    var response interface{}
    var err error

    gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
    defer cancel()

    // Response GET OK 200
    response, err = i.GnmiGetTrafficMonitoringDefaultValuesLeafref(gnmiCtx, "/sca/v0.1.x/{city-id}/district", cityId)
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

    return ctx.JSON(http.StatusOK, response)
}


func (i *ServerImpl) GnmiGetTrafficClassificationDefaultValuesLeafref (ctx context.Context, openApiPath string, enterpriseId CityId , args ...string) (*LeafRefOptions, error) {
    gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
    if err != nil {
        return nil, err
    }
    log.Infof("gnmiGetRequest %s", gnmiGet.String())
    gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
    if err != nil {
        log.Info("getresponse update error: ", err)
        return nil, err
    }
    if gnmiVal == nil {
        log.Info("gnmiVal is empty")
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

    response, err := mpd.ToDistrictList(args[:0]...)
    if err != nil {
        log.Info("error unmarshaling to DistrictList: ", err)
        return nil, err
    }

    return DistrictListToLeafRefOptions(response, "source/source-id", args...)

}


func (i *ServerImpl) GetTrafficClassificationDefaultValuesLeafref (ctx echo.Context, cityId CityId) error {
    var response interface{}
    var err error

    gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
    defer cancel()

    // Response GET OK 200
    response, err = i.GnmiGetTrafficClassificationDefaultValuesLeafref(gnmiCtx, "/sca/v0.1.x/{city-id}/district", cityId)
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

    return ctx.JSON(http.StatusOK, response)
}


func (i *ServerImpl) GnmiGetTrafficClassificationDistrictDistrictRefValuesLeafref (ctx context.Context, openApiPath string, enterpriseId CityId , args ...string) (*LeafRefOptions, error) {
    gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
    if err != nil {
        return nil, err
    }
    log.Infof("gnmiGetRequest %s", gnmiGet.String())
    gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
    if err != nil {
        log.Info("getresponse update error: ", err)
        return nil, err
    }
    if gnmiVal == nil {
        log.Info("gnmiVal is empty")
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

    response, err := mpd.ToDistrictList(args[:0]...)
    if err != nil {
        log.Info("error unmarshaling to DistrictList: ", err)
        return nil, err
    }

    return DistrictListToLeafRefOptions(response, "district-id", args...)

}


func (i *ServerImpl) GetTrafficClassificationDistrictDistrictRefValuesLeafref (ctx echo.Context, cityId CityId, districtRef string) error {
    var response interface{}
    var err error

    gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
    defer cancel()

    // Response GET OK 200
    response, err = i.GnmiGetTrafficClassificationDistrictDistrictRefValuesLeafref(gnmiCtx, "/sca/v0.1.x/{city-id}/district", cityId, districtRef)
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

    return ctx.JSON(http.StatusOK, response)
}


func (i *ServerImpl) GnmiGetTrafficMonitoringDistrictDistrictRefValuesLeafref (ctx context.Context, openApiPath string, enterpriseId CityId , args ...string) (*LeafRefOptions, error) {
    gnmiGet, err := utils.NewGnmiGetRequest(openApiPath, string(enterpriseId), args...)
    if err != nil {
        return nil, err
    }
    log.Infof("gnmiGetRequest %s", gnmiGet.String())
    gnmiVal, err := utils.GetResponseUpdate(i.GnmiClient.Get(ctx, gnmiGet))
    if err != nil {
        log.Info("getresponse update error: ", err)
        return nil, err
    }
    if gnmiVal == nil {
        log.Info("gnmiVal is empty")
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

    response, err := mpd.ToDistrictList(args[:0]...)
    if err != nil {
        log.Info("error unmarshaling to DistrictList: ", err)
        return nil, err
    }

    return DistrictListToLeafRefOptions(response, "district-id", args...)

}


func (i *ServerImpl) GetTrafficMonitoringDistrictDistrictRefValuesLeafref (ctx echo.Context, cityId CityId, districtRef string) error {
    var response interface{}
    var err error

    gnmiCtx, cancel := utils.NewGnmiContext(ctx, i.GnmiTimeout)
    defer cancel()

    // Response GET OK 200
    response, err = i.GnmiGetTrafficMonitoringDistrictDistrictRefValuesLeafref(gnmiCtx, "/sca/v0.1.x/{city-id}/district", cityId, districtRef)
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

    return ctx.JSON(http.StatusOK, response)
}


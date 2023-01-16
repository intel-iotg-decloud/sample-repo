// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package sca_0_1_x

import (
	"context"
	"github.com/labstack/echo/v4"
	"github.com/onosproject/aether-roc-api/pkg/utils"
	"net/http"
	"reflect"
)

// Dummy function so that the imports above are referenced even when there are no LeafSelections
func Dummy() {
	reflect.TypeOf(echo.Version)
	reflect.TypeOf(http.StatusOK)
}

func (i *ServerImpl) LeafSelection(ctx context.Context, queryPath string, enterpriseId CityId, args ...string) (*LeafRefOptions, error) {

	selectionValues, err := utils.LeafSelection(ctx, i.ConfigAdminServiceClient, "sca", "0.1.x", queryPath, string(enterpriseId), args...)
	if err != nil {
		return nil, err
	}

	leafSelectionOptions := make(LeafRefOptions, 0)
	for _, r := range selectionValues {
		r := r // pinning
		leafSelectionOptions = append(leafSelectionOptions, LeafRefOption{
			Label: &r,
			Value: &r,
		})
	}

	return &leafSelectionOptions, nil
}



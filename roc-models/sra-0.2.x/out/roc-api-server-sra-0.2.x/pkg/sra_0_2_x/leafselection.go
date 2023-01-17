// SPDX-FileCopyrightText: 2022-present Intel Corporation
//
// SPDX-License-Identifier: LicenseRef-Intel

package sra_0_2_x

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

func (i *ServerImpl) LeafSelection(ctx context.Context, queryPath string, enterpriseId StoreId, args ...string) (*LeafRefOptions, error) {

	selectionValues, err := utils.LeafSelection(ctx, i.ConfigAdminServiceClient, "sra", "0.2.x", queryPath, string(enterpriseId), args...)
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



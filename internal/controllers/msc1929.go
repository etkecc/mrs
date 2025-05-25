package controllers

import (
	"net/http"
	"slices"
	"strings"

	"github.com/etkecc/go-msc1929"
	"github.com/etkecc/mrs/internal/model"
	"github.com/etkecc/mrs/internal/utils"
	"github.com/labstack/echo/v4"
)

var (
	errMSC1929Empty = &model.MatrixError{
		Code:    "CC.ETKE.MSC1929_EMPTY",
		Message: "The support file is missing or empty (contains neither contacts nor support url).",
	}
	errMSC1929MissingRole = &model.MatrixError{
		Code:    "CC.ETKE.MSC1929_MISSING_ROLE",
		Message: "The support file contains a contact without a role. Supported roles are: " + strings.Join(msc1929.SupportedRoles, ", "),
	}
	errMSC1929UnsupportedRole = &model.MatrixError{
		Code:    "CC.ETKE.MSC1929_UNSUPPORTED_ROLE",
		Message: "The support file contains an unsupported role in one of the contacts. Supported roles are: " + strings.Join(msc1929.SupportedRoles, ", "),
	}
	errMSC1929NoEmails = &model.MatrixError{
		Code:    "CC.ETKE.MSC1929_NO_EMAILS",
		Message: "The support file doesn't contain any email addresses.",
	}
	errMSC1929NoMatrixIDs = &model.MatrixError{
		Code:    "CC.ETKE.MSC1929_NO_MXIDS",
		Message: "The support file doesn't contain any Matrix IDs.",
	}
	errMSC1929Outdated = &model.MatrixError{
		Code:    "CC.ETKE.MSC1929_OUTDATED",
		Message: "The support file uses the deprecated 'admins' field. Please use 'contacts' instead.",
	}
)

func validateMSC1929(resp *msc1929.Response) []*model.MatrixError {
	errs := []*model.MatrixError{}
	if resp.IsEmpty() {
		errs = append(errs, errMSC1929Empty)
	}
	if len(resp.Admins) > 0 {
		errs = append(errs, errMSC1929Outdated)
	}
	if len(resp.AllEmails()) == 0 {
		errs = append(errs, errMSC1929NoEmails)
	}
	if len(resp.AllMatrixIDs()) == 0 {
		errs = append(errs, errMSC1929NoMatrixIDs)
	}

	withoutRole := false
	for _, contact := range resp.Contacts {
		if contact.Role == "" && !withoutRole {
			withoutRole = true
			errs = append(errs, errMSC1929MissingRole)
			continue
		}
		if !slices.Contains(msc1929.SupportedRoles, contact.Role) {
			errs = append(errs, errMSC1929UnsupportedRole)
			break
		}
	}

	return errs
}

// checkMSC1929 is a simple tool to check if a server has a valid MSC1929 support file.
func checkMSC1929() echo.HandlerFunc {
	return func(c echo.Context) error {
		name := c.Param("name")
		resp, err := msc1929.GetWithContext(c.Request().Context(), name)
		if err != nil {
			return c.JSONBlob(http.StatusBadRequest, utils.MustJSON([]*model.MatrixError{{
				Code:    "CC.ETKE.MSC1929_ERROR",
				Message: err.Error(),
			}}))
		}

		errs := validateMSC1929(resp)
		if len(errs) == 0 {
			return c.NoContent(http.StatusNoContent)
		}
		return c.JSONBlob(http.StatusBadRequest, utils.MustJSON(errs))
	}
}

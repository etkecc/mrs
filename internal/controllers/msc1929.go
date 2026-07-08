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
	errMSC1929NoContacts = &model.MatrixError{
		Code:    "CC.ETKE.MSC1929_NO_CONTACTS",
		Message: "The support file contains neither email addresses, nor matrix ids.",
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
		return errs
	}
	if len(resp.Admins) > 0 {
		errs = append(errs, errMSC1929Outdated)
	}
	if len(resp.AllEmails()) == 0 && len(resp.AllMatrixIDs()) == 0 {
		errs = append(errs, errMSC1929NoContacts)
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

// @Summary		Check a server's MSC1929 support file
// @Description	Fetches and validates a server's MSC1929 support file (the contacts in /.well-known/matrix/support). Returns 204 when it is valid, or 400 with a list of the specific problems: empty file, a contact missing a role, an unsupported role, no contacts at all, or the deprecated 'admins' field. A handy pre-flight before relying on a server's support contacts.
// @Tags			discovery
// @Produce		json
// @Param			name	path	string	true	"Server name to check"
// @Success		204		"Support file is valid"
// @Failure		400		{array}	model.MatrixError	"Support file is missing, unreachable, or invalid"
// @Router			/discover/msc1929/{name} [post]
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

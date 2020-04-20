package contentful

import (
	"errors"
	"os"
)

// noinspection GoUnusedGlobalVariable
var (
	baseURL               = "https://api.contentful.com"
	contentfulContentType = "application/vnd.contentful.management.v1+json"
	spaceID               = os.Getenv("SPACE_ID")
	// User friendly errors we return
	errorUnauthorized         = errors.New("401 Unauthorized. Is the CMA token valid")
	errorSpaceNotFound        = errors.New("space not found")
	errorOrganizationNotFound = errors.New("organization not found")
	errorLocaleNotFound       = errors.New("locale not found")
	errorWebhookNotFound      = errors.New("the webhook could not be found")
)
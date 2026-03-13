package bird

import "errors"

// errMissingCredentials is returned when auth_token or ct0 is empty.
var errMissingCredentials = errors.New("bird: auth_token and ct0 are required")

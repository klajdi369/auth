package api

import (
	"github.com/supabase/gotrue/internal/models"
	"github.com/supabase/gotrue/internal/storage"
	"regexp"
	"time"
)

const PKCE = "pkce"

const MinCodeChallengeLength = 43
const MaxCodeChallengeLength = 128

var codeChallengePattern = regexp.MustCompile("^[a-zA-Z._~0-9-]+$")

const InvalidFlowTypeErrorMessage = "Invalid flow type. Flow Type must be either implicit or pkce"
const InvalidPKCEParamsErrorMessage = "PKCE flow requires code_challenge_method and code_challenge"

func isValidCodeChallenge(codeChallenge string) (bool, error) {
	// See RFC 7636 Section 4.2: https://www.rfc-editor.org/rfc/rfc7636#section-4.2

	switch codeChallengeLength := len(codeChallenge); {
	case codeChallengeLength < MinCodeChallengeLength, codeChallengeLength > MaxCodeChallengeLength:
		return false, badRequestError("code challenge has to be between %v and %v characters", MinCodeChallengeLength, MaxCodeChallengeLength)
	case !codeChallengePattern.MatchString(codeChallenge):
		return false, badRequestError("code challenge can only contain alphanumeric characters, hyphens, periods, underscores and tildes")
	default:
		return true, nil
	}
}

func addFlowPrefixToToken(token string, flowType models.FlowType) string {
	if isPKCEFlow(flowType) {
		return models.PKCEFlow.String() + "_" + token
	} else if isImplicitFlow(flowType) {
		return token
	}
	return token
}

func issueAuthCode(tx *storage.Connection, user *models.User, expiryDuration time.Duration) (string, error) {
	flowState, err := models.FindFlowStateByUserID(tx, user.ID.String())
	if models.IsNotFoundError(err) {
		return "", badRequestError("No valid flow state found for user.")
	} else if err != nil {
		return "", err
	}

	if flowState.IsExpired(expiryDuration) {
		return "", badRequestError("Flow state is expired")
	}
	return flowState.AuthCode, nil
}

func isPKCEFlow(flowType models.FlowType) bool {
	return flowType == models.PKCEFlow
}

func isImplicitFlow(flowType models.FlowType) bool {
	return flowType == models.ImplicitFlow
}

func validatePKCEParams(codeChallengeMethod, codeChallenge string) error {
	switch true {
	// Explicitly spell out each case
	case codeChallenge == "" && codeChallengeMethod != "":
		return badRequestError(InvalidPKCEParamsErrorMessage)
	case codeChallenge != "" && codeChallengeMethod == "":
		return badRequestError(InvalidPKCEParamsErrorMessage)
	case codeChallenge != "" && codeChallengeMethod != "":
		break
	case codeChallenge == "" && codeChallengeMethod == "":
		break
	default:
		return badRequestError(InvalidPKCEParamsErrorMessage)
	}
	return nil
}

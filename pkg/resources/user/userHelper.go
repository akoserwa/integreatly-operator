package user

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	keycloak "github.com/keycloak/keycloak-operator/pkg/apis/keycloak/v1alpha1"
	usersv1 "github.com/openshift/api/user/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Helper for User associated functions

const (
	updateProfileAction         = "UPDATE_PROFILE"
	invalidCharacterReplacement = "-"
	GeneratedNamePrefix         = "generated-"
)

var (
	exclusionGroups = []string{
		"layered-cs-sre-admins",
		"osd-sre-admins",
	}
)

func GetUserEmailFromIdentity(ctx context.Context, serverClient k8sclient.Client, user usersv1.User) (string, error) {
	email := ""

	// User can have multiple identities
	for _, identityName := range user.Identities {
		identity := &usersv1.Identity{}
		err := serverClient.Get(ctx, k8sclient.ObjectKey{Name: identityName}, identity)

		if err != nil {
			return "", fmt.Errorf("failed to get identity provider: %w", err)
		}

		// Get first identity with email and break loop
		if identity.Extra["email"] != "" {
			email = identity.Extra["email"]
			break
		}
	}

	return email, nil
}

func AppendUpdateProfileActionForUserWithoutEmail(keycloakUser *keycloak.KeycloakAPIUser) {
	if keycloakUser.Email == "" {
		keycloakUser.RequiredActions = []string{updateProfileAction}
	}
}

func GetValidGeneratedUserName(keycloakUser keycloak.KeycloakAPIUser) string {
	// Regex for only alphanumeric values
	reg, _ := regexp.Compile("[^a-zA-Z0-9]+")

	// Replace all non-alphanumeric values with the replacement character
	processedString := reg.ReplaceAllString(strings.ToLower(keycloakUser.UserName), invalidCharacterReplacement)

	// Remove occurrence of replacement character at end of string
	processedString = strings.TrimSuffix(processedString, invalidCharacterReplacement)

	for _, federatedIdentity := range keycloakUser.FederatedIdentities {
		userId := federatedIdentity.UserID
		// Append user id to name to ensure uniqueness
		if userId != "" {
			processedString = fmt.Sprintf("%s-%s", processedString, federatedIdentity.UserID)
			break
		}
	}

	return fmt.Sprintf("%v%v", GeneratedNamePrefix, processedString)
}

func UserInExclusionGroup(user usersv1.User, groups *usersv1.GroupList) bool {

	// Below is a slightly complex way to determine if the user exists in an exlcusion group
	// Ideally we would use the user.Groups field but this does not seem to get populated.
	for _, group := range groups.Items {
		for _, xGroup := range exclusionGroups {
			if group.Name == xGroup {
				for _, groupUser := range group.Users {
					if groupUser == user.Name {
						return true
					}
				}
			}
		}
	}
	return false
}

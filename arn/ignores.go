package arn

var ignores = []string{
	// Use ListPolicies
	"iam.Client.ListPoliciesGrantingServiceAccess",
	// Use ListGroups
	"iam.Client.ListGroupsForUser",
	// Use ListInstanceProfiles
	"iam.Client.ListInstanceProfilesForRole",
	// Use ListEnabledProductsForImport. ref: https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/securityhub#Client.DescribeProducts
	"securityhub.Client.DescribeProducts",
	// Does not have ARNs.
	"iam.Client.ListPolicyVersions",
	"iam.Client.ListUserPolicies",
	"iam.Client.ListGroupPolicies",
	"iam.Client.ListRolePolicies",
	"iam.Client.ListEntitiesForPolicy",
	"iam.Client.ListServiceSpecificCredentials",
	"iam.Client.ListSigningCertificates",
	"iam.Client.ListSSHPublicKeys",
	"iam.Client.ListMFADevices",
	"iam.Client.ListAccessKeys",
	"securityhub.Client.ListMembers",
	"securityhub.Client.ListInvitations",
	"securityhub.Client.DescribeOrganizationConfiguration",
	"securityhub.Client.ListOrganizationAdminAccounts",
}

var extras = []string{
	"securityhub.Client.DescribeHub",
	"securityhub.Client.DescribeStandards",
	"securityhub.Client.DescribeStandardsControls",
}

var methodPrefixIgnores = []string{
	// List even if not attached
	"ListAttached",
	// Tag does not have ARNs.
	"ListTagsFor",
}

var methodSuffixIgnores = []string{
	// Tag does not have ARNs.
	"Tags",
}

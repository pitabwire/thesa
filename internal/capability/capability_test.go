package capability

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/pitabwire/frame/config"
	"github.com/pitabwire/frame/frametests"
	"github.com/pitabwire/frame/frametests/definition"
	"github.com/pitabwire/frame/frametests/deps/testoryketo"
	"github.com/pitabwire/frame/security"
	"github.com/pitabwire/frame/security/authorizer"
	"github.com/stretchr/testify/suite"

	thesaconfig "github.com/pitabwire/thesa/internal/config"
	thesadefinition "github.com/pitabwire/thesa/internal/definition"
	"github.com/pitabwire/thesa/model"
)

// servicePartitionOPL defines the OPL namespace for the partition service.
// Relations use underscores (OPL-compatible). SubjectSet<service_partition, "admin">
// allows the admin role to grant all permissions via Keto's graph traversal.
const servicePartitionOPL = `import { Namespace, Context } from "@ory/keto-namespace-types"

class User implements Namespace {}
class profile_user implements Namespace {}

class service_partition implements Namespace {
  related: {
    tenants_view: (profile_user | SubjectSet<service_partition, "admin">)[]
    tenants_create: (profile_user | SubjectSet<service_partition, "admin">)[]
    tenants_edit: (profile_user | SubjectSet<service_partition, "admin">)[]
    partitions_view: (profile_user | SubjectSet<service_partition, "admin">)[]
    partitions_create: (profile_user | SubjectSet<service_partition, "admin">)[]
    roles_view: (profile_user | SubjectSet<service_partition, "admin">)[]
    roles_create: (profile_user | SubjectSet<service_partition, "admin">)[]
    roles_delete: (profile_user | SubjectSet<service_partition, "admin">)[]
    access_view: (profile_user | SubjectSet<service_partition, "admin">)[]
    access_create: (profile_user | SubjectSet<service_partition, "admin">)[]
    access_delete: (profile_user | SubjectSet<service_partition, "admin">)[]
    access_edit: (profile_user | SubjectSet<service_partition, "admin">)[]
    admin: (profile_user)[]
  }
}
`

// allPartitionPermissions lists all Keto permission names (underscore format)
// for the partition service.
var allPartitionPermissions = []string{
	"tenants_view", "tenants_create", "tenants_edit",
	"partitions_view", "partitions_create",
	"roles_view", "roles_create", "roles_delete",
	"access_view", "access_create", "access_delete", "access_edit",
}

// --- Test Suite ---

type CapabilityTestSuite struct {
	frametests.FrameBaseTestSuite
	adapter security.Authorizer
}

func initKetoResources(_ context.Context) []definition.TestResource {
	return []definition.TestResource{
		testoryketo.NewWithNamespaces(
			testoryketo.KetoConfiguration,
			[]testoryketo.NamespaceFile{
				{
					ContainerPath: "/home/ory/namespaces/namespaces.ts",
					Content:       servicePartitionOPL,
				},
			},
			definition.WithEnableLogging(false),
		),
	}
}

func (s *CapabilityTestSuite) SetupSuite() {
	s.InitResourceFunc = initKetoResources
	s.FrameBaseTestSuite.SetupSuite()

	ctx := s.T().Context()

	var ketoDep definition.DependancyConn
	for _, res := range s.Resources() {
		if res.Name() == testoryketo.OryKetoImage {
			ketoDep = res
			break
		}
	}
	s.Require().NotNil(ketoDep, "keto dependency should be available")

	writeURL, err := url.Parse(string(ketoDep.GetDS(ctx)))
	s.Require().NoError(err)
	writeURI := writeURL.Host

	readPort, err := ketoDep.PortMapping(ctx, "4466/tcp")
	s.Require().NoError(err)
	readURI := fmt.Sprintf("%s:%s", writeURL.Hostname(), readPort)

	cfg := &config.ConfigurationDefault{
		AuthorizationServiceReadURI:  readURI,
		AuthorizationServiceWriteURI: writeURI,
	}
	s.adapter = authorizer.NewKetoAdapter(cfg, nil)
}

func TestCapabilitySuite(t *testing.T) {
	suite.Run(t, &CapabilityTestSuite{})
}

// --- Helper methods ---

// writeTuple writes a direct permission tuple: subject has relation on object.
func (s *CapabilityTestSuite) writeTuple(namespace, objectID, relation, subjectID string) {
	s.T().Helper()
	err := s.adapter.WriteTuple(s.T().Context(), security.RelationTuple{
		Object:   security.ObjectRef{Namespace: namespace, ID: objectID},
		Relation: relation,
		Subject:  security.SubjectRef{Namespace: security.NamespaceProfile, ID: subjectID},
	})
	s.Require().NoError(err)
}

// writeSubjectSetTuple writes a SubjectSet tuple: anyone with sourceRelation
// on the object also gets targetRelation on the same object.
func (s *CapabilityTestSuite) writeSubjectSetTuple(namespace, objectID, targetRelation, sourceRelation string) {
	s.T().Helper()
	err := s.adapter.WriteTuple(s.T().Context(), security.RelationTuple{
		Object:   security.ObjectRef{Namespace: namespace, ID: objectID},
		Relation: targetRelation,
		Subject: security.SubjectRef{
			Namespace: namespace,
			ID:        objectID,
			Relation:  sourceRelation,
		},
	})
	s.Require().NoError(err)
}

// provisionAdminRole writes the admin role and all admin→permission SubjectSet
// mappings for a given tenancy path. This is how role-based access is provisioned
// in Keto: the role grants each permission via graph traversal.
func (s *CapabilityTestSuite) provisionAdminRole(tenancyPath, subjectID string) {
	s.T().Helper()

	// Grant admin role to user.
	s.writeTuple("service_partition", tenancyPath, "admin", subjectID)

	// Map admin role → every permission via SubjectSet.
	for _, perm := range allPartitionPermissions {
		s.writeSubjectSetTuple("service_partition", tenancyPath, perm, "admin")
	}
}

func (s *CapabilityTestSuite) newEvaluator(checks []CapabilityCheck) *KetoPolicyEvaluator {
	return NewKetoPolicyEvaluator(s.adapter, checks)
}

func (s *CapabilityTestSuite) allPartitionChecks() []CapabilityCheck {
	return []CapabilityCheck{
		{Capability: "tenants:view", Namespace: "service_partition"},
		{Capability: "tenants:create", Namespace: "service_partition"},
		{Capability: "tenants:edit", Namespace: "service_partition"},
		{Capability: "partitions:view", Namespace: "service_partition"},
		{Capability: "partitions:create", Namespace: "service_partition"},
		{Capability: "roles:view", Namespace: "service_partition"},
		{Capability: "roles:create", Namespace: "service_partition"},
		{Capability: "roles:delete", Namespace: "service_partition"},
		{Capability: "access:view", Namespace: "service_partition"},
		{Capability: "access:create", Namespace: "service_partition"},
		{Capability: "access:delete", Namespace: "service_partition"},
		{Capability: "access:edit", Namespace: "service_partition"},
	}
}

// --- Direct permission tests ---

func (s *CapabilityTestSuite) TestDirectPermission() {
	ctx := s.T().Context()
	tenancyPath := "tenant-direct/part-1"

	s.writeTuple("service_partition", tenancyPath, "tenants_view", "user-viewer")
	s.writeTuple("service_partition", tenancyPath, "partitions_view", "user-viewer")

	eval := s.newEvaluator(s.allPartitionChecks())
	rctx := &model.RequestContext{
		SubjectID: "user-viewer", TenantID: "tenant-direct", PartitionID: "part-1",
	}

	caps, err := eval.ResolveCapabilities(ctx, rctx)
	s.Require().NoError(err)

	s.True(caps.Has("tenants:view"), "should have tenants:view")
	s.True(caps.Has("partitions:view"), "should have partitions:view")
	s.False(caps.Has("tenants:create"), "should not have tenants:create")
	s.False(caps.Has("tenants:edit"), "should not have tenants:edit")
	s.False(caps.Has("roles:view"), "should not have roles:view")
	s.False(caps.Has("access:view"), "should not have access:view")
}

// TestAdminRoleGrantsAllPermissions verifies that the admin role grants all
// capabilities through Keto's SubjectSet graph traversal.
func (s *CapabilityTestSuite) TestAdminRoleGrantsAllPermissions() {
	ctx := s.T().Context()

	s.provisionAdminRole("tenant-admin/part-1", "user-admin")

	eval := s.newEvaluator(s.allPartitionChecks())
	rctx := &model.RequestContext{
		SubjectID: "user-admin", TenantID: "tenant-admin", PartitionID: "part-1",
	}

	caps, err := eval.ResolveCapabilities(ctx, rctx)
	s.Require().NoError(err)

	allCaps := []string{
		"tenants:view", "tenants:create", "tenants:edit",
		"partitions:view", "partitions:create",
		"roles:view", "roles:create", "roles:delete",
		"access:view", "access:create", "access:delete", "access:edit",
	}

	for _, cap := range allCaps {
		s.True(caps.Has(cap), "admin should have %s", cap)
	}
	s.Equal(len(allCaps), len(caps), "admin should have exactly %d capabilities", len(allCaps))
}

// TestTenantIsolation verifies that permissions for one tenant do not leak
// to another tenant.
func (s *CapabilityTestSuite) TestTenantIsolation() {
	ctx := s.T().Context()

	s.provisionAdminRole("tenant-a/part-1", "user-isolated")

	eval := s.newEvaluator(s.allPartitionChecks())

	capsA, err := eval.ResolveCapabilities(ctx, &model.RequestContext{
		SubjectID: "user-isolated", TenantID: "tenant-a", PartitionID: "part-1",
	})
	s.Require().NoError(err)
	s.True(capsA.Has("tenants:view"), "should have tenants:view on tenant-a")
	s.Equal(12, len(capsA), "should have all 12 capabilities on tenant-a")

	capsB, err := eval.ResolveCapabilities(ctx, &model.RequestContext{
		SubjectID: "user-isolated", TenantID: "tenant-b", PartitionID: "part-1",
	})
	s.Require().NoError(err)
	s.Equal(0, len(capsB), "should have no capabilities on tenant-b")
}

// TestPartitionIsolation verifies that permissions are scoped to a specific
// partition within a tenant.
func (s *CapabilityTestSuite) TestPartitionIsolation() {
	ctx := s.T().Context()

	s.writeTuple("service_partition", "tenant-pi/part-main", "tenants_view", "user-part")

	eval := s.newEvaluator(s.allPartitionChecks())

	caps1, err := eval.ResolveCapabilities(ctx, &model.RequestContext{
		SubjectID: "user-part", TenantID: "tenant-pi", PartitionID: "part-main",
	})
	s.Require().NoError(err)
	s.True(caps1.Has("tenants:view"))

	caps2, err := eval.ResolveCapabilities(ctx, &model.RequestContext{
		SubjectID: "user-part", TenantID: "tenant-pi", PartitionID: "part-other",
	})
	s.Require().NoError(err)
	s.False(caps2.Has("tenants:view"))
}

// TestMixedPermissions grants some direct permissions and verifies
// the exact set that resolves.
func (s *CapabilityTestSuite) TestMixedPermissions() {
	ctx := s.T().Context()
	tenancyPath := "tenant-mixed/part-1"

	s.writeTuple("service_partition", tenancyPath, "tenants_view", "user-mgr")
	s.writeTuple("service_partition", tenancyPath, "tenants_create", "user-mgr")
	s.writeTuple("service_partition", tenancyPath, "tenants_edit", "user-mgr")
	s.writeTuple("service_partition", tenancyPath, "partitions_view", "user-mgr")
	s.writeTuple("service_partition", tenancyPath, "partitions_create", "user-mgr")

	eval := s.newEvaluator(s.allPartitionChecks())
	rctx := &model.RequestContext{
		SubjectID: "user-mgr", TenantID: "tenant-mixed", PartitionID: "part-1",
	}

	caps, err := eval.ResolveCapabilities(ctx, rctx)
	s.Require().NoError(err)

	s.Equal(5, len(caps), "should have exactly 5 capabilities")
	s.True(caps.Has("tenants:view"))
	s.True(caps.Has("tenants:create"))
	s.True(caps.Has("tenants:edit"))
	s.True(caps.Has("partitions:view"))
	s.True(caps.Has("partitions:create"))
	s.False(caps.Has("roles:view"))
	s.False(caps.Has("access:view"))
}

// TestNoPermissions verifies that a user with no tuples gets empty capabilities.
func (s *CapabilityTestSuite) TestNoPermissions() {
	ctx := s.T().Context()

	eval := s.newEvaluator(s.allPartitionChecks())
	caps, err := eval.ResolveCapabilities(ctx, &model.RequestContext{
		SubjectID: "user-nobody", TenantID: "tenant-none", PartitionID: "part-1",
	})
	s.Require().NoError(err)
	s.Equal(0, len(caps))
}

// TestEmptyChecks verifies that an evaluator with no checks returns empty capabilities.
func (s *CapabilityTestSuite) TestEmptyChecks() {
	ctx := s.T().Context()

	eval := s.newEvaluator(nil)
	caps, err := eval.ResolveCapabilities(ctx, &model.RequestContext{
		SubjectID: "user-1", TenantID: "tenant-1", PartitionID: "part-1",
	})
	s.Require().NoError(err)
	s.Equal(0, len(caps))
}

// --- Resolver caching tests ---

func (s *CapabilityTestSuite) TestResolverCaching() {
	ctx := s.T().Context()

	s.writeTuple("service_partition", "tenant-cache/part-1", "tenants_view", "user-cache")

	eval := s.newEvaluator(s.allPartitionChecks())
	resolver := NewResolver(eval, 5*time.Minute)

	rctx := &model.RequestContext{
		SubjectID: "user-cache", TenantID: "tenant-cache", PartitionID: "part-1",
	}

	caps1, err := resolver.Resolve(ctx, rctx)
	s.Require().NoError(err)
	s.True(caps1.Has("tenants:view"))

	caps2, err := resolver.Resolve(ctx, rctx)
	s.Require().NoError(err)
	s.True(caps2.Has("tenants:view"))
}

func (s *CapabilityTestSuite) TestResolverInvalidate() {
	ctx := s.T().Context()

	s.writeTuple("service_partition", "tenant-inv/part-1", "tenants_view", "user-inv")

	eval := s.newEvaluator(s.allPartitionChecks())
	resolver := NewResolver(eval, 5*time.Minute)

	rctx := &model.RequestContext{
		SubjectID: "user-inv", TenantID: "tenant-inv", PartitionID: "part-1",
	}

	caps1, err := resolver.Resolve(ctx, rctx)
	s.Require().NoError(err)
	s.True(caps1.Has("tenants:view"))

	resolver.Invalidate("user-inv", "tenant-inv")

	caps2, err := resolver.Resolve(ctx, rctx)
	s.Require().NoError(err)
	s.True(caps2.Has("tenants:view"))
}

// --- CollectCapabilityChecks with real definitions ---

func (s *CapabilityTestSuite) TestCollectCapabilityChecks_AccessControlDomain() {
	loader := thesadefinition.NewLoader()
	defs, err := loader.LoadAll([]string{"../../definitions"})
	s.Require().NoError(err)

	var accessControl *model.DomainDefinition
	for _, d := range defs {
		if d.Domain == "access-control" {
			accessControl = &d
			break
		}
	}
	s.Require().NotNil(accessControl, "access-control domain not found")

	services := map[string]thesaconfig.ServiceConfig{
		"partition-svc": {AuthorizationNamespace: "service_partition"},
	}

	checks := CollectCapabilityChecks([]model.DomainDefinition{*accessControl}, services)

	found := make(map[string]string)
	for _, chk := range checks {
		found[chk.Capability] = chk.Namespace
	}

	expectedCaps := []string{
		"tenants:view", "tenants:create", "tenants:edit",
		"partitions:view", "partitions:create",
		"roles:view", "roles:create", "roles:delete",
		"access:view", "access:create", "access:delete", "access:edit",
	}

	for _, cap := range expectedCaps {
		ns, ok := found[cap]
		s.True(ok, "missing capability %q", cap)
		if ok {
			s.Equal("service_partition", ns, "capability %q namespace", cap)
		}
	}

	if len(checks) != len(expectedCaps) {
		var got []string
		for _, chk := range checks {
			got = append(got, chk.Capability)
		}
		sort.Strings(got)
		sort.Strings(expectedCaps)
		s.Equalf(expectedCaps, got, "capability list mismatch")
	}
}

// --- Permission mapping tests ---

func (s *CapabilityTestSuite) TestCapabilityToPermission() {
	s.Equal("tenants_view", CapabilityToPermission("tenants:view"))
	s.Equal("access_create", CapabilityToPermission("access:create"))
	s.Equal("simple", CapabilityToPermission("simple"))
	s.Equal("a_b_c", CapabilityToPermission("a:b:c"))
}

func (s *CapabilityTestSuite) TestPermissionToCapability() {
	s.Equal("tenants:view", PermissionToCapability("tenants_view"))
	s.Equal("access:create", PermissionToCapability("access_create"))
	s.Equal("simple", PermissionToCapability("simple"))
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package sesv2_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/YakDriver/regexache"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hashicorp/terraform-provider-aws/internal/acctest"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"

	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"

	tfsesv2 "github.com/hashicorp/terraform-provider-aws/internal/service/sesv2"
)

// TIP: File Structure. The basic outline for all test files should be as
// follows. Improve this resource's maintainability by following this
// outline.
//
// 1. Package declaration (add "_test" since this is a test file)
// 2. Imports
// 3. Unit tests
// 4. Basic test
// 5. Disappears test
// 6. All the other tests
// 7. Helper functions (exists, destroy, check, etc.)
// 8. Functions that return Terraform configurations

// TIP: ==== UNIT TESTS ====
// This is an example of a unit test. Its name is not prefixed with
// "TestAcc" like an acceptance test.
//
// Unlike acceptance tests, unit tests do not access AWS and are focused on a
// function (or method). Because of this, they are quick and cheap to run.
//
// In designing a resource's implementation, isolate complex bits from AWS bits
// so that they can be tested through a unit test. We encourage more unit tests
// in the provider.
//
// Cut and dry functions using well-used patterns, like typical flatteners and
// expanders, don't need unit testing. However, if they are complex or
// intricate, they should be unit tested.
//func TestTenantExampleUnitTest(t *testing.T) {
//	t.Parallel()
//
//	testCases := []struct {
//		TestName string
//		Input    string
//		Expected string
//		Error    bool
//	}{
//		{
//			TestName: "empty",
//			Input:    "",
//			Expected: "",
//			Error:    true,
//		},
//		{
//			TestName: "Basic Tenant",
//			Input:    "some input",
//			Expected: "some output",
//			Error:    false,
//		},
//		{
//			TestName: "another descriptive name",
//			Input:    "more input",
//			Expected: "more output",
//			Error:    false,
//		},
//	}
//
//	for _, testCase := range testCases {
//		t.Run(testCase.TestName, func(t *testing.T) {
//			t.Parallel()
//			got, err := tfsesv2.FunctionFromResource(testCase.Input)
//
//			if err != nil && !testCase.Error {
//				t.Errorf("got error (%s), expected no error", err)
//			}
//
//			if err == nil && testCase.Error {
//				t.Errorf("got (%s) and no error, expected error", got)
//			}
//
//			if got != testCase.Expected {
//				t.Errorf("got %s, expected %s", got, testCase.Expected)
//			}
//		})
//	}
//}

// TIP: ==== ACCEPTANCE TESTS ====
// This is an example of a basic acceptance test. This should test as much of
// standard functionality of the resource as possible, and test importing, if
// applicable. We prefix its name with "TestAcc", the service, and the
// resource name.
//
// Acceptance test access AWS and cost money to run.
func TestAccSESV2Tenant_basic(t *testing.T) {
	ctx := acctest.Context(t)
	if testing.Short() {
		t.Skip("skipping long-running test in short mode")
	}

	rName := acctest.RandomWithPrefix(t, "tf-acc-test")
	resourceName := "aws_sesv2_tenant.test"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			acctest.PreCheck(ctx, t)
			testAccPreCheck(ctx, t)
		},
		ErrorCheck:               acctest.ErrorCheck(t, tfsesv2.ResNameTenant),
		ProtoV5ProviderFactories: acctest.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckTenantDestroy(ctx),
		Steps: []resource.TestStep{
			{
				Config: testAccTenantConfig_basic(rName),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckTenantExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "tags.testkey", "testvalue"),
					resource.TestCheckResourceAttr(resourceName, "tags_all.testkey", "testvalue"),
					acctest.MatchResourceAttrRegionalARN(ctx, resourceName, names.AttrARN, "ses", regexache.MustCompile(`tenant/.+$`)),
				),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCheckTenantDestroy(ctx context.Context) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		conn := acctest.Provider.Meta().(*conns.AWSClient).SESV2Client(ctx)

		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_sesv2_tenant" {
				continue
			}

			tenantName := rs.Primary.Attributes["tenant_name"]
			_, err := tfsesv2.FindTenantByName(ctx, conn, tenantName)

			if tfresource.NotFound(err) {
				return nil
			}
			if err != nil {
				return create.Error(names.SESV2, create.ErrActionCheckingDestroyed, tfsesv2.ResNameTenant, tenantName, err)
			}

			return create.Error(names.SESV2, create.ErrActionCheckingDestroyed, tfsesv2.ResNameTenant, tenantName, errors.New("not destroyed"))
		}

		return nil
	}
}

func testAccCheckTenantExists(ctx context.Context, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return create.Error(names.SESV2, create.ErrActionCheckingExistence, tfsesv2.ResNameTenant, name, errors.New("not found"))
		}

		tenantName, ok := rs.Primary.Attributes["tenant_name"]
		if !ok || tenantName == "" {
			return create.Error(names.SESV2, create.ErrActionCheckingExistence, tfsesv2.ResNameTenant, name, errors.New("tenant_name attribute not found or empty"))
		}

		conn := acctest.Provider.Meta().(*conns.AWSClient).SESV2Client(ctx)

		_, err := tfsesv2.FindTenantByName(ctx, conn, tenantName)
		if err != nil {
			return create.Error(names.SESV2, create.ErrActionCheckingExistence, tfsesv2.ResNameTenant, tenantName, err)
		}

		return nil
	}
}

func testAccPreCheck(ctx context.Context, t *testing.T) {
	conn := acctest.Provider.Meta().(*conns.AWSClient).SESV2Client(ctx)

	input := &sesv2.ListTenantsInput{}

	_, err := conn.ListTenants(ctx, input)

	if acctest.PreCheckSkipError(err) {
		t.Skipf("skipping acceptance testing: %s", err)
	}
	if err != nil {
		t.Fatalf("unexpected PreCheck error: %s", err)
	}
}

func testAccCheckTenantNotRecreated(before, after *sesv2.GetTenantOutput) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		beforeID := aws.ToString(before.Tenant.TenantId)
		afterID := aws.ToString(before.Tenant.TenantId)
		if beforeID != afterID {
			return create.Error(names.SESV2, create.ErrActionCheckingNotRecreated, tfsesv2.ResNameTenant, beforeID, errors.New("recreated"))
		}

		return nil
	}
}

func testAccTenantConfig_basic(rName string) string {
	return fmt.Sprintf(`
resource "aws_sesv2_tenant" "test" {
  tenant_name             = %[1]q
	tags = {
		"testkey" = "testvalue"
	}
}
`, rName)
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package sesv2

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/YakDriver/smarterr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	awstypes "github.com/aws/aws-sdk-go-v2/service/sesv2/types"

	//"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	//"github.com/hashicorp/terraform-plugin-framework-timetypes/timetypes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"

	//"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"

	//"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/errs"
	"github.com/hashicorp/terraform-provider-aws/internal/errs/fwdiag"
	"github.com/hashicorp/terraform-provider-aws/internal/framework"
	"github.com/hashicorp/terraform-provider-aws/internal/framework/flex"

	//fwtypes "github.com/hashicorp/terraform-provider-aws/internal/framework/types"
	"github.com/hashicorp/terraform-provider-aws/internal/smerr"
	"github.com/hashicorp/terraform-provider-aws/internal/sweep"
	sweepfw "github.com/hashicorp/terraform-provider-aws/internal/sweep/framework"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// TIP: ==== FILE STRUCTURE ====
// All resources should follow this basic outline. Improve this resource's
// maintainability by sticking to it.
//
// 1. Package declaration
// 2. Imports
// 3. Main resource struct with schema method
// 4. Create, read, update, delete methods (in that order)
// 5. Other functions (flatteners, expanders, waiters, finders, etc.)

// Function annotations are used for resource registration to the Provider. DO NOT EDIT.
// @FrameworkResource("aws_sesv2_tenant", name="Tenant")
// @Tags(identifierAttribute="arn")
func newResourceTenant(_ context.Context) (resource.ResourceWithConfigure, error) {
	r := &resourceTenant{}
	return r, nil
}

const (
	ResNameTenant = "Tenant"
)

type resourceTenant struct {
	framework.ResourceWithModel[resourceTenantModel]
}

func (r *resourceTenant) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			names.AttrARN: framework.ARNAttributeComputedOnly(),
			"created_timestamp": schema.StringAttribute{
				Computed:    true,
				Description: "The timestamp of when the Tenant was created",
			},
			names.AttrID: framework.IDAttribute(),
			"tenant_name": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Description: "Name of the Tenant",
			},
			"sending_status": schema.StringAttribute{
				Computed:    true,
				Description: "The sending status of the tenant. ENABLED, DISABLED, or REINSTATED",
			},
			names.AttrTags:    tftags.TagsAttribute(),
			names.AttrTagsAll: tftags.TagsAttributeComputedOnly(),
		},
	}
}

func (r *resourceTenant) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TIP: -- 1. Get a client connection to the relevant service
	conn := r.Meta().SESV2Client(ctx)

	// TIP: -- 2. Fetch the plan
	var plan resourceTenantModel

	smerr.AddEnrich(ctx, &resp.Diagnostics, req.Plan.Get(ctx, &plan))
	if resp.Diagnostics.HasError() {
		return
	}
	// TIP: -- 3. Populate a Create input structure
	var input sesv2.CreateTenantInput

	// TIP: Using a field name prefix allows mapping fields such as `ID` to `TenantId`
	smerr.AddEnrich(ctx, &resp.Diagnostics, flex.Expand(ctx, plan, &input, flex.WithFieldNamePrefix("Tenant")))
	if resp.Diagnostics.HasError() {
		return
	}

	// TIP: -- 4. Call the AWS Create function
	out, err := conn.CreateTenant(ctx, &input)
	if err != nil {
		// TIP: Since ID has not been set yet, you cannot use plan.ID.String()
		// in error messages at this point.
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, plan.TenantName.String())
		return
	}
	if out == nil || out.TenantId == nil {
		smerr.AddError(ctx, &resp.Diagnostics, errors.New("empty output"), smerr.ID, plan.TenantName.String())
		return
	}
	smerr.AddEnrich(ctx, &resp.Diagnostics, flex.Flatten(ctx, out, &plan, flex.WithFieldNamePrefix("Tenant"), flex.WithIgnoredFieldNames([]string{"CreatedTimestamp", "Tags"})))
	if resp.Diagnostics.HasError() {
		return
	}
	plan.ID = types.StringValue(aws.ToString(out.TenantId))
	plan.ARN = types.StringValue(aws.ToString(out.TenantArn))
	plan.CreatedTimestamp = types.StringValue(aws.ToTime(out.CreatedTimestamp).Format(time.RFC3339))

	// TIP: -- 7. Save the request plan to response state
	smerr.AddEnrich(ctx, &resp.Diagnostics, resp.State.Set(ctx, &plan))
}

func (r *resourceTenant) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TIP: ==== RESOURCE READ ====
	// Generally, the Read function should do the following things. Make
	// sure there is a good reason if you don't do one of these.
	//
	// 1. Get a client connection to the relevant service
	// 2. Fetch the state
	// 3. Get the resource from AWS
	// 4. Remove resource from state if it is not found
	// 5. Set the arguments and attributes
	// 6. Set the state

	// TIP: -- 1. Get a client connection to the relevant service
	conn := r.Meta().SESV2Client(ctx)

	// TIP: -- 2. Fetch the state
	var state resourceTenantModel
	smerr.AddEnrich(ctx, &resp.Diagnostics, req.State.Get(ctx, &state))
	if resp.Diagnostics.HasError() {
		return
	}

	// TIP: -- 3. Get the resource from AWS using an API Get, List, or Describe-
	// type function, or, better yet, using a finder.
	out, err := FindTenantByName(ctx, conn, state.TenantName.ValueString())

	fmt.Printf("DEBUG :::: FindTenantByName == %v\n", *out.TenantName)
	// TIP: -- 4. Remove resource from state if it is not found
	if tfresource.NotFound(err) {
		resp.Diagnostics.Append(fwdiag.NewResourceNotFoundWarningDiagnostic(err))
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.String())
		return
	}

	state.CreatedTimestamp = types.StringValue(aws.ToTime(out.CreatedTimestamp).Format(time.RFC3339))

	// TIP: -- 5. Set the arguments and attributes
	smerr.AddEnrich(ctx, &resp.Diagnostics, flex.Flatten(ctx, out, &state, flex.WithIgnoredFieldNames([]string{"Tags", "CreatedTimestamp"})))
	if resp.Diagnostics.HasError() {
		return
	}

	// TIP: -- 6. Set the state
	smerr.AddEnrich(ctx, &resp.Diagnostics, resp.State.Set(ctx, &state))
}

//func (r *resourceTenant) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
//	// TIP: ==== RESOURCE UPDATE ====
//	// Not all resources have Update functions. There are a few reasons:
//	// a. The AWS API does not support changing a resource
//	// b. All arguments have RequiresReplace() plan modifiers
//	// c. The AWS API uses a create call to modify an existing resource
//	//
//	// In the cases of a. and b., the resource will not have an update method
//	// defined. In the case of c., Update and Create can be refactored to call
//	// the same underlying function.
//	//
//	// The rest of the time, there should be an Update function and it should
//	// do the following things. Make sure there is a good reason if you don't
//	// do one of these.
//	//
//	// 1. Get a client connection to the relevant service
//	// 2. Fetch the plan and state
//	// 3. Populate a modify input structure and check for changes
//	// 4. Call the AWS modify/update function
//	// 5. Use a waiter to wait for update to complete
//	// 6. Save the request plan to response state
//	// TIP: -- 1. Get a client connection to the relevant service
//	conn := r.Meta().SESV2Client(ctx)
//
//	// TIP: -- 2. Fetch the plan
//	var plan, state resourceTenantModel
//	smerr.EnrichAppend(ctx, &resp.Diagnostics, req.Plan.Get(ctx, &plan))
//	smerr.EnrichAppend(ctx, &resp.Diagnostics, req.State.Get(ctx, &state))
//	if resp.Diagnostics.HasError() {
//		return
//	}
//
//	// TIP: -- 3. Get the difference between the plan and state, if any
//	diff, d := flex.Diff(ctx, plan, state)
//	smerr.EnrichAppend(ctx, &resp.Diagnostics, d)
//	if resp.Diagnostics.HasError() {
//		return
//	}
//
//	if diff.HasChanges() {
//		var input sesv2.UpdateTenantInput
//		smerr.EnrichAppend(ctx, &resp.Diagnostics, flex.Expand(ctx, plan, &input, flex.WithFieldNamePrefix("Test")))
//		if resp.Diagnostics.HasError() {
//			return
//		}
//
//		// TIP: -- 4. Call the AWS modify/update function
//		out, err := conn.UpdateTenant(ctx, &input)
//		if err != nil {
//			smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, plan.ID.String())
//			return
//		}
//		if out == nil || out.Tenant == nil {
//			smerr.AddError(ctx, &resp.Diagnostics, errors.New("empty output"), smerr.ID, plan.ID.String())
//			return
//		}
//
//		// TIP: Using the output from the update function, re-set any computed attributes
//		smerr.EnrichAppend(ctx, &resp.Diagnostics, flex.Flatten(ctx, out, &plan))
//		if resp.Diagnostics.HasError() {
//			return
//		}
//	}
//
//	// TIP: -- 5. Use a waiter to wait for update to complete
//	updateTimeout := r.UpdateTimeout(ctx, plan.Timeouts)
//	_, err := waitTenantUpdated(ctx, conn, plan.ID.ValueString(), updateTimeout)
//	if err != nil {
//		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, plan.ID.String())
//		return
//	}
//
//	// TIP: -- 6. Save the request plan to response state
//	smerr.EnrichAppend(ctx, &resp.Diagnostics, resp.State.Set(ctx, &plan))
//}

func (r *resourceTenant) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TIP: ==== RESOURCE DELETE ====
	// Most resources have Delete functions. There are rare situations
	// where you might not need a delete:
	// a. The AWS API does not provide a way to delete the resource
	// b. The point of your resource is to perform an action (e.g., reboot a
	//    server) and deleting serves no purpose.
	//
	// The Delete function should do the following things. Make sure there
	// is a good reason if you don't do one of these.
	//
	// 1. Get a client connection to the relevant service
	// 2. Fetch the state
	// 3. Populate a delete input structure
	// 4. Call the AWS delete function
	// 5. Use a waiter to wait for delete to complete
	// TIP: -- 1. Get a client connection to the relevant service
	conn := r.Meta().SESV2Client(ctx)

	// TIP: -- 2. Fetch the state
	var state resourceTenantModel
	smerr.EnrichAppend(ctx, &resp.Diagnostics, req.State.Get(ctx, &state))
	if resp.Diagnostics.HasError() {
		return
	}

	// TIP: -- 3. Populate a delete input structure
	input := sesv2.DeleteTenantInput{
		TenantName: state.TenantName.ValueStringPointer(),
	}

	// TIP: -- 4. Call the AWS delete function
	_, err := conn.DeleteTenant(ctx, &input)
	// TIP: On rare occassions, the API returns a not found error after deleting a
	// resource. If that happens, we don't want it to show up as an error.
	if err != nil {
		if errs.IsA[*awstypes.NotFoundException](err) {
			return
		}

		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.String())
		return
	}

	// TIP: -- 5. Use a waiter to wait for delete to complete
	//	deleteTimeout := r.DeleteTimeout(ctx, state.Timeouts)
	//	_, err = waitTenantDeleted(ctx, conn, state.ID.ValueString(), deleteTimeout)
	//	if err != nil {
	//		smerr.AddError(ctx, &resp.Diagnostics, err, smerr.ID, state.ID.String())
	//		return
	//	}
}

// TIP: ==== TERRAFORM IMPORTING ====
// If Read can get all the information it needs from the Identifier
// (i.e., path.Root("id")), you can use the PassthroughID importer. Otherwise,
// you'll need a custom import function.
//
// See more:
// https://developer.hashicorp.com/terraform/plugin/framework/resources/import
func (r *resourceTenant) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root(names.AttrID), req, resp)
}

// TIP: ==== STATUS CONSTANTS ====
// Create constants for states and statuses if the service does not
// already have suitable constants. We prefer that you use the constants
// provided in the service if available (e.g., awstypes.StatusInProgress).
const (
	statusChangePending = "Pending"
	statusDeleting      = "Deleting"
	statusNormal        = "Normal"
	statusUpdated       = "Updated"
)

// TIP: ==== WAITERS ====
// Some resources of some services have waiters provided by the AWS API.
// Unless they do not work properly, use them rather than defining new ones
// here.
//
// Sometimes we define the wait, status, and find functions in separate
// files, wait.go, status.go, and find.go. Follow the pattern set out in the
// service and define these where it makes the most sense.
//
// If these functions are used in the _test.go file, they will need to be
// exported (i.e., capitalized).
//
// You will need to adjust the parameters and names to fit the service.
//func waitTenantCreated(ctx context.Context, conn *sesv2.Client, id string, timeout time.Duration) (*awstypes.Tenant, error) {
//	stateConf := &retry.StateChangeConf{
//		Pending:                   []string{},
//		Target:                    []string{statusNormal},
//		Refresh:                   statusTenant(ctx, conn, id),
//		Timeout:                   timeout,
//		NotFoundChecks:            20,
//		ContinuousTargetOccurence: 2,
//	}
//
//	outputRaw, err := stateConf.WaitForStateContext(ctx)
//	if out, ok := outputRaw.(*sesv2.Tenant); ok {
//		return out, smarterr.NewError(err)
//	}
//
//	return nil, smarterr.NewError(err)
//}

// TIP: It is easier to determine whether a resource is updated for some
// resources than others. The best case is a status flag that tells you when
// the update has been fully realized. Other times, you can check to see if a
// key resource argument is updated to a new value or not.
//func waitTenantUpdated(ctx context.Context, conn *sesv2.Client, id string, timeout time.Duration) (*awstypes.Tenant, error) {
//	stateConf := &retry.StateChangeConf{
//		Pending:                   []string{statusChangePending},
//		Target:                    []string{statusUpdated},
//		Refresh:                   statusTenant(ctx, conn, id),
//		Timeout:                   timeout,
//		NotFoundChecks:            20,
//		ContinuousTargetOccurence: 2,
//	}
//
//	outputRaw, err := stateConf.WaitForStateContext(ctx)
//	if out, ok := outputRaw.(*sesv2.Tenant); ok {
//		return out, smarterr.NewError(err)
//	}
//
//	return nil, smarterr.NewError(err)
//}
//
//// TIP: A deleted waiter is almost like a backwards created waiter. There may
//// be additional pending states, however.
//func waitTenantDeleted(ctx context.Context, conn *sesv2.Client, id string, timeout time.Duration) (*awstypes.Tenant, error) {
//	stateConf := &retry.StateChangeConf{
//		Pending: []string{statusDeleting, statusNormal},
//		Target:  []string{},
//		Refresh: statusTenant(ctx, conn, id),
//		Timeout: timeout,
//	}
//
//	outputRaw, err := stateConf.WaitForStateContext(ctx)
//	if out, ok := outputRaw.(*sesv2.Tenant); ok {
//		return out, smarterr.NewError(err)
//	}
//
//	return nil, smarterr.NewError(err)
//}

// TIP: ==== STATUS ====
// The status function can return an actual status when that field is
// available from the API (e.g., out.Status). Otherwise, you can use custom
// statuses to communicate the states of the resource.
//
// Waiters consume the values returned by status functions. Design status so
// that it can be reused by a create, update, and delete waiter, if possible.
//func statusTenant(ctx context.Context, conn *sesv2.Client, name string) retry.StateRefreshFunc {
//	return func() (any, string, error) {
//		out, err := findTenantByName(ctx, conn, name)
//		if tfresource.NotFound(err) {
//			return nil, "", nil
//		}
//
//		if err != nil {
//			return nil, "", smarterr.NewError(err)
//		}
//
//		return out, aws.ToString(out.Status), nil
//	}
//}

// TIP: ==== FINDERS ====
// The find function is not strictly necessary. You could do the API
// request from the status function. However, we have found that find often
// comes in handy in other places besides the status function. As a result, it
// is good practice to define it separately.
func FindTenantByName(ctx context.Context, conn *sesv2.Client, name string) (*awstypes.Tenant, error) {
	input := sesv2.GetTenantInput{
		TenantName: aws.String(name),
	}

	out, err := conn.GetTenant(ctx, &input)
	if err != nil {
		if errs.IsA[*awstypes.NotFoundException](err) {
			return nil, smarterr.NewError(&retry.NotFoundError{
				LastError:   err,
				LastRequest: &input,
			})
		}

		return nil, smarterr.NewError(err)
	}

	if out == nil || out.Tenant == nil {
		return nil, smarterr.NewError(tfresource.NewEmptyResultError(&input))
	}

	return out.Tenant, nil
}

// TIP: ==== DATA STRUCTURES ====
// With Terraform Plugin-Framework configurations are deserialized into
// Go types, providing type safety without the need for type assertions.
// These structs should match the schema definition exactly, and the `tfsdk`
// tag value should match the attribute name.
//
// Nested objects are represented in their own data struct. These will
// also have a corresponding attribute type mapping for use inside flex
// functions.
//
// See more:
// https://developer.hashicorp.com/terraform/plugin/framework/handling-data/accessing-values
type resourceTenantModel struct {
	framework.WithRegionModel
	ARN              types.String `tfsdk:"arn"`
	CreatedTimestamp types.String `tfsdk:"created_timestamp"`
	ID               types.String `tfsdk:"id"`
	SendingStatus    types.String `tfsdk:"sending_status"`
	Tags             tftags.Map   `tfsdk:"tags"`
	TagsAll          tftags.Map   `tfsdk:"tags_all"`
	TenantName       types.String `tfsdk:"tenant_name"`
}

// TIP: ==== SWEEPERS ====
// When acceptance testing resources, interrupted or failed tests may
// leave behind orphaned resources in an account. To facilitate cleaning
// up lingering resources, each resource implementation should include
// a corresponding "sweeper" function.
//
// The sweeper function lists all resources of a given type and sets the
// appropriate identifers required to delete the resource via the Delete
// method implemented above.
//
// Once the sweeper function is implemented, register it in sweep.go
// as follows:
//
//	awsv2.Register("aws_sesv2_tenant", sweepTenants)
//
// See more:
// https://hashicorp.github.io/terraform-provider-aws/running-and-writing-acceptance-tests/#acceptance-test-sweepers
func sweepTenants(ctx context.Context, client *conns.AWSClient) ([]sweep.Sweepable, error) {
	input := sesv2.ListTenantsInput{}
	conn := client.SESV2Client(ctx)
	var sweepResources []sweep.Sweepable

	pages := sesv2.NewListTenantsPaginator(conn, &input)
	for pages.HasMorePages() {
		page, err := pages.NextPage(ctx)
		if err != nil {
			return nil, smarterr.NewError(err)
		}

		for _, v := range page.Tenants {
			sweepResources = append(sweepResources, sweepfw.NewSweepResource(newResourceTenant, client,
				sweepfw.NewAttribute(names.AttrID, aws.ToString(v.TenantId))),
			)
		}
	}

	return sweepResources, nil
}

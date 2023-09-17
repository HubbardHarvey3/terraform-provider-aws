// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cleanrooms

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cleanrooms"
	"github.com/aws/aws-sdk-go-v2/service/cleanrooms/types"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-aws/internal/conns"
	"github.com/hashicorp/terraform-provider-aws/internal/create"
	"github.com/hashicorp/terraform-provider-aws/internal/flex"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
	"github.com/hashicorp/terraform-provider-aws/internal/tfresource"
	"github.com/hashicorp/terraform-provider-aws/names"
)

// @SDKResource("aws_cleanrooms_configured_table")
// @Tags(identifierAttribute="arn")
func ResourceConfiguredTable() *schema.Resource {
	return &schema.Resource{
		CreateWithoutTimeout: resourceConfiguredTableCreate,
		ReadWithoutTimeout:   resourceConfiguredTableRead,
		UpdateWithoutTimeout: resourceConfiguredTableUpdate,
		DeleteWithoutTimeout: resourceConfiguredTableDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(1 * time.Minute),
			Update: schema.DefaultTimeout(1 * time.Minute),
			Delete: schema.DefaultTimeout(1 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"allowed_columns": {
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				MinItems: 1,
				MaxItems: 225,
			},
			"analysis_method": {
				Type:     schema.TypeString,
				Required: true,
			},
			names.AttrARN: {
				Type:     schema.TypeString,
				Computed: true,
			},
			"create_time": {
				Type:     schema.TypeString,
				Computed: true,
			},
			names.AttrDescription: {
				Type:     schema.TypeString,
				Optional: true,
			},
			names.AttrName: {
				Type:     schema.TypeString,
				Required: true,
			},
			"table_reference": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"database_name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"table_name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},
			names.AttrTags:    tftags.TagsSchema(),
			names.AttrTagsAll: tftags.TagsSchemaComputed(),
			"update_time": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

const (
	ResNameConfiguredTable = "Configured Table"
)

func resourceConfiguredTableCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).CleanRoomsClient(ctx)

	input := &cleanrooms.CreateConfiguredTableInput{
		Name:           aws.String(d.Get(names.AttrName).(string)),
		AllowedColumns: flex.ExpandStringValueSet(d.Get("allowed_columns").(*schema.Set)),
		TableReference: expandTableReference(d.Get("table_reference").([]interface{})),
		Tags:           getTagsIn(ctx),
	}

	analysisMethod, err := expandAnalysisMethod(d.Get("analysis_method").(string))
	if err != nil {
		return create.DiagError(names.CleanRooms, create.ErrActionCreating, ResNameConfiguredTable, d.Get("name").(string), err)
	}
	input.AnalysisMethod = analysisMethod

	if v, ok := d.GetOk(names.AttrDescription); ok {
		input.Description = aws.String(v.(string))
	}

	out, err := conn.CreateConfiguredTable(ctx, input)
	if err != nil {
		return create.DiagError(names.CleanRooms, create.ErrActionCreating, ResNameConfiguredTable, d.Get("name").(string), err)
	}

	if out == nil || out.ConfiguredTable == nil {
		return create.DiagError(names.CleanRooms, create.ErrActionCreating, ResNameCollaboration, d.Get("name").(string), errors.New("empty output"))
	}
	d.SetId(aws.ToString(out.ConfiguredTable.Id))

	return resourceConfiguredTableRead(ctx, d, meta)
}

func resourceConfiguredTableRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).CleanRoomsClient(ctx)

	out, err := findConfiguredTableByID(ctx, conn, d.Id())

	if !d.IsNewResource() && tfresource.NotFound(err) {
		log.Printf("[WARN] Clean Rooms Configured Table (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return create.DiagError(names.CleanRooms, create.ErrActionReading, ResNameConfiguredTable, d.Id(), err)
	}

	configuredTable := out.ConfiguredTable
	d.Set(names.AttrARN, configuredTable.Arn)
	d.Set(names.AttrName, configuredTable.Name)
	d.Set(names.AttrDescription, configuredTable.Description)
	d.Set("allowed_columns", configuredTable.AllowedColumns)
	d.Set("analysis_method", configuredTable.AnalysisMethod)
	d.Set("create_time", configuredTable.CreateTime)

	if err := d.Set("table_reference", flattenTableReference(configuredTable.TableReference)); err != nil {
		return diag.Errorf("setting table_reference: %s", err)
	}

	return nil
}

func resourceConfiguredTableUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var diags diag.Diagnostics
	conn := meta.(*conns.AWSClient).CleanRoomsClient(ctx)

	if d.HasChangesExcept("tags", "tags_all") {
		input := &cleanrooms.UpdateConfiguredTableInput{
			ConfiguredTableIdentifier: aws.String(d.Id()),
		}

		if d.HasChanges(names.AttrDescription) {
			input.Description = aws.String(d.Get(names.AttrDescription).(string))
		}

		if d.HasChanges(names.AttrName) {
			input.Name = aws.String(d.Get(names.AttrName).(string))
		}

		_, err := conn.UpdateConfiguredTable(ctx, input)
		if err != nil {
			return create.DiagError(names.CleanRooms, create.ErrActionUpdating, ResNameConfiguredTable, d.Id(), err)
		}
	}

	return append(diags, resourceConfiguredTableRead(ctx, d, meta)...)
}

func resourceConfiguredTableDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	conn := meta.(*conns.AWSClient).CleanRoomsClient(ctx)

	log.Printf("[INFO] Deleting Clean Rooms Configured Table %s", d.Id())
	in := &cleanrooms.DeleteConfiguredTableInput{
		ConfiguredTableIdentifier: aws.String(d.Id()),
	}

	if _, err := conn.DeleteConfiguredTable(ctx, in); err != nil {
		return create.DiagError(names.CleanRooms, create.ErrActionDeleting, ResNameConfiguredTable, d.Id(), err)
	}

	return nil
}

func findConfiguredTableByID(ctx context.Context, conn *cleanrooms.Client, id string) (*cleanrooms.GetConfiguredTableOutput, error) {
	in := &cleanrooms.GetConfiguredTableInput{
		ConfiguredTableIdentifier: aws.String(id),
	}

	out, err := conn.GetConfiguredTable(ctx, in)
	if err != nil {
		return nil, err
	}

	if out == nil || out.ConfiguredTable == nil {
		return nil, tfresource.NewEmptyResultError(in)
	}

	return out, nil
}

func expandAnalysisMethod(analysisMethod string) (types.AnalysisMethod, error) {
	switch analysisMethod {
	case "DIRECT_QUERY":
		return types.AnalysisMethodDirectQuery, nil
	default:
		return types.AnalysisMethodDirectQuery, fmt.Errorf("Invalid analysis method. The only valid value is currently `DIRECT_QUERY`")
	}
}

func expandTableReference(data []interface{}) types.TableReference {
	tableReference := data[0].(map[string]interface{})
	return &types.TableReferenceMemberGlue{
		Value: types.GlueTableReference{
			DatabaseName: aws.String(tableReference["database_name"].(string)),
			TableName:    aws.String(tableReference["table_name"].(string)),
		},
	}
}

func flattenTableReference(tableReference types.TableReference) []interface{} {
	switch v := tableReference.(type) {
	case *types.TableReferenceMemberGlue:
		m := map[string]interface{}{
			"database_name": v.Value.DatabaseName,
			"table_name":    v.Value.TableName,
		}
		return []interface{}{m}
	default:
		return nil
	}
}

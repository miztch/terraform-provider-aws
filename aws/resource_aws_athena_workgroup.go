package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAthenaWorkgroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAthenaWorkgroupCreate,
		Read:   resourceAwsAthenaWorkgroupRead,
		Update: resourceAwsAthenaWorkgroupUpdate,
		Delete: resourceAwsAthenaWorkgroupDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"bytes_scanned_cutoff_per_query": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtLeast(10485760),
			},
			"enforce_workgroup_configuration": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"publish_cloudwatch_metrics_enable": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"output_location": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"encryption_option": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					athena.EncryptionOptionCseKms,
					athena.EncryptionOptionSseKms,
					athena.EncryptionOptionSseS3,
				}, false),
			},
			"kms_key": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsAthenaWorkgroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).athenaconn

	name := d.Get("name").(string)

	input := &athena.CreateWorkGroupInput{
		Name: aws.String(name),
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	inputConfiguration := &athena.WorkGroupConfiguration{}

	if v, ok := d.GetOk("bytes_scanned_cutoff_per_query"); ok {
		inputConfiguration.BytesScannedCutoffPerQuery = aws.Int64(int64(v.(int)))
	}

	input.Configuration = inputConfiguration

	_, err := conn.CreateWorkGroup(input)

	if err != nil {
		return err
	}

	d.SetId(name)

	return resourceAwsAthenaWorkgroupRead(d, meta)
}

func resourceAwsAthenaWorkgroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).athenaconn

	input := &athena.GetWorkGroupInput{
		WorkGroup: aws.String(d.Id()),
	}

	resp, err := conn.GetWorkGroup(input)

	if err != nil {
		if isAWSErr(err, athena.ErrCodeInvalidRequestException, d.Id()) {
			log.Printf("[WARN] Athena WorkGroup (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", resp.WorkGroup.Name)
	d.Set("description", resp.WorkGroup.Description)

	if resp.WorkGroup.Configuration != nil {
		if resp.WorkGroup.Configuration.BytesScannedCutoffPerQuery != nil {
			d.Set("bytes_scanned_cutoff_per_query", resp.WorkGroup.Configuration.BytesScannedCutoffPerQuery)
		}
	}

	// d.Set("publish_cloudwatch_metrics_enabled", resp.WorkGroup.Configuration.PublishCloudWatchMetricsEnabled)
	// d.Set("enforce_workgroup_configuration", resp.WorkGroup.Configuration.EnforceWorkGroupConfiguration)
	// d.Set("output_location", resp.WorkGroup.Configuration.ResultConfiguration.OutputLocation)
	// d.Set("encryption_option", resp.WorkGroup.Configuration.ResultConfiguration.EncryptionConfiguration.EncryptionOption)
	// d.Set("kms_key", resp.WorkGroup.Configuration.ResultConfiguration.EncryptionConfiguration.KmsKey)

	return nil
}

func resourceAwsAthenaWorkgroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).athenaconn

	input := &athena.DeleteWorkGroupInput{
		WorkGroup: aws.String(d.Id()),
	}

	_, err := conn.DeleteWorkGroup(input)

	return err
}

func resourceAwsAthenaWorkgroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).athenaconn

	workGroupUpdate := false
	configUpdate := false

	input := &athena.UpdateWorkGroupInput{
		WorkGroup: aws.String(d.Get("name").(string)),
	}

	if d.HasChange("description") {
		workGroupUpdate = true
		input.Description = aws.String(d.Get("description").(string))
	}

	inputConfigurationUpdates := &athena.WorkGroupConfigurationUpdates{}

	if d.HasChange("bytes_scanned_cutoff_per_query") {
		workGroupUpdate = true
		configUpdate = true

		if v, ok := d.GetOk("bytes_scanned_cutoff_per_query"); ok {
			inputConfigurationUpdates.BytesScannedCutoffPerQuery = aws.Int64(int64(v.(int)))
		} else {
			inputConfigurationUpdates.RemoveBytesScannedCutoffPerQuery = aws.Bool(true)
		}
	}

	if workGroupUpdate {
		if configUpdate {
			input.ConfigurationUpdates = inputConfigurationUpdates
		}

		_, err := conn.UpdateWorkGroup(input)

		if err != nil {
			return fmt.Errorf("Error updating Athena WorkGroup (%s): %s", d.Id(), err)
		}
	}

	return resourceAwsAthenaWorkgroupRead(d, meta)
}

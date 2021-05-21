package contentful

import (
	"github.com/hashicorp/terraform/helper/schema"
	contentful "github.com/regressivetech/contentful-go"
)

func resourceContentfulAsset() *schema.Resource {
	return &schema.Resource{
		Create: resourceCreateAsset,
		Read:   resourceReadAsset,
		Update: resourceUpdateAsset,
		Delete: resourceDeleteAsset,

		Schema: map[string]*schema.Schema{
			"asset_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"locale": {
				Type:     schema.TypeString,
				Required: true,
			},
			"space_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"fields": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"title": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"content": {
										Type:     schema.TypeString,
										Required: true,
									},
									"locale": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"description": {
							Type:     schema.TypeList,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"content": {
										Type:     schema.TypeString,
										Required: true,
									},
									"locale": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
						"file": {
							Type:     schema.TypeMap,
							Required: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"url": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"upload": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"details": {
										Type:     schema.TypeSet,
										Optional: true,
										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"size": {
													Type:     schema.TypeInt,
													Required: true,
												},
												"image": {
													Type:     schema.TypeSet,
													Required: true,
													Elem: &schema.Resource{
														Schema: map[string]*schema.Schema{
															"width": {
																Type:     schema.TypeInt,
																Required: true,
															},
															"height": {
																Type:     schema.TypeInt,
																Required: true,
															},
														},
													},
												},
											},
										},
									},
									"uploadFrom": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"fileName": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"contentType": {
										Type:     schema.TypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			"published": {
				Type:     schema.TypeBool,
				Required: true,
			},
			"archived": {
				Type:     schema.TypeBool,
				Required: true,
			},
		},
	}
}

func resourceCreateAsset(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Client)

	fields := d.Get("fields").([]interface{})[0].(map[string]interface{})

	localizedTitle := map[string]string{}
	rawTitle := fields["title"].([]interface{})
	for i := 0; i < len(rawTitle); i++ {
		field := rawTitle[i].(map[string]interface{})
		localizedTitle[field["locale"].(string)] = field["content"].(string)
	}

	localizedDescription := map[string]string{}
	rawDescription := fields["description"].([]interface{})
	for i := 0; i < len(rawDescription); i++ {
		field := rawDescription[i].(map[string]interface{})
		localizedDescription[field["locale"].(string)] = field["content"].(string)
	}

	file := fields["file"].(map[string]interface{})

	asset := &contentful.Asset{
		Sys: &contentful.Sys{
			ID:      d.Get("asset_id").(string),
			Version: 0,
		},
		Locale: d.Get("locale").(string),
		Fields: &contentful.AssetFields{
			Title:       localizedTitle,
			Description: localizedDescription,
			File: map[string]*contentful.File{
				d.Get("locale").(string): {
					FileName:    file["fileName"].(string),
					ContentType: file["contentType"].(string),
				},
			},
		},
	}

	if url, ok := file["url"].(string); ok {
		asset.Fields.File[d.Get("locale").(string)].URL = url
	}

	if upload, ok := file["upload"].(string); ok {
		asset.Fields.File[d.Get("locale").(string)].UploadURL = upload
	}

	if details, ok := file["fileDetails"].(*contentful.FileDetails); ok {
		asset.Fields.File[d.Get("locale").(string)].Details = details
	}

	if uploadFrom, ok := file["uploadFrom"].(string); ok {
		asset.Fields.File[d.Get("locale").(string)].UploadFrom.Sys.ID = uploadFrom
	}

	err = client.Assets.Upsert(d.Get("space_id").(string), asset)
	if err != nil {
		return err
	}

	err = client.Assets.Process(d.Get("space_id").(string), asset)
	if err != nil {
		return err
	}

	d.SetId(asset.Sys.ID)

	if err := setAssetProperties(d, asset); err != nil {
		return err
	}

	err = setAssetState(d, m)
	if err != nil {
		return err
	}

	return err
}

func resourceUpdateAsset(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	assetID := d.Id()

	asset, err := client.Assets.Get(spaceID, assetID)
	if err != nil {
		return err
	}

	fields := d.Get("fields").([]interface{})[0].(map[string]interface{})

	localizedTitle := map[string]string{}
	rawTitle := fields["title"].([]interface{})
	for i := 0; i < len(rawTitle); i++ {
		field := rawTitle[i].(map[string]interface{})
		localizedTitle[field["locale"].(string)] = field["content"].(string)
	}

	localizedDescription := map[string]string{}
	rawDescription := fields["description"].([]interface{})
	for i := 0; i < len(rawDescription); i++ {
		field := rawDescription[i].(map[string]interface{})
		localizedDescription[field["locale"].(string)] = field["content"].(string)
	}

	file := fields["file"].(map[string]interface{})

	asset = &contentful.Asset{
		Sys: &contentful.Sys{
			ID:      d.Get("asset_id").(string),
			Version: d.Get("version").(int),
		},
		Locale: d.Get("locale").(string),
		Fields: &contentful.AssetFields{
			Title:       localizedTitle,
			Description: localizedDescription,
			File: map[string]*contentful.File{
				d.Get("locale").(string): {
					FileName:    file["fileName"].(string),
					ContentType: file["contentType"].(string),
				},
			},
		},
	}

	if url, ok := file["url"].(string); ok {
		asset.Fields.File[d.Get("locale").(string)].URL = url
	}

	if upload, ok := file["upload"].(string); ok {
		asset.Fields.File[d.Get("locale").(string)].UploadURL = upload
	}

	if details, ok := file["fileDetails"].(*contentful.FileDetails); ok {
		asset.Fields.File[d.Get("locale").(string)].Details = details
	}

	if uploadFrom, ok := file["uploadFrom"].(string); ok {
		asset.Fields.File[d.Get("locale").(string)].UploadFrom.Sys.ID = uploadFrom
	}

	err = client.Assets.Upsert(d.Get("space_id").(string), asset)
	if err != nil {
		return err
	}

	err = client.Assets.Process(d.Get("space_id").(string), asset)
	if err != nil {
		return err
	}

	d.SetId(asset.Sys.ID)

	if err := setAssetProperties(d, asset); err != nil {
		return err
	}

	err = setAssetState(d, m)
	if err != nil {
		return err
	}

	return err
}

func setAssetState(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	assetID := d.Id()

	asset, _ := client.Assets.Get(spaceID, assetID)

	if d.Get("published").(bool) && asset.Sys.PublishedAt == "" {
		err = client.Assets.Publish(spaceID, asset)
	} else if !d.Get("published").(bool) && asset.Sys.PublishedAt != "" {
		err = client.Assets.Unpublish(spaceID, asset)
	}

	if d.Get("archived").(bool) && asset.Sys.ArchivedAt == "" {
		err = client.Assets.Archive(spaceID, asset)
	} else if !d.Get("archived").(bool) && asset.Sys.ArchivedAt != "" {
		err = client.Assets.Unarchive(spaceID, asset)
	}

	err = setAssetProperties(d, asset)

	return err
}

func resourceReadAsset(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	assetID := d.Id()

	asset, err := client.Assets.Get(spaceID, assetID)
	if _, ok := err.(contentful.NotFoundError); ok {
		d.SetId("")
		return nil
	}

	return setAssetProperties(d, asset)
}

func resourceDeleteAsset(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	assetID := d.Id()

	asset, err := client.Assets.Get(spaceID, assetID)
	if err != nil {
		return err
	}

	return client.Assets.Delete(spaceID, asset)
}

func setAssetProperties(d *schema.ResourceData, asset *contentful.Asset) (err error) {
	if err = d.Set("space_id", asset.Sys.Space.Sys.ID); err != nil {
		return err
	}

	if err = d.Set("version", asset.Sys.Version); err != nil {
		return err
	}

	return err
}

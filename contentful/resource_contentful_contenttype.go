package contentful

import (
	"errors"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	contentful "github.com/regressivetech/contentful-go"
)

func resourceContentfulContentType() *schema.Resource {
	return &schema.Resource{
		Create: resourceContentTypeCreate,
		Read:   resourceContentTypeRead,
		Update: resourceContentTypeUpdate,
		Delete: resourceContentTypeDelete,

		Schema: map[string]*schema.Schema{
			"space_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"display_field": {
				Type:     schema.TypeString,
				Required: true,
			},
			"content_type_id": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"env_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"field": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
						"link_type": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"items": {
							Type:     schema.TypeList,
							Optional: true,
							MaxItems: 1,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Required: true,
									},
									"link_type": {
										Type:     schema.TypeString,
										Optional: true,
									},
									"validations": {
										Type:     schema.TypeList,
										Optional: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
						"required": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"localized": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"disabled": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"omitted": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
						"validations": {
							Type:     schema.TypeList,
							Optional: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func resourceContentTypeCreate(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	envID := d.Get("env_id").(string)

	ct := &contentful.ContentType{
		Name:         d.Get("name").(string),
		DisplayField: d.Get("display_field").(string),
		Fields:       []*contentful.Field{},
		Sys: &contentful.Sys{
			ID: d.Get("content_type_id").(string),
		},
	}

	id := d.Get("content_type_id")

	if id != nil {
		ct.Sys = &contentful.Sys{
			ID: id.(string),
		}
	}

	env, err := client.Environments.Get(spaceID, envID)
	if err != nil {
		return err
	}

	if env == nil {
		return errors.New("Unable to get environment " + envID)
	}

	if description, ok := d.GetOk("description"); ok {
		ct.Description = description.(string)
	}

	rawField := d.Get("field").([]interface{})
	for i := 0; i < len(rawField); i++ {
		field := rawField[i].(map[string]interface{})

		contentfulField := &contentful.Field{
			ID:        field["id"].(string),
			Name:      field["name"].(string),
			Type:      field["type"].(string),
			Localized: field["localized"].(bool),
			Required:  field["required"].(bool),
			Disabled:  field["disabled"].(bool),
			Omitted:   field["omitted"].(bool),
		}

		if linkType, ok := field["link_type"].(string); ok {
			contentfulField.LinkType = linkType
		}

		if validations, ok := field["validations"].([]interface{}); ok {
			parsedValidations, err := contentful.ParseValidations(validations)
			if err != nil {
				return err
			}

			contentfulField.Validations = parsedValidations
		}

		if items := processItems(field["items"].([]interface{})); items != nil {
			contentfulField.Items = items
		}

		ct.Fields = append(ct.Fields, contentfulField)
	}

	if err = upsertAndActivate(client, env, ct); err != nil {
		return err
	}

	if err = setContentTypeProperties(d, ct); err != nil {
		return err
	}

	d.SetId(ct.Sys.ID)

	return nil
}

func resourceContentTypeRead(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	envID := d.Get("env_id").(string)
	env, err := client.Environments.Get(spaceID, envID)
	if err != nil {
		return err
	}
	_, err = client.ContentTypes.Get(env, d.Id())
	return err
}

func resourceContentTypeUpdate(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	envID := d.Get("env_id").(string)

	env, err := client.Environments.Get(spaceID, envID)
	if err != nil {
		return err
	}

	ct, err := client.ContentTypes.Get(env, d.Id())
	if err != nil {
		return err
	}

	ct.Name = d.Get("name").(string)
	ct.DisplayField = d.Get("display_field").(string)

	if description, ok := d.GetOk("description"); ok {
		ct.Description = description.(string)
	}

	if d.HasChange("field") {
		old, nw := d.GetChange("field")

		firstApplyFields, secondApplyFields, shouldSecondApply := checkFieldsToOmit(old.([]interface{}), nw.([]interface{}))

		ct.Fields = firstApplyFields
		// To remove a field from a content type 4 API calls need to be made.
		// Omit the removed fields and publish the new version of the content type,
		// followed by the field removal and final publish.
		if err = upsertAndActivate(client, env, ct); err != nil {
			return err
		}

		if shouldSecondApply {
			ct.Fields = secondApplyFields
			if err = upsertAndActivate(client, env, ct); err != nil {
				return err
			}
		}
	}

	ct.Fields = newFields(d.Get("field").([]interface{}))
	if err = upsertAndActivate(client, env, ct); err != nil {
		return err
	}

	return setContentTypeProperties(d, ct)
}

func upsertAndActivate(client *contentful.Client, env *contentful.Environment, ct *contentful.ContentType) error {
	if err := client.ContentTypes.Upsert(env, ct); err != nil {
		return err
	}

	if err := client.ContentTypes.Activate(env, ct); err != nil {
		return err
	}
	return nil
}

func resourceContentTypeDelete(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Client)
	spaceID := d.Get("space_id").(string)
	envID := d.Get("env_id").(string)

	env, err := client.Environments.Get(spaceID, envID)

	ct, err := client.ContentTypes.Get(env, d.Id())
	if err != nil {
		return err
	}

	if err = client.ContentTypes.Deactivate(env, ct); err != nil {
		return err
	}

	if err = client.ContentTypes.Delete(env, ct); err != nil {
		return err
	}

	return nil
}

func setContentTypeProperties(d *schema.ResourceData, ct *contentful.ContentType) (err error) {
	if err = d.Set("version", ct.Sys.Version); err != nil {
		return err
	}

	return nil
}

// Contentful API should omit the field.
// And if user want to change field type, user should delete the field completely before user create new field type field.
func checkFieldsToOmit(oldFields, newFields []interface{}) (firstApplyFields, secondApplyFields []*contentful.Field, shouldSecondApply bool) {
	getFieldFromID := func(fields []interface{}, id string) (map[string]interface{}, bool) {
		for _, field := range fields {
			castedField := field.(map[string]interface{})
			if castedField["id"].(string) == id {
				return castedField, true
			}
		}
		return nil, false
	}

	for i := 0; i < len(oldFields); i++ {
		oldField := oldFields[i].(map[string]interface{})

		newField, ok := getFieldFromID(newFields, oldField["id"].(string))

		toOmitted := false
		if !ok {
			// field was deleted
			toOmitted = true
		} else {
			if oldField["type"].(string) != newField["type"].(string) {
				// field type is changed
				toOmitted = true
			}
		}

		shouldDelete := false
		if ok {
			// if field type is changed, should delete field completely
			if oldField["type"].(string) != newField["type"].(string) {
				shouldDelete = true
			}
		}

		field := &contentful.Field{
			ID:        oldField["id"].(string),
			Name:      oldField["name"].(string),
			Type:      oldField["type"].(string),
			LinkType:  oldField["link_type"].(string),
			Localized: oldField["localized"].(bool),
			Required:  oldField["required"].(bool),
			Disabled:  oldField["disabled"].(bool),
			Omitted:   oldField["omitted"].(bool),
		}
		if toOmitted {
			field.Omitted = true
		}

		firstApplyFields = append(firstApplyFields, field)
		if !shouldDelete {
			secondApplyFields = append(secondApplyFields, field)
		} else {
			shouldSecondApply = true
		}
	}
	return
}

func newFields(newFields []interface{}) []*contentful.Field {
	result := make([]*contentful.Field, len(newFields))
	for i := 0; i < len(newFields); i++ {
		newField := newFields[i].(map[string]interface{})

		contentfulField := &contentful.Field{
			ID:        newField["id"].(string),
			Name:      newField["name"].(string),
			Type:      newField["type"].(string),
			Localized: newField["localized"].(bool),
			Required:  newField["required"].(bool),
			Disabled:  newField["disabled"].(bool),
			Omitted:   newField["omitted"].(bool),
		}

		if linkType, ok := newField["link_type"].(string); ok {
			contentfulField.LinkType = linkType
		}

		if validations, ok := newField["validations"].([]interface{}); ok {
			parsedValidations, _ := contentful.ParseValidations(validations)

			contentfulField.Validations = parsedValidations
		}

		if items := processItems(newField["items"].([]interface{})); items != nil {
			contentfulField.Items = items
		}

		result[i] = contentfulField
	}
	return result
}

func processItems(fieldItems []interface{}) *contentful.FieldTypeArrayItem {
	var items *contentful.FieldTypeArrayItem

	for i := 0; i < len(fieldItems); i++ {
		item := fieldItems[i].(map[string]interface{})

		var validations []contentful.FieldValidation

		if fieldValidations, ok := item["validations"].([]interface{}); ok {
			validations, _ = contentful.ParseValidations(fieldValidations)
		}

		items = &contentful.FieldTypeArrayItem{
			Type:        item["type"].(string),
			Validations: validations,
			LinkType:    item["link_type"].(string),
		}
	}
	return items
}

package contentful

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
	contentful "github.com/labd/contentful-go"
)

func TestAccContentfulContentType_Basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckContentfulContentTypeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccContentfulContentTypeConfig,
				Check: resource.TestCheckResourceAttr(
					"contentful_contenttype.mycontenttype", "name", "tf_test1"),
			},
			{
				Config: testAccContentfulContentTypeUpdateConfig,
				Check: resource.TestCheckResourceAttr(
					"contentful_contenttype.mycontenttype", "name", "tf_test1"),
			},
			{
				Config: testAccContentfulContentTypeLinkConfig,
				Check: resource.TestCheckResourceAttr(
					"contentful_contenttype.mylinked_contenttype", "name", "tf_linked"),
			},
			{
				Config: testAccContentfulContentTypeUpdateConfig,
				Check: resource.TestCheckResourceAttr(
					"contentful_configtype.content_type_with_id", "id", "contentTypeWithID"),
			},
		},
	})
}

// noinspection GoUnusedFunction
func testAccCheckContentfulContentTypeExists(n string, contentType *contentful.ContentType) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no content type ID is set")
		}

		spaceID := rs.Primary.Attributes["space_id"]
		if spaceID == "" {
			return fmt.Errorf("no space_id is set")
		}

		client := testAccProvider.Meta().(*contentful.Client)

		ct, err := client.ContentTypes.Get(spaceID, rs.Primary.ID)
		if err != nil {
			return err
		}

		*contentType = *ct

		return nil
	}
}

func testAccCheckContentfulContentTypeDestroy(s *terraform.State) (err error) {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "contentful_contenttype" {
			continue
		}

		spaceID := rs.Primary.Attributes["space_id"]
		if spaceID == "" {
			return fmt.Errorf("no space_id is set")
		}

		client := testAccProvider.Meta().(*contentful.Client)

		_, err := client.ContentTypes.Get(spaceID, rs.Primary.ID)
		if _, ok := err.(contentful.NotFoundError); ok {
			return nil
		}

		return fmt.Errorf("content type still exists with id: %s", rs.Primary.ID)
	}

	return nil
}

var testAccContentfulContentTypeConfig = `
resource "contentful_contenttype" "mycontenttype" {
  space_id = "` + spaceID + `"
  name = "tf_test1"
  description = "Terraform Acc Test Content Type"
  display_field = "field1"
  field {
	disabled  = false
	id        = "field1"
	localized = false
	name      = "Field 1"
	omitted   = false
	required  = true
	type      = "Text"
  }
  field {
	disabled  = false
	id        = "field2"
	localized = false
	name      = "Field 2"
	omitted   = false
	required  = true
	type      = "Integer"
  }
}
`

var testAccContentfulContentTypeUpdateConfig = `
resource "contentful_contenttype" "mycontenttype" {
  space_id = "` + spaceID + `"
  name = "tf_test1"
  description = "Terraform Acc Test Content Type description change"
  display_field = "field1"
  field {
	disabled  = false
	id        = "field1"
	localized = false
	name      = "Field 1 name change"
	omitted   = false
	required  = true
	type      = "Text"
  }
  field {
	disabled  = false
	id        = "field3"
	localized = false
	name      = "Field 3 new field"
	omitted   = false
	required  = true
	type      = "Integer"
  }	
}
`

var testAccContentfulContentTypeLinkConfig = `
resource "contentful_contenttype" "mycontenttype" {
  space_id = "` + spaceID + `"
  name = "tf_test1"
  description = "Terraform Acc Test Content Type description change"
  display_field = "field1"
  field {
	disabled  = false
	id        = "field1"
	localized = false
	name      = "Field 1 name change"
	omitted   = false
	required  = true
	type      = "Text"
  }
  field {
	disabled  = false
	id        = "field3"
	localized = false
	name      = "Field 3 new field"
	omitted   = false
	required  = true
	type      = "Integer"
  }	
}

resource "contentful_contenttype" "mylinked_contenttype" {
  space_id = "` + spaceID + `"
  name          = "tf_linked"
  description   = "Terraform Acc Test Content Type with links"
  display_field = "asset_field"
  field {
    id   = "asset_field"
    name = "Asset Field"
    type = "Array"
    items {
      type      = "Link"
      link_type = "Asset"
    }
    required = true
  }
  field {
    id        = "entry_link_field"
    name      = "Entry Link Field"
    type      = "Link"
    link_type = "Entry"
    validations = [
	  jsonencode({
		linkContentType = [
          contentful_contenttype.mycontenttype.id
		]
	  })
	]
    required = false
  }
}

resource "contentful_contenttype" "content_type_with_id" {
  space_id = "` + spaceID + `"
  name = "tf_test_with_id"
  description = "Content Type with ID"
  display_field = "fieldWithID"
	content_type_id = "contentTypeWithID"
  field {
	disabled  = false
	id        = "field1"
	localized = false
	name      = "Field 1 name"
	omitted   = false
	required  = true
	type      = "Text"
  }
  field {
	disabled  = false
	id        = "field2"
	localized = false
	name      = "Field 2 name"
	omitted   = false
	required  = true
	type      = "Integer"
  }	
}
`

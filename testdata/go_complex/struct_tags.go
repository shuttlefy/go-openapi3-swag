package complex

type CreateUserRequest struct {
	Owner   string  `json:"owner" binding:"required" example:"brayden"`
	Remark  *string `json:"remark" example:"some remark"`
	Name    string  `json:"name" validate:"required" example:"John"`
	Email   string  `json:"email" binding:"required,email" example:"john@example.com"`
	Age     int     `json:"age,omitempty" example:"25"`
	Hidden  string  `json:"-"`
	NoTag   string
	XMLOnly string `xml:"xmlField"`
}

type UpdateUserRequest struct {
	Name    *string `json:"name,omitempty" validate:"required,min=1"`
	Status  string  `json:"status" binding:"required,oneof=active inactive" enums:"active,inactive" example:"active"`
	Profile string  `json:"profile" validate:"required"`
}

type OrderRequest struct {
	State  string `json:"state" enums:"passed,rejected,terminated,cancelled,pending" example:"pending"`
	Type   int    `json:"type" enums:"1,2,3"`
	NoEnum string `json:"no_enum"`
}

type ProductDetail struct {
	Name       string   `json:"name" description:"Product name" minLength:"1" maxLength:"200" example:"Widget"`
	Price      float64  `json:"price" minimum:"0" maximum:"99999.99" default:"0" format:"double"`
	SKU        string   `json:"sku" pattern:"^[A-Z]{2}-\\d{6}$" example:"AB-123456"`
	Quantity   int      `json:"quantity" minimum:"0" maximum:"10000" default:"1"`
	Tags       []string `json:"tags" minItems:"1" maxItems:"10" uniqueItems:"true"`
	InternalID string   `json:"internal_id" readonly:"true" format:"uuid"`
	Password   string   `json:"password" writeonly:"true" minLength:"8"`
	OldField   string   `json:"old_field" deprecated:"true"`
	CreatedAt  string   `json:"created_at" format:"date-time" readonly:"true"`
	Email      string   `json:"email" format:"email" description:"Contact email"`
	Score      float64  `json:"score" minimum:"-1.5" maximum:"1.5"`
}

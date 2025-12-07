// Package form provides type-safe form handling with validation for Vango applications.
//
// # Overview
//
// The form package provides a generic Form[T] type that binds to Go structs,
// offering declarative validation through struct tags and built-in validators.
//
// # Basic Usage
//
//	type ContactForm struct {
//	    Name    string `form:"name" validate:"required,min=2,max=100"`
//	    Email   string `form:"email" validate:"required,email"`
//	    Message string `form:"message" validate:"required,max=1000"`
//	}
//
//	func ContactPage() vango.Component {
//	    return vango.Func(func() *vango.VNode {
//	        form := UseForm(ContactForm{})
//
//	        submit := func() {
//	            if !form.Validate() {
//	                return // Errors displayed automatically
//	            }
//	            data := form.Values()
//	            sendEmail(data.Name, data.Email, data.Message)
//	            form.Reset()
//	        }
//
//	        return Form(OnSubmit(submit),
//	            form.Field("Name", Input(Type("text"))),
//	            form.Field("Email", Input(Type("email"))),
//	            form.Field("Message", Textarea()),
//	            Button(Type("submit"), Text("Send")),
//	        )
//	    })
//	}
//
// # Validation
//
// The package includes built-in validators for common patterns:
//
//   - Required: Non-empty value
//   - MinLength/MaxLength: String length constraints
//   - Email: Valid email format
//   - Pattern: Regular expression matching
//   - Min/Max: Numeric range constraints
//   - Custom: User-defined validation logic
//
// # Form Arrays
//
// For forms with dynamic arrays of nested objects, use the Array method:
//
//	form.Array("Items", func(item FormArrayItem, i int) *vango.VNode {
//	    return Div(
//	        item.Field("Name", Input(Type("text"))),
//	        Button(OnClick(item.Remove), Text("Remove")),
//	    )
//	})
package form

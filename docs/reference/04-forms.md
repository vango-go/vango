# Forms Reference

## Basic Forms

Bind inputs to signals:

```go
func LoginForm() vango.Component {
    return vango.Func(func() *vango.VNode {
        email := vango.Signal("")
        password := vango.Signal("")

        submit := func() {
            auth.Login(email(), password())
        }

        return Form(OnSubmit(submit),
            Input(Type("email"), Value(email()), OnInput(email.Set)),
            Input(Type("password"), Value(password()), OnInput(password.Set)),
            Button(Type("submit"), Text("Login")),
        )
    })
}
```

## UseForm Hook

For complex forms with validation:

```go
type ContactInput struct {
    Name    string `form:"name"`
    Email   string `form:"email"`
    Message string `form:"message"`
}

func ContactForm() vango.Component {
    return vango.Func(func() *vango.VNode {
        form := vango.UseForm(ContactInput{})

        submit := func() {
            if !form.Validate() {
                return
            }
            sendEmail(form.Values())
        }

        return Form(OnSubmit(submit),
            form.Field("Name",
                Input(Type("text")),
                vango.Required("Name is required"),
            ),
            form.Field("Email",
                Input(Type("email")),
                vango.Required("Email is required"),
                vango.Email("Invalid email"),
            ),
            Button(Type("submit"), Disabled(form.Submitting()), Text("Send")),
        )
    })
}
```

## Form API

```go
form := vango.UseForm(initialValues)

form.Field(name, element, ...validators)  // Create field
form.Validate() bool                      // Run validation
form.Values() T                           // Get current values
form.Reset()                              // Reset to initial
form.Submitting() bool                    // Is submitting?
form.Errors() map[string]string           // Validation errors
```

## Validators

```go
vango.Required("Field is required")
vango.MinLength(3, "Too short")
vango.MaxLength(100, "Too long")
vango.Email("Invalid email")
vango.Pattern(`^\d+$`, "Numbers only")
vango.Custom(func(v string) error { ... })
```
